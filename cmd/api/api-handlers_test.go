package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_app_authenticate(t *testing.T) {
	var tests = []struct {
		name               string
		requestBody        string
		expectedStatusCode int
	}{
		{"valid-user", `{"email": "admin@example.com", "password": "secret"}`, http.StatusOK},
		{"not-json", "not a json", http.StatusUnauthorized},
		{"empty-json", "{}", http.StatusUnauthorized},
		{"empty-email", `{"password": "secret"}`, http.StatusUnauthorized},
		{"empty-password", `{"email": "admin@example.com"}`, http.StatusUnauthorized},
		{"invalid-user", `{"email": "invalid@example.com", "password": "secret"}`, http.StatusUnauthorized},
		{"invalid-password", `{"email": "admin@example.com", "password": "invalid"}`, http.StatusUnauthorized},
	}

	for _, e := range tests {
		var reader io.Reader
		reader = strings.NewReader(e.requestBody)
		req, _ := http.NewRequest("POST", "/auth", reader)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.authenticate)

		handler.ServeHTTP(rr, req)

		if e.expectedStatusCode != rr.Code {
			t.Errorf("%s: returned wrong status code;expected %d hot %d", e.name, e.expectedStatusCode, rr.Code)
		}
	}
}
