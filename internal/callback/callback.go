package callback

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httputil"
	"time"
)

type Executor struct {
	*http.Client
	*zap.Logger
}

type Config struct {
	Timeout time.Duration
	*zap.Logger
}

func (c *Executor) Execute(uri string, body io.Reader) error {
	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		c.Logger.Error("failed to create request:", zap.Error(err))
		return err
	}

	bodyarr, _ := httputil.DumpRequest(req, true)
	c.Logger.Info("", zap.String("request", string(bodyarr)))
	resp, err := c.Do(req)
	if err != nil {
		c.Logger.Error("could not execute callback:", zap.Error(err))
		return err
	}

	if resp.StatusCode != 200 {
		c.Logger.Error(
			"callback execution failed, non 200 status code:", zap.Int("callback_status_code", resp.StatusCode),
		)
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
		cfg.Logger,
	}
}
