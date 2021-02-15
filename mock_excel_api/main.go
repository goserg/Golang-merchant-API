package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func indexPage(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadFile("./excels" + r.URL.Path)
	if err != nil {
		log.Print(err)
	}
	w.Write(b)
}

func main() {
	fmt.Println("Mock excel server run.")
	http.HandleFunc("/", indexPage)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
