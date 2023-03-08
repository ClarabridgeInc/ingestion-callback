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
	}

	tests := []test{
		{Uri: mockServer.URL + "/callback", ShouldError: false},
		{Uri: mockServer.URL + "/nonexistent", ShouldError: true},
		{Uri: "https://nonesistent-server", ShouldError: true},
	}

	for _, tc := range tests {
		err := executor.Execute(tc.Uri, strings.NewReader("callback"))
		if !tc.ShouldError {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err, "callback execution failed with non 200 status code")
		}
	}
}
