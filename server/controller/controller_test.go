package controller

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

func TestInfoHandlerIncorrectID(t *testing.T) {
	db := getDB()
	defer db.Close()
	c := NewController(db)
	body := infoRequest{
		TaskID: 1,
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", "/info", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(c.InfoHandler)

	fillTestSchema(db)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expectedBody := `{"error":"incorrect task_id"}`
	expectedCode := http.StatusNotFound

	if rr.Code != expectedCode {
		t.Errorf("handler returned unexpected code: got %d want %d", rr.Code, expectedCode)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %s want %s", rr.Body.String(), expectedBody)
	}
}

func TestInfoHandlerCorrectID(t *testing.T) {
	db := getDB()
	defer db.Close()
	c := NewController(db)
	fillTestSchema(db)

	body := infoRequest{
		TaskID: 5,
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", "/info", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(c.InfoHandler)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expectedBody := `{"task_id":5,"status":"statusT","elapsed_time":"20","lines_parsed":1,"new_offers":1,"updated_offers":1,"errors":1}`
	expecetedCode := http.StatusOK

	if rr.Code != expecetedCode {
		t.Errorf("handler returned unexpected code: got %d want %d", rr.Code, expecetedCode)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %s want %s", rr.Body.String(), expectedBody)
	}
}

func TestGetOfferHandlerNoResults(t *testing.T) {
	db := getDB()
	defer db.Close()
	c := NewController(db)
	fillTestSchema(db)

	body := getOffersReq{
		OfferID:   1,
		SellerID:  1,
		NameSerch: "no offer",
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", "/info", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(c.OffersHandler)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expectedBody := `{"error":"No match"}`
	expectedCode := http.StatusNotFound

	if rr.Code != expectedCode {
		t.Errorf("handler return unexpected code: got %d want %d", rr.Code, expectedCode)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %s want %s", rr.Body.String(), expectedBody)
	}
}

func TestGetOfferHandlerHaveResults(t *testing.T) {
	db := getDB()
	defer db.Close()
	c := NewController(db)
	fillTestSchema(db)

	body := getOffersReq{
		OfferID:   1,
		SellerID:  3,
		NameSerch: "test_name",
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("GET", "/offers", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(c.OffersHandler)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expectedBody := `[{"offer_id":1,"name":"test_name","price":1.1,"quantity":1,"available":true,"seller_id":3}]`
	expectedCode := http.StatusOK

	if rr.Code != expectedCode {
		t.Errorf("handler return unexpected code: got %d want %d", rr.Code, expectedCode)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %s want %s", rr.Body.String(), expectedBody)
	}
}

func TestPostOfferAsync(t *testing.T) {
	db := getDB()
	defer db.Close()
	c := NewController(db)
	fillTestSchema(db)

	body := postOffersRequest{
		URL:      "test",
		SellerID: 4,
		Async:    true,
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/offers", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(c.OffersHandler)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expectedBody := `Processing started, your task ID is 1`
	expectedCode := http.StatusOK

	if rr.Code != expectedCode {
		t.Errorf("handler return unexpected code: got %d want %d", rr.Code, expectedCode)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %s want %s", rr.Body.String(), expectedBody)
	}
}

func TestPostOfferSyncBadURL(t *testing.T) {
	db := getDB()
	defer db.Close()
	c := NewController(db)
	fillTestSchema(db)

	body := postOffersRequest{
		URL:      "test",
		SellerID: 4,
		Async:    false,
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/offers", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(c.OffersHandler)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expectedBody := `{"task_id":1,"status":"ERROR: Parsing error. Cannot load file","elapsed_time":"","lines_parsed":0,"new_offers":0,"updated_offers":0,"errors":0}`
	expectedCode := http.StatusBadRequest

	if rr.Code != expectedCode {
		t.Errorf("handler return unexpected code: got %d want %d", rr.Code, expectedCode)
	}

	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %s want %s", rr.Body.String(), expectedBody)
	}
}

func TestPostOfferSyncMockURL(t *testing.T) {
	db := getDB()
	defer db.Close()
	c := NewController(db)
	fillTestSchema(db)

	body := postOffersRequest{
		URL:      "http://localhost:8080/1.xlsx",
		SellerID: 4,
		Async:    false,
	}
	jBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/offers", bytes.NewReader(jBody))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(c.OffersHandler)

	handler.ServeHTTP(rr, req)

	clearTestSchema(db)

	expectedBodyPrefix := `{"task_id":1,"status":"Finished"`
	expectedCode := http.StatusOK

	if rr.Code != expectedCode {
		t.Errorf("handler return unexpected code: got %d want %d", rr.Code, expectedCode)
	}

	if !strings.HasPrefix(rr.Body.String(), expectedBodyPrefix) {
		t.Errorf("handler returned unexpected body: got %s has to start with %s", rr.Body.String(), expectedBodyPrefix)
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
	db.Exec(
		`INSERT INTO "offer" (id, name, price, quantity, available, seller_id)
		VALUES(1, 'test_name', 1.1, 1, true, 3);`,
	)
}

func clearTestSchema(db *sql.DB) {
	db.Exec(`DROP SCHEMA test_schema CASCADE`)
}

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
