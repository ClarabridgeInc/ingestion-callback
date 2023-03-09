package callback

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCallBack(t *testing.T) {
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

	logger, _ := zap.NewDevelopment()
	cfg := Config{
		Timeout: 5 * time.Second,
		Logger:  logger,
	}
	executor := NewCallbackExecutor(cfg)

	type test struct {
		Uri         string
		ShouldError bool
		ErrMessage  string
	}

	tests := map[string]test{
		"valid callback does not error": {Uri: mockServer.URL + "/callback", ShouldError: false},
		"invalid callback endpoint returns error": {
			Uri: mockServer.URL + "/nonexistent", ShouldError: true,
			ErrMessage: "callback execution failed with non 200 status code",
		},
		"invalid callback server returns error": {
			Uri: "https://nonesistent-server", ShouldError: true,
			ErrMessage: "could not execute callback",
		},
		//"invalid uri specification returns error": {
		//	Uri: "https://", ShouldError: true,
		//	ErrMessage: "failed to create http request",
		//},
	}

	for name, tc := range tests {
		t.Run(
			name, func(t *testing.T) {
				err := executor.Execute(tc.Uri, strings.NewReader("callback"))
				if !tc.ShouldError {
					assert.NoError(t, err)
				} else {
					assert.ErrorContainsf(
						t, err, tc.ErrMessage, "expected error containing %v, got %v",
						tc.ErrMessage, err,
					)
				}
			},
		)
	}
}
