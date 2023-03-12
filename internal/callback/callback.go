package callback

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"time"
)

type Executor struct {
	*http.Client
}

type Config struct {
	Timeout time.Duration
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (c *Executor) Execute(ctx context.Context, uri string, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, "POST", uri, body)
	if err != nil {
		return fmt.Errorf("failed to create http request: %v", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("could not execute callback: %v", err)
	}

	if resp.StatusCode != 200 {
		return errors.New("callback execution failed with non 200 status code")
	}

	return nil
}

func NewCallbackExecutor(cfg Config) Executor {
	return Executor{
		&http.Client{
			Transport: http.DefaultTransport,
			Timeout:   cfg.Timeout,
		},
	}
}
