package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type infraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }

func (c *infraiClient) Publish(ctx context.Context, channel, event string, data any, id string) error {
	return c.post(ctx, "/v1/realtime/publish", map[string]any{
		"channel": channel, "event": event, "data": data,
	}, id)
}

func (c *infraiClient) Queue(ctx context.Context, payload any, id string) error {
	return c.post(ctx, "/v1/queue/publish", map[string]any{"queue": "creator-notifications", "payload": payload}, id)
}

func (c *infraiClient) post(ctx context.Context, path string, body any, id string) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+path, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", id)
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		var envelope apiEnvelope
		decodeErr := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&envelope)
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			wait := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
				wait = time.Duration(seconds) * time.Second
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if decodeErr != nil {
			return fmt.Errorf("decode Infrai response (HTTP %d): %w", res.StatusCode, decodeErr)
		}
		if !envelope.OK {
			if envelope.Error != nil {
				return envelope.Error
			}
			return fmt.Errorf("Infrai rejected request (HTTP %d)", res.StatusCode)
		}
		if res.StatusCode >= 400 {
			return fmt.Errorf("Infrai HTTP %d", res.StatusCode)
		}
		return nil
	}
	return errors.New("retry limit reached")
}
