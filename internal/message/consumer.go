package message

import (
	"context"
	"encoding/json"
	"github.com/ClarabridgeInc/ingestion-callback/internal/callback"
	"github.com/ClarabridgeInc/ingestion-callback/internal/pb/services/ingest"
	"github.com/ClarabridgeInc/ingestion-callback/internal/storage"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"io"
	"strings"
	"time"
)

type Consumer struct {
	*zap.Logger
	sqsClient ReceiverDeleter
	s3Reader  storage.S3Reader
	queueName string
	queueUrl  *string
	callback.Executor
}

type Queue struct {
	Name string `yaml:"queue_name"`
}

type Config struct {
	*zap.Logger
	Queue
	ReceiverDeleter
	storage.S3Reader
	callback.Executor
}

type sqsNotification struct {
	Records []record `json:"Records"`
}

type record struct {
	EventName string     `json:"eventName"`
	Metadata  s3Metadata `json:"s3"`
}

type s3Metadata struct {
	Object s3MetadataObject `json:"object"`
}

type s3MetadataObject struct {
	Key string `json:"key"`
}

type ReceiverAPI interface {
	GetQueueUrl(
		ctx context.Context,
		params *sqs.GetQueueUrlInput,
		optFuncs ...func(options *sqs.Options),
	) (*sqs.GetQueueUrlOutput, error)
	ReceiveMessage(
		ctx context.Context,
		params *sqs.ReceiveMessageInput,
		optFuncs ...func(*sqs.Options),
	) (*sqs.ReceiveMessageOutput, error)
}

type DeleterAPI interface {
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (
		*sqs.DeleteMessageOutput, error,
	)
}

type ReceiverDeleter interface {
	ReceiverAPI
	DeleterAPI
}

func GetQueueUrl(c context.Context, api ReceiverAPI, input *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return api.GetQueueUrl(c, input)
}

func GetMessages(c context.Context, api ReceiverAPI, input *sqs.ReceiveMessageInput) (
	*sqs.
		ReceiveMessageOutput, error,
) {
	return api.ReceiveMessage(c, input)
}

func DeleteMessage(c context.Context, api DeleterAPI, input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return api.DeleteMessage(c, input)
}

func (n *Consumer) Consume(ctx context.Context) {
	getQueueInput := &sqs.GetQueueUrlInput{
		QueueName: &n.queueName,
	}

	queueUrlResult, err := GetQueueUrl(ctx, n.sqsClient, getQueueInput)
	if err != nil {
		n.Error("got an error while getting queue url:", zap.Error(err))
		return
	}

	queueUrl := queueUrlResult.QueueUrl

	getMessageInput := &sqs.ReceiveMessageInput{
		QueueUrl:            queueUrl,
		MaxNumberOfMessages: 10,
	}

	msgResult, err := GetMessages(ctx, n.sqsClient, getMessageInput)
	if err != nil {
		n.Error("got an error while receiving messages:", zap.Error(err))
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msgResult, err = GetMessages(ctx, n.sqsClient, getMessageInput)
			if err != nil {
				n.Error("got an error while receiving messages:", zap.Error(err))
				return
			}
			for _, v := range msgResult.Messages {
				err = n.ProcessMessage(ctx, v)
				if err != nil {
					n.Logger.Error("error processing message: ", zap.String("error_message_id", *v.MessageId))
				} else {
					err = n.deleteMessage(ctx, v.ReceiptHandle)
					if err != nil {
						n.Logger.Error(
							"could not delete message: ", zap.String("failed_delete_message_id", *v.MessageId),
						)
					}
				}
			}
		}
	}
}

func (n *Consumer) ProcessMessage(ctx context.Context, m types.Message) error {
	n.Logger.Debug("Message ID:", zap.String("messageId", *m.MessageId))
	n.Logger.Debug("Message Body:", zap.String("messageBody", *m.Body))

	// get the s3 document from the sqs notification's metadata
	notification := sqsNotification{}
	err := json.Unmarshal([]byte(*m.Body), &notification)
	if err != nil {
		n.Logger.Error("could not deserialize sqs notification")
		return err
	}

	if notification.Records[0].EventName == "ObjectCreated:Put" && notification.Records[0].Metadata.Object.Key != "" {
		n.Logger.Info(
			"s3 object to retrieve is:", zap.String(
				"s3_object_retrieve",
				notification.Records[0].Metadata.Object.Key,
			),
		)
		res, err := n.s3Reader.Read(ctx, notification.Records[0].Metadata.Object.Key)
		if err != nil {
			n.Logger.Error("could not read s3 document based on sqs message: ", zap.Error(err))
			return err
		}

		// deserialize into IngestDocument to find out callback url
		doc := &ingest.IngestDocument{}
		s3body, err := io.ReadAll(res)
		if err != nil {
			n.Logger.Error("could deserialize s3 document:", zap.Error(err))
			return err
		}

		if err = proto.Unmarshal(s3body, doc); err != nil {
			n.Logger.Error("could not unmarshal into ingest document", zap.Error(err))
			return err
		}
		n.Logger.Debug("ingest document is: ", zap.String("ingest_document", doc.String()))

		// call the callback url
		configMap := doc.Topology.GetConfiguration()
		if c, ok := configMap["callback_url"]; ok {
			n.Logger.Debug("calling the callback at:", zap.String("callback_url", c))
			err = n.Executor.Execute(c, strings.NewReader(doc.String()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Consumer) deleteMessage(ctx context.Context, receiptHandle *string) error {
	deleteMessageInput := sqs.DeleteMessageInput{
		QueueUrl:      n.queueUrl,
		ReceiptHandle: receiptHandle,
	}
	_, err := DeleteMessage(ctx, n.sqsClient, &deleteMessageInput)
	if err != nil {
		n.Logger.Error("could not delete SQS message: ", zap.String("receipt_handle", *receiptHandle), zap.Error(err))
		return err
	}
	n.Logger.Info("deleted message:", zap.String("deleted_receipt_handle", *receiptHandle))
	return nil
}

func NewConsumer(ctx context.Context, config Config) (Consumer, error) {
	c := Consumer{
		Logger:    config.Logger,
		sqsClient: config.ReceiverDeleter,
		s3Reader:  config.S3Reader,
		queueName: config.Name,
		Executor:  config.Executor,
	}
	c.init(ctx)
	return c, nil
}

func (n *Consumer) init(ctx context.Context) {
	getQueueInput := &sqs.GetQueueUrlInput{
		QueueName: &n.queueName,
	}

	queueUrlResult, err := GetQueueUrl(ctx, n.sqsClient, getQueueInput)
	if err != nil {
		n.Error("got an error while getting queue url:", zap.Error(err))
		return
	}

	n.queueUrl = queueUrlResult.QueueUrl
}
