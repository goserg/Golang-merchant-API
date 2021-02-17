package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goserg/Golang-merchant-API/controller"
)

func getDB() *sql.DB {
	const (
		user     = "postgres"
		password = "pass"
		host     = "localhost"
		port     = 5432
		dbname   = "postgres"
	)
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=disable",
		user,
		password,
		host,
		port,
		dbname))
	if err != nil {
		panic(err)
	}
	return db
}

type infoRequest struct {
	TaskID int64 `json:"task_id"`
}

func TestInfoHandlerUncorrectID(t *testing.T) {
	db := getDB()
	defer db.Close()
	controller := controller.NewController(db)
	body := infoRequest{
		TaskID: 1,
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", "/info", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(controller.InfoHandler)

	fillTestSchema(db)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expected := `{"error":"incorrect task_id"}`

	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestInfoHandlerCorrectID(t *testing.T) {
	db := getDB()
	defer db.Close()
	controller := controller.NewController(db)
	body := infoRequest{
		TaskID: 5,
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", "/info", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(controller.InfoHandler)

	fillTestSchema(db)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expected := `{"task_id":5,"status":"statusT","elapsed_time":"20","lines_parsed":1,"new_offers":1,"updated_offers":1,"errors":1}`

	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func fillTestSchema(db *sql.DB) {
	db.Exec(
		`CREATE SCHEMA test_schema
		create table seller (
			id integer,
			PRIMARY KEY (id)
		)
		create table offer (
			id integer,
			name text NOT NULL,
			price real NOT NULL,
			quantity integer NOT NULL,
			available boolean,
			seller_id integer REFERENCES seller ON DELETE CASCADE,
			CONSTRAINT offer_seller_id UNIQUE (id, seller_id)
		)
		create table task_log (
			id BIGSERIAL,
			url char(2000),
			seller_id integer REFERENCES seller ON DELETE CASCADE,
			status text,
			elapsed_time text,
			lines_parsed integer,
			new_offers integer,
			updated_offers integer,
			errors integer,
			PRIMARY KEY (id)
		);`)
	db.Exec(`set search_path='test_schema'`)
	db.Exec(`INSERT INTO "seller" ("id") VALUES(3)`)
	db.Exec(
		`INSERT INTO "task_log" ("id", "url", "seller_id",
		"status", "elapsed_time", "lines_parsed", "new_offers",
		"updated_offers", "errors") 
		VALUES(5, 'urlT', 3, 'statusT', 20, 1, 1, 1, 1)`,
	)
}
func clearTestSchema(db *sql.DB) {
	db.Exec(`DROP SCHEMA test_schema CASCADE`)
}
