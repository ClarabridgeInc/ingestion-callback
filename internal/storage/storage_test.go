package storage

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

type MockReader struct {
}

func (m *MockReader) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(options *s3.Options)) (
	*s3.GetObjectOutput, error,
) {
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader("test")),
	}, nil
}

func TestS3ReadSuccess(t *testing.T) {
	ctx := context.TODO()
	s3Reader := &S3Reader{
		Bucket: "test-bucket",
		Reader: &MockReader{},
	}
	res, err := s3Reader.Read(ctx, "test-key")
	assert.NoError(t, err)
	resBody, err := io.ReadAll(res)
	assert.NoError(t, err)
	assert.Equal(t, "test", string(resBody))
}
