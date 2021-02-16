package main

import (
	"github.com/goserg/Golang-merchant-API/controller"

	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

const (
	user     = "postgres"
	password = "pass"
	dbname   = "postgres"
)

func main() {
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	controller := controller.NewController(db)

	http.HandleFunc("/", controller.HomePage)
	http.HandleFunc("/offers", controller.OffersHandler)
	http.HandleFunc("/info", controller.InfoHandler)

	fmt.Println("API started.")

	log.Fatal(http.ListenAndServe(":8001", nil))
}
