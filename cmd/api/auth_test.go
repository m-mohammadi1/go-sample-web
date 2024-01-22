package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"webapp/pkg/data"
)

func Test_app_getTokenFromHeaderAndVerify(t *testing.T) {
	testUser := data.User{
		ID:        1,
		FirstName: "admin",
		LastName:  "admini",
		Email:     "admin@example.com",
	}

	tokens, _ := app.generateTokenPair(&testUser)

	tests := []struct {
		name          string
		token         string
		errorExpected bool
		setHeader     bool
		issuer        string
	}{
		{"valid", fmt.Sprintf("Bearer %s", tokens.Token), false, true, app.Domain},
		{"valid-expired", fmt.Sprintf("Bearer %s", expiredToken), true, true, app.Domain},
		{"no-header", "", true, false, app.Domain},
		{"invalid-token", fmt.Sprintf("Bearer %sasdac", tokens.Token), true, true, app.Domain},
		{"no-bearer", fmt.Sprintf("Bearerss %s", tokens.Token), true, true, app.Domain},
		{"wrong-issuer", fmt.Sprintf("Bearer %s", tokens.Token), true, true, "invalid.com"},
		{"three-header-parts", fmt.Sprintf("Bearer %s third-part", tokens.Token), true, true, app.Domain},
	}

	mainDomain := app.Domain
	for _, e := range tests {
		if e.issuer != app.Domain {
			app.Domain = e.issuer
			tokens, _ = app.generateTokenPair(&testUser)
		}

		req, _ := http.NewRequest("GET", "/", nil)

		if e.setHeader {
			req.Header.Set("Authorization", e.token)
		}

		rr := httptest.NewRecorder()

		_, _, err := app.getTokenFromHeaderAndVerify(rr, req)

		if err != nil && !e.errorExpected {
			t.Errorf("%s: did not expect error; got one - %s", e.name, err.Error())
		}

		if err == nil && e.errorExpected {
			t.Errorf("%s: expected error; got nothing", e.name)
		}

		app.Domain = mainDomain
	}
}
