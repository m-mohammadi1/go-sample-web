package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestForm_Has(t *testing.T) {
	form := NewForm(nil)

	has := form.Has("not_matter")

	if has {
		t.Error("form shows field when it should not")
	}

	postData := url.Values{}
	postData.Add("a", "a")
	form = NewForm(postData)

	has = form.Has("a")
	if !has {
		t.Error("shows does not have fields when it should")
	}
}

func TestForm_Required(t *testing.T) {
	r := httptest.NewRequest("POST", "/test", nil)
	form := NewForm(r.PostForm)

	form.Required("a", "b", "c")
	if form.Valid() {
		t.Error("form shows valid when required fields are missing")
	}

	postData := url.Values{}
	postData.Add("a", "a")
	postData.Add("b", "b")
	postData.Add("c", "c")

	r, _ = http.NewRequest("POST", "/test", nil)
	r.PostForm = postData

	form = NewForm(r.PostForm)
	form.Required("a", "b", "c")
	if !form.Valid() {
		t.Error("form shows invalid when required fields are exist")
	}
}

func TestForm_Check(t *testing.T) {
	form := NewForm(nil)

	form.Check(false, "password", "password is required")
	if form.Valid() {
		t.Error("Valid() returns false, it should return true when calling Check()")
	}
}

func TestForm_ErrorGet(t *testing.T) {
	form := NewForm(nil)
	form.Check(false, "password", "password is required")
	s := form.Errors.Get("password")
	if len(s) == 0 {
		t.Error("should have an error returned from Get(), but not")
	}

	s = form.Errors.Get("not_exists")
	if len(s) != 0 {
		t.Error("should not have an error, got one")
	}
}
