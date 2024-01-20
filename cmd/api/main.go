package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"webapp/pkg/repository"
	"webapp/pkg/repository/dbrepo"
)

const port = 8090

type application struct {
	DSN       string
	DB        repository.DatabaseRepo
	Domain    string
	JWTSecret string
}

func main() {
	var app application

	flag.StringVar(&app.Domain, "domain", "example.com", "Domain for application")
	flag.StringVar(&app.DSN, "dsn", "host=localhost port=6432 user=postgres password=postgres dbname=users sslmode=disable timezone=UTC connect_timeout=5", "postgres connection")
	flag.StringVar(&app.JWTSecret, "jwt-secret", "teasd32safasd1zvczvckxbnz82q", "signing secret")
	flag.Parse()

	conn, err := app.connectToDB()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	app.DB = &dbrepo.PostgresDBRepo{
		DB: conn,
	}

	log.Printf("starting api on port %d", port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), app.routes())
	if err != nil {
		log.Fatal(err)
	}
}
