package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"webapp/pkg/data"
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

func Test_app_UploadFiles(t *testing.T) {
	// set up pipes
	pr, pw := io.Pipe()

	// create a new writer
	writer := multipart.NewWriter(pw)

	// create a wait group and add 1 to it
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// simulate uploading file using goroutine
	go simulatePNGUpload("./testdata/img.png", writer, t, wg)

	// read from pipe that receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	// call app.UploadFiles
	uploadedFiles, err := app.uploadFiles(request, "./testdata/uploads/")
	if err != nil {
		t.Error(err)
	}

	// perform tests
	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].OriginalFileName)); os.IsNotExist(err) {
		t.Errorf("excpected file to exists: %s", err.Error())
	}

	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].OriginalFileName))

	wg.Wait()
}

func simulatePNGUpload(fileToUpload string, writer *multipart.Writer, t *testing.T, wg *sync.WaitGroup) {
	defer writer.Close()
	defer wg.Done()

	// create the form data field 'file'
	part, err := writer.CreateFormFile("file", path.Base(fileToUpload))
	if err != nil {
		t.Error(err)
	}

	// open the file
	f, err := os.Open(fileToUpload)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	// decode image
	img, _, err := image.Decode(f)
	if err != nil {
		t.Error("error decoding image: ", err)
	}

	// write the png to io.Writer
	err = png.Encode(part, img)
	if err != nil {
		t.Error(err)
	}
}

func Test_app_UploadProfilePic(t *testing.T) {
	uploadPath = "./testdata/uploads"
	filePath := "./testdata/img.png"

	// specify a field name for the form
	fieldName := "file"

	// create a bytes.Buffer to add as request body
	body := new(bytes.Buffer)

	// create a new writer
	mw := multipart.NewWriter(body)

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}

	w, err := mw.CreateFormFile(fieldName, filePath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := io.Copy(w, file); err != nil {
		t.Fatal(err)
	}

	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/", body)
	req = addContextAndSessionToRequest(req, app)
	app.Session.Put(req.Context(), "user", data.User{ID: 1})
	req.Header.Add("Content-Type", mw.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(app.UploadProfilePic)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code")
	}

	_ = os.Remove("./testdata/uploads/img.png")
}
