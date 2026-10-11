package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginForwardsCaptchaFields(t *testing.T) {
	var body map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/captcha/verify" {
			t.Errorf("unexpected upstream path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode upstream request: %v", err)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"INVALID_CAPTCHA","message":"test"}}`))
	}))
	defer upstream.Close()
	s := &server{auth: &infraiClient{baseURL: upstream.URL, httpClient: upstream.Client()}, sessions: &sessionStore{values: map[string]string{}}}
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"user_id":"tech-1","captcha_token":"token-1","widget_record_id":"widget-1"}`))
	w := httptest.NewRecorder()
	s.login(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if body["token"] != "token-1" || body["widget_record_id"] != "widget-1" {
		t.Fatalf("captcha request: %#v", body)
	}
}

func TestSessionStoreKeepsTechnician(t *testing.T) {
	s := &sessionStore{values: map[string]string{}}
	cases := []struct{ name, id, want string }{{"dispatch photo follow-up", "sess-1", "tech-7"}, {"second work order", "sess-2", "tech-8"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s.put(tc.id, tc.want)
			got, ok := s.get(tc.id)
			if !ok || got != tc.want {
				t.Fatalf("session=%q ok=%v", got, ok)
			}
		})
	}
}
