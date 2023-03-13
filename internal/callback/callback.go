package callback

import (
	"context"
	"fmt"
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

func (c *Executor) Execute(ctx context.Context, uri string, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, "POST", uri, body)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("could not execute callback: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("callback execution failed with non 200 status code: %w", err)
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
