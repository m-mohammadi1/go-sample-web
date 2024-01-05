package main

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func Test_application_handlers(t *testing.T) {
	var theTests = []struct {
		name                    string
		url                     string
		expectedStatusCode      int
		expectedUrl             string
		expectedFirstStatusCode int
	}{
		{"home", "/", http.StatusOK, "/", http.StatusOK},
		{"404", "/does-not-exists", http.StatusNotFound, "/does-not-exists", http.StatusNotFound},
		{"profile", "/user/profile", http.StatusOK, "/", http.StatusTemporaryRedirect},
	}

	routes := app.routes()

	// create test server
	ts := httptest.NewTLSServer(routes)
	defer ts.Close() // before function closes

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, test := range theTests {
		resp, err := ts.Client().Get(ts.URL + test.url)

		if err != nil {
			t.Log(err)
			t.Fatal(err)
		}

		if resp.StatusCode != test.expectedStatusCode {
			t.Errorf("for %s: expected status %d; got %d", test.name, test.expectedStatusCode, resp.StatusCode)
		}

		if resp.Request.URL.Path != test.expectedUrl {
			t.Errorf("for %s: expected final url %s; got %s", test.name, test.expectedUrl, resp.Request.URL.Path)
		}

		resp2, _ := client.Get(ts.URL + test.url)

		if resp2.StatusCode != test.expectedFirstStatusCode {
			t.Errorf("for %s: expected first status %d; got %d", test.name, test.expectedStatusCode, resp2.StatusCode)
		}
	}
}

func TestApp_renderBadTemplate(t *testing.T) {
	// set invalid template path
	pathTpTemplates = "./testdata/"

	req, _ := http.NewRequest("GET", "/", nil)
	req = addContextAndSessionToRequest(req, app)
	rr := httptest.NewRecorder()

	err := app.render(rr, req, "bad.page.gohtml", &TemplateData{})

	if err == nil {
		t.Error("expected error from bad template, not get any")
	}

	pathTpTemplates = "./../../templates"
}

func TestApplication_Home(t *testing.T) {
	var tests = []struct {
		name         string
		putInSession string
		expectedHtml string
	}{
		{"first-visit", "", "From Session"},
		{"second-visit", "Hello,world", "Hello,world"},
	}

	for _, e := range tests {
		// create request
		req, _ := http.NewRequest("GET", "/", nil)
		req = addContextAndSessionToRequest(req, app)
		_ = app.Session.Destroy(req.Context())

		if e.putInSession != "" {
			app.Session.Put(req.Context(), "test", e.putInSession)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.Home)
		handler.ServeHTTP(rr, req)

		// check status
		if rr.Code != http.StatusOK {
			t.Errorf("TestAppHome returned wrong status code; expected 200; got %d", rr.Code)
		}

		body, _ := io.ReadAll(rr.Body)
		if !strings.Contains(string(body), e.expectedHtml) {
			t.Errorf("%s: did not found %s in response body", e.name, e.expectedHtml)
		}
	}
}

func getCtx(req *http.Request) context.Context {
	ctx := context.WithValue(req.Context(), contextUserKey, "unknown")

	return ctx
}

func addContextAndSessionToRequest(req *http.Request, app application) *http.Request {
	req = req.WithContext(getCtx(req))
	ctx, _ := app.Session.Load(req.Context(), req.Header.Get("X-Session"))

	return req.WithContext(ctx)
}

func Test_app_login(t *testing.T) {
	var tests = []struct {
		name               string
		postData           url.Values
		expectedStatusCode int
		expectedLoc        string
	}{
		{
			name: "valid-login",
			postData: url.Values{
				"email":    {"admin@example.com"},
				"password": {"secret"},
			},
			expectedStatusCode: http.StatusSeeOther,
		},
	}

	for _, e := range tests {
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(e.postData.Encode()))
		req = addContextAndSessionToRequest(req, app)
		req.Header.Set("Content-Type", "application/x-www-form-url-urlencoded") // when browser send post form
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(app.Login)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("%s: returned wrong status code, expected %d; got %d", e.name, e.expectedStatusCode, rr.Code)
		}
	}
}
