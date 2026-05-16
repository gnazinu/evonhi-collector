package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	headerToken     = "X-Connector-Token"
	headerTimestamp = "X-Evonhi-Timestamp"
	userAgent       = "evonhi-collector/0.1"
)

type Client struct {
	endpoint   string
	token      string
	http       *http.Client
	backoff    *Backoff
}

func NewClient(endpoint, token string, perRequestTimeout, maxElapsed time.Duration) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	return &Client{
		endpoint: endpoint,
		token:    token,
		http: &http.Client{
			Transport: transport,
			Timeout:   perRequestTimeout,
		},
		backoff: NewBackoff(maxElapsed),
	}
}

// Send POSTs the envelope with exponential-backoff retries on 5xx/429/network errors.
// 4xx (except 429) fail fast — likely auth or contract issue.
func (c *Client) Send(ctx context.Context, env *Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	c.backoff.Reset()
	attempt := 0
	for {
		attempt++
		status, respBody, err := c.doOnce(ctx, body)
		if err == nil && status >= 200 && status < 300 {
			log.Printf("send: ok (attempt=%d, status=%d, bytes=%d)", attempt, status, len(body))
			return nil
		}

		retryable := isRetryable(status, err)
		log.Printf("send: attempt=%d status=%d err=%v retryable=%v body=%s", attempt, status, err, retryable, truncate(respBody, 200))

		if !retryable {
			if err != nil {
				return fmt.Errorf("non-retryable error: %w", err)
			}
			return fmt.Errorf("non-retryable status %d: %s", status, truncate(respBody, 200))
		}

		delay, ok := c.backoff.NextDelay()
		if !ok {
			return fmt.Errorf("max elapsed exceeded after %d attempts (last status=%d, err=%v)", attempt, status, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (c *Client) doOnce(ctx context.Context, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set(headerToken, c.token)
	req.Header.Set(headerTimestamp, time.Now().UTC().Format(time.RFC3339))

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	return resp.StatusCode, rb, nil
}

func isRetryable(status int, err error) bool {
	if err != nil {
		return true
	}
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= 500 && status <= 599
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
