package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
)

type sessionStore struct {
	mu     sync.RWMutex
	values map[string]string
}

func (s *sessionStore) put(id, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[id] = userID
}
func (s *sessionStore) get(id string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[id]
	return v, ok
}
func token() string { b := make([]byte, 18); _, _ = rand.Read(b); return hex.EncodeToString(b) }

type server struct {
	auth     *infraiClient
	sessions *sessionStore
}

func (s *server) signup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Email == "" || in.Password == "" {
		http.Error(w, "email and password required", 400)
		return
	}
	u, err := s.auth.createUser(in.Email, in.Password, in.Name, token())
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]string{"user_id": u.ID})
}
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID         string `json:"user_id"`
		CaptchaToken   string `json:"captcha_token"`
		WidgetRecordID string `json:"widget_record_id"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.UserID == "" {
		http.Error(w, "user_id required", 400)
		return
	}
	if in.CaptchaToken == "" || in.WidgetRecordID == "" {
		http.Error(w, "captcha_token and widget_record_id required", 400)
		return
	}
	if err := s.auth.verifyCaptcha(in.CaptchaToken, in.WidgetRecordID); err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	ss, err := s.auth.createSession(in.UserID)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	s.sessions.put(ss.SessionID, in.UserID)
	writeJSON(w, map[string]string{"session_id": ss.SessionID, "refresh_token": ss.RefreshToken})
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func main() {
	s := &server{auth: newInfraiClient(), sessions: &sessionStore{values: map[string]string{}}}
	http.HandleFunc("/signup", s.signup)
	http.HandleFunc("/login", s.login)
	_ = http.ListenAndServe(":8080", nil)
}
