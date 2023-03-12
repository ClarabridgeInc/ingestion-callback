package storage

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
)

// Reader defines read behavior that returns a *s3.GetObjectOutput
type Reader interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type S3Reader struct {
	Bucket string
	Reader
}

type S3Config struct {
	Bucket string `yaml:"bucket"`
}

func GetObject(c context.Context, api Reader, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return api.GetObject(c, input)
}

func (r *S3Reader) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(r.Bucket),
		Key:    aws.String(key),
	}

	res, err := GetObject(ctx, r.Reader, getObjectInput)
	if err != nil {
		return nil, fmt.Errorf("GetObject failed: %v", err)
	}
	return res.Body, nil
}
