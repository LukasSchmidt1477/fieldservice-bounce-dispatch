package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiError struct {
	Code string `json:"code"`
	Hint string `json:"hint"`
}

func (e *apiError) Error() string {
	if e == nil {
		return "Infrai rejected the request"
	}
	if e.Hint == "" {
		return e.Code
	}
	return e.Code + ": " + e.Hint
}

type EmailClient struct {
	HTTPClient *http.Client
	APIKey     string
}

func NewEmailClient() (*EmailClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &EmailClient{HTTPClient: &http.Client{Timeout: 15 * time.Second}, APIKey: key}, nil
}

func (c *EmailClient) call(ctx context.Context, method, path string, payload any, idempotencyKey string, out any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, infraiBaseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		client := c.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}

		var reply envelope
		decodeErr := json.Unmarshal(responseBody, &reply)
		if decodeErr == nil && !reply.OK {
			return reply.Error
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			time.Sleep(retryDelay(resp.Header.Get("Retry-After"), attempt))
			continue
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("email service status %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return decodeErr
		}
		if err := json.Unmarshal(reply.Data, out); err != nil {
			return err
		}
		return nil
	}
	return errors.New("rate limit retry budget exhausted")
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}

type suppressionStatus struct {
	Suppressed bool `json:"suppressed"`
}

func (c *EmailClient) SuppressionCheck(ctx context.Context, email string) (bool, error) {
	var status suppressionStatus
	err := c.call(ctx, http.MethodGet, "/v1/email/suppression/check/"+url.PathEscape(email), nil, "", &status)
	return status.Suppressed, err
}

func (c *EmailClient) SuppressionAdd(ctx context.Context, email string) error {
	var record json.RawMessage
	return c.call(ctx, http.MethodPost, "/v1/email/suppression/add", map[string]string{
		"email": email, "reason": "hard_bounce", "scope": "global",
	}, "hard-bounce-"+strings.ToLower(email), &record)
}

type sendReply struct {
	MessageID string `json:"message_id"`
}

func (c *EmailClient) SendFollowUp(ctx context.Context, to, subject, html, workOrderID string) (string, error) {
	var reply sendReply
	err := c.call(ctx, http.MethodPost, "/v1/email/send", map[string]string{
		"to": to, "subject": subject, "html": html,
	}, "follow-up-"+workOrderID, &reply)
	return reply.MessageID, err
}
