package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"webapp/pkg/data"
)

func Test_application_addIPToContext(t *testing.T) {
	tests := []struct {
		headerName  string
		headerValue string
		addr        string
		emptyAddr   bool
	}{
		{"", "", "", false},
		{"", "", "", true},
		{"X-Forwarded-For", "192.2.2.1", "", false},
		{"", "", "hello:world", false},
	}

	// create dummy handler to check context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// value exists in context
		val := r.Context().Value(contextUserKey)

		if val == nil {
			t.Error(contextUserKey, "not present")
		}

		// make sure it is string
		ip, ok := val.(string)
		if !ok {
			t.Error("not string")
		}
		t.Log(ip)
	})

	for _, e := range tests {
		// create handler
		handlerToTest := app.addIpToContext(nextHandler)

		req := httptest.NewRequest("GET", "http://testing", nil)

		if e.emptyAddr {
			req.RemoteAddr = ""
		}

		if len(e.headerName) > 0 {
			req.Header.Add(e.headerName, e.headerValue)
		}

		if len(e.addr) > 0 {
			req.RemoteAddr = e.addr
		}

		handlerToTest.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func Test_application_ipFromContext(t *testing.T) {
	// get a context
	ctx := context.Background()

	// put something in context using contex
	ctx = context.WithValue(ctx, contextUserKey, "test1")

	//call the function
	ip := app.ipFromContext(ctx)

	// perform test
	if !strings.EqualFold("test1", ip) {
		t.Error("wrong value returned from context")
	}
}

func Test_app_auth(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})

	var tests = []struct {
		name   string
		isAuth bool
	}{
		{"logged in", true},
		{"not logged in", false},
	}

	for _, e := range tests {
		handlerToTest := app.auth(nextHandler)

		req := httptest.NewRequest("GET", "http://testing", nil)
		req = addContextAndSessionToRequest(req, app)

		if e.isAuth {
			app.Session.Put(req.Context(), "user", data.User{ID: 1})
		}
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		if e.isAuth && rr.Code != http.StatusOK {
			t.Errorf("%s: excpected status code 200; got %d", e.name, rr.Code)
		}

		if !e.isAuth && rr.Code != http.StatusTemporaryRedirect {
			t.Errorf("%s: excpected status code 307; got %d", e.name, rr.Code)
		}
	}
}
