package main

import (
	"os"
	"testing"
	"webapp/pkg/repository/dbrepo"
)

var app application
var expiredToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhZG1pbiI6dHJ1ZSwiYXVkIjoiZXVkIjoiZXhhbXBsZS5jb20iLCJleHAiOjE3MDU1NDc3MDksImlzcyI6ImVuY29tIiw4YW1wbGUuY29tIiwibmFtZSI6IkpvaG4gRG9lIiwic3ViIjoiMSJ9.0ltNfGQwDsyhAcbljXmC9kJXtpRhfGQwDsyhW-yb4knaoiqG2zI"

func TestMain(m *testing.M) {

	app.DB = &dbrepo.TestDBRepo{}
	app.Domain = "example.com"
	app.JWTSecret = "teasd32safasd1zvczvckxbnz82q"

	os.Exit(m.Run())
}
