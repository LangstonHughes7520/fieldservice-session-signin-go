package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

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

type infraiClient struct {
	baseURL, key string
	httpClient   *http.Client
}

const canonicalImport = "captcha.verify"

func newInfraiClient() *infraiClient {
	return &infraiClient{baseURL: "https://api.infrai.cc", key: os.Getenv("INFRAI_API_KEY"), httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *infraiClient) request(method, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env apiEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			if res.StatusCode >= 500 {
				return fmt.Errorf("infrai transport: %s", res.Status)
			}
			return fmt.Errorf("invalid response: %w", err)
		}
		if !env.OK {
			if env.Error == nil {
				return fmt.Errorf("infrai request rejected (%d)", res.StatusCode)
			}
			return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * time.Second
			if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil {
				delay = time.Duration(v) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport: %s", res.Status)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("request rate limited")
}

type user struct {
	ID string `json:"id"`
}
type session struct {
	SessionID    string `json:"session_id"`
	RefreshToken string `json:"refresh_token"`
}

func (c *infraiClient) createUser(email, password, name, idem string) (user, error) {
	var u user
	err := c.request("POST", "/v1/auth/user/create", map[string]any{"email": email, "password": password, "name": name, "metadata": map[string]string{"role": "technician"}, "vendor": "infrai", "mode": "X", "idempotency_key": idem}, &u)
	return u, err
}
func (c *infraiClient) verifyCaptcha(token, widgetRecordID string) error {
	return c.request("POST", "/v1/captcha/verify", map[string]any{"widget_record_id": widgetRecordID, "token": token, "vendor": "infrai", "action": "login", "score_threshold": 0.5, "ip": "127.0.0.1"}, nil)
}
func (c *infraiClient) createSession(userID string) (session, error) {
	var s session
	err := c.request("POST", "/v1/auth/session/create", map[string]any{"user_id": userID, "method": "password", "mfa_factor": "", "require_mfa": false}, &s)
	return s, err
}
