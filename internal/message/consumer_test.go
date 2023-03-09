package message

import (
	"context"
	"github.com/ClarabridgeInc/ingestion-callback/internal/callback"
	"github.com/ClarabridgeInc/ingestion-callback/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type MockReader struct {
}

func (m *MockReader) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(options *s3.Options)) (
	*s3.GetObjectOutput, error,
) {
	if *params.Key == "test-key" {
		protofile, err := os.ReadFile("fixtures/test.pb")
		if err != nil {
			panic("could not read test fixture: fixtures/test.pb")
		}
		return &s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader(string(protofile))),
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
				MessageId:     aws.String("test-message-id"),
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

type MockHttpClient struct {
}

func (m *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "SUCCESS",
		StatusCode: 200,
	}, nil
}

func TestConsumer(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Error("could not create logger")
		t.Fail()
	}

	cfg := Config{
		Logger:          logger,
		Queue:           Queue{Name: "test-queue"},
		ReceiverDeleter: &MockSQSClient{},
		S3Reader: storage.S3Reader{
			Bucket: "test-bucket",
			Reader: &MockReader{},
		},
		Executor: callback.Executor{
			HTTPClient: &MockHttpClient{},
			Logger:     logger,
		},
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
		MessageId:     aws.String("test-message-id"),
		ReceiptHandle: aws.String("test-receipt-handle"),
	}
	err = consumer.ProcessMessage(ctx, testMessage)
	assert.NoError(t, err)
}
