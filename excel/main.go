package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"database/sql"

    _ "github.com/lib/pq"
	"github.com/360EntSecGroup-Skylar/excelize"
)

func main() {
	fmt.Println("API started.")
	http.HandleFunc("/", homePage)
	log.Fatal(http.ListenAndServe(":8070", nil))
}

type Data struct {
	URL      string `json:"url"`
	SellerID string `json:"seller_id"`
}

type Offer struct {
	offerID   int64
	name      string
	price     float64
	quantity  int64
	available bool
}

func dbConnect() *sql.DB{
	connStr := "user=postgres password=pass dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to db")
	return db
}

func downloadFile(url string, filename string) {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	// Create the file
	out, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
}

func parseExcel(file *excelize.File) []Offer {
	var offers []Offer

	rows, err := file.GetRows("data")
	if err != nil {
		panic(err)
	}

	for n, row := range rows {
		if n == 0 {
			continue
		}
		o := Offer{}
		o.offerID, err = strconv.ParseInt(row[0], 10, 64)
		o.name = row[1]
		o.price, err = strconv.ParseFloat(row[2], 64)
		o.quantity, err = strconv.ParseInt(row[3], 10, 64)
		o.available, err = strconv.ParseBool(row[4])
		offers = append(offers, o)

		if err != nil {
			panic(err)
		}
	}
	return offers
}

func homePage(w http.ResponseWriter, r *http.Request) {
	db := dbConnect()
	defer db.Close()

	result, err := db.Exec("select * from foss")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(result)

	var data Data
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(body, &data)

	fmt.Fprintln(w, data.SellerID)
	fmt.Fprintln(w, data.URL)

	// Get the data
	fileName := "t.xlsx"
	downloadFile(data.URL, fileName)

	file, err := excelize.OpenFile(fileName)
	if err != nil {
		panic(err)
	}
	offers := parseExcel(file)

	fmt.Fprintln(w, offers)
}
