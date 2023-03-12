package sqsconsumer

import (
	"bytes"
	"context"
	"github.com/ClarabridgeInc/ingestion-callback/internal/callback"
	"github.com/ClarabridgeInc/ingestion-callback/internal/pb/services/ingest"
	"github.com/ClarabridgeInc/ingestion-callback/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type MockReader struct {
	MockServerUrl string
}

func (m *MockReader) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(options *s3.Options)) (
	*s3.GetObjectOutput, error,
) {
	if *params.Key == "test-key" {
		testDocument := &ingest.IngestDocument{
			Version:                0,
			Uuid:                   "test-uuid",
			RoutingKey:             "test-routing",
			NaturalId:              "test-natural-id",
			Source:                 "call",
			DocumentDate:           "test-date",
			LanguageId:             "en",
			Attributes:             nil,
			Verbatims:              nil,
			ImportApiRequest:       "",
			DuplicateDetectionType: 0,
			ClassificationResult:   nil,
			DerivedAttributes:      nil,
			Topology: &ingest.Topology{
				Name: "agent_assist",
				Configuration: map[string]string{
					"callback_url": m.MockServerUrl,
				},
			},
		}

		byteMessage, err := proto.Marshal(testDocument)
		if err != nil {
			return nil, err
		}

		return &s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader(byteMessage)),
		}, nil
	}
	return nil, errors.New("key does not exist")
}

type MockSQSClient struct {
}

func (s *MockSQSClient) GetQueueUrl(
	ctx context.Context,
	params *sqs.GetQueueUrlInput,
	optFuncs ...func(options *sqs.Options),
) (*sqs.GetQueueUrlOutput, error) {
	return &sqs.GetQueueUrlOutput{
		QueueUrl:       aws.String("test-queue"),
		ResultMetadata: middleware.Metadata{},
	}, nil
}

func (s *MockSQSClient) ReceiveMessage(
	ctx context.Context,
	params *sqs.ReceiveMessageInput,
	optFuncs ...func(*sqs.Options),
) (*sqs.ReceiveMessageOutput, error) {
	notificationJson, err := os.ReadFile("fixtures/test_sqs_notification.json")
	if err != nil {
		panic("could not read test fixture: fixtures/test_sqs_notification.json")
	}
	return &sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{
				Body:          aws.String(string(notificationJson)),
				MessageId:     aws.String("test-sqsconsumer-id"),
				ReceiptHandle: aws.String("test-receipt-handle"),
			},
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}

func (s *MockSQSClient) DeleteMessage(
	c context.Context,
	input *sqs.DeleteMessageInput,
	optFuncs ...func(*sqs.Options),
) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

func TestConsumer(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Error("could not create logger")
		t.Fail()
	}

	mockServer := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/callback" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("string"))
				} else {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(""))
				}
			},
		),
	)
	defer mockServer.Close()

	cfg := Config{
		Logger:          logger,
		Queue:           Queue{Name: "test-queue"},
		ReceiverDeleter: &MockSQSClient{},
		S3Reader: storage.S3Reader{
			Bucket: "test-bucket",
			Reader: &MockReader{
				MockServerUrl: mockServer.URL + "/callback",
			},
		},
		Executor: callback.NewCallbackExecutor(callback.Config{Timeout: 5 * time.Second}),
	}

	ctx := context.TODO()
	consumer, err := NewConsumer(ctx, cfg)
	if err != nil {
		t.Error("could not create consumer for tests")
		t.Fail()
	}

	notificationJson, err := os.ReadFile("fixtures/test_sqs_notification.json")
	if err != nil {
		t.Error("could not read test fixture: fixtures/test_sqs_notification.json")
		t.Fail()
	}
	testMessage := types.Message{
		Body:          aws.String(string(notificationJson)),
		MessageId:     aws.String("test-sqsconsumer-id"),
		ReceiptHandle: aws.String("test-receipt-handle"),
	}
	err = consumer.ProcessMessage(ctx, testMessage)
	assert.NoError(t, err)
}
