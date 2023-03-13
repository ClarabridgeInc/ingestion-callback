package storage

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
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
	if *params.Key == "test" {
		return &s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader("test")),
		}, nil
	}
	return nil, errors.New("key does not exist")
}

func TestS3ReadSuccess(t *testing.T) {
	type test struct {
		ObjectKey   string
		ShouldError bool
		ErrMessage  string
	}

	tests := map[string]test{
		"valid object key does not error": {ObjectKey: "test", ShouldError: false},
		"nonexistent key returns error": {
			ObjectKey: "nonexistent-key", ShouldError: true,
			ErrMessage: "GetObject failed",
		},
	}

	ctx := context.TODO()
	s3Reader := &S3Reader{
		Bucket: "test-bucket",
		Reader: &MockReader{},
	}

	for name, tc := range tests {
		t.Run(
			name, func(t *testing.T) {
				res, err := s3Reader.Read(ctx, tc.ObjectKey)
				if !tc.ShouldError {
					assert.NoError(t, err)
					assert.NotNil(t, res)
				} else {
					assert.ErrorContains(t, err, tc.ErrMessage)
				}
			},
		)

	}
}
