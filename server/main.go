package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"database/sql"
	"time"
	"strings"

    _ "github.com/lib/pq"
	"github.com/360EntSecGroup-Skylar/excelize"
)

var (
	taskLog []string
	db *sql.DB
	err error
)
const (
	user = "postgres"
	password = "pass"
	dbname = "postgres"
)
type Data struct {
	URL      string `json:"url"`
	SellerID int    `json:"seller_id"`
	TaskID   int    `json:"task_id"`
}

type GetOffersReq  struct {
	OfferID   int    `json:"offer_id"`
	SellerID  int    `json:"seller_id"`
	NameSerch string `json:"name_search"`
}

func main() {
	fmt.Println("API started.")

	db, err = sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	http.HandleFunc("/", homePage)
	http.HandleFunc("/offers", offersHandler)
	http.HandleFunc("/info", infoHandler)

	log.Fatal(http.ListenAndServe(":8001", nil))
}

func offersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var search GetOffersReq
		var offers []Offer
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		json.Unmarshal(body, &search)

		var query []string
		if search.OfferID != 0 {
			query = append(query, fmt.Sprintf("offer_id=%d", search.OfferID))
		}
		if search.SellerID != 0 {
			query = append(query, fmt.Sprintf("seller_id=%d", search.SellerID))
		}
		query = append(query, fmt.Sprintf(`"name" LIKE %s`, "'%" + search.NameSerch + "%'"))

		rows, err := db.Query(`SELECT * FROM "offers" WHERE ` + fmt.Sprint(strings.Join(query, " AND ")))
		for rows.Next() {
			var offer Offer
			err = rows.Scan(&offer.offerID, &offer.name, &offer.price, &offer.quantity, &offer.available, &offer.sellerID)
			if err!= nil {
				fmt.Println(err)
			}
			offers = append(offers, offer)
		}
		fmt.Fprintln(w, offers)
	}
	if r.Method == "POST" {
		var data Data
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		json.Unmarshal(body, &data)
		if data.URL != "" && data.SellerID != 0 {
			if err = db.Ping(); err != nil {
				fmt.Fprintf(w, "ERROR: Unable to connect to the Data Base")
				return
			}
			taskLog = append(taskLog, "This task is not finished yet. Please try again later.")
			fmt.Fprintf(w, "Processing started, your task ID is %d\n", len(taskLog))
			log_id := len(taskLog)
			go process(data.URL, data.SellerID, log_id)
		}
	}
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		var data Data
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		json.Unmarshal(body, &data)
		fmt.Fprintf(w, "GET запрос: %d\n", data.TaskID)
		if (data.TaskID > 0) {
			if data.TaskID > len(taskLog) {
				fmt.Fprintln(w, "incorrect task_id")
				return
			}
			fmt.Fprintln(w, taskLog[data.TaskID-1])
		}
	}
}

type Offer struct {
	offerID   int
	name      string
	price     float64
	quantity  int64
	available bool
	sellerID  int
}

func parseExcel(file *excelize.File) ([]Offer, int) {
	var offers []Offer
	numberOfErrors := 0

	rows, err := file.GetRows("data")
	if err != nil {
		fmt.Println(err)
		return offers, 1
	}

	for _, row := range rows {
		o := Offer{}
		off_id, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			numberOfErrors++
			continue
		}
		o.offerID = int(off_id)
		o.name = row[1]
		o.price, err = strconv.ParseFloat(row[2], 64)
		if err != nil {
			numberOfErrors++
			continue
		}
		o.quantity, err = strconv.ParseInt(row[3], 10, 64)
		if err != nil {
			numberOfErrors++
			continue
		}
		o.available, err = strconv.ParseBool(row[4])
		if err != nil {
			numberOfErrors++
			continue
		}
		
		if (o.offerID < 0 || o.name == "" || o.price < 0 || o.quantity < 0) {
			numberOfErrors++
			continue
		}

		offers = append(offers, o)
	}
	return offers, numberOfErrors
}

func homePage(w http.ResponseWriter, r *http.Request) {

}

func process(url string, seller_id int, log_id int) {
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		taskLog[log_id-1] = "ERROR: Parsing error"
		return
	}
	defer resp.Body.Close()

    f, err := excelize.OpenReader(resp.Body)
    if err != nil {
		taskLog[log_id-1] = "ERROR: Parsing error"
        return
    }

	offers, numberOfErrors := parseExcel(f)

	if (!hasSeller(seller_id, db)) {
		insertSeller(seller_id, db)
	}
	updates := 0
	inserts := 0
	for i := 0; i < len(offers); i++ {
		offer, hasOffer := getOffer(offers[i].offerID, seller_id, db)
		if (!hasOffer) {
			insertOffer(offers[i], seller_id, db)
			inserts++
			continue
		}
		if (*offer != offers[i]) {
			updateOffer(offers[i], seller_id, db)
			updates++
		}
	}
	t := time.Now()
	elapsed := t.Sub(start)

	taskLog[log_id-1] = fmt.Sprintf(`
	Task №%d info:
	Data parsing to db finished in %s
	offers parsed: %d
	errors: %d
	new offers: %d
	updated offers: %d
	`,
	log_id, elapsed, len(offers), numberOfErrors, inserts, updates,
	)
	fmt.Println(taskLog[log_id-1])
}

func updateOffer(offer Offer, seller_id int, db *sql.DB) {
	_, err := db.Exec(`UPDATE offers SET name=$1, price=$2, quantity=$3, available=$4 WHERE offer_id=$5 AND seller_id=$6`,
		offer.name, offer.price, offer.quantity, offer.available, offer.offerID, seller_id)
	if (err != nil) {
		fmt.Println(err)
	}
}

func getOffer(offer_id int, seller_id int, db *sql.DB) (*Offer, bool) {
	var offer Offer
	var current_seller_id int

	err := db.QueryRow(`SELECT * from "offers" WHERE  offer_id=$1 AND seller_id=$2`, offer_id, seller_id).Scan(&offer.offerID, &offer.name, &offer.price, &offer.quantity, &offer.available, &current_seller_id)
	if err != nil {
		return nil, false		
	}
	return &offer, true
}

func hasSeller(seller_id int, db *sql.DB) bool {
	var current_seller_id int
	err := db.QueryRow(`SELECT * from "sellers" WHERE seller_id=$1`, seller_id).Scan(&current_seller_id)
	if err != nil {
		return false
	}
	return true
}

func insertOffer(offer Offer, seller_id int, db *sql.DB) {
	_, err := db.Exec(`INSERT INTO public.offers
	(offer_id, name, price, quantity, available, seller_id)
	VALUES($1, $2, $3, $4, $5, $6);`, offer.offerID, offer.name, offer.price, offer.quantity, offer.available, seller_id)
	if err != nil {
		fmt.Println(err)
	}
}

func insertSeller(seller_id int, db *sql.DB) {
	_, err := db.Exec(fmt.Sprintf(`INSERT INTO "sellers" ("seller_id") VALUES(%d)`, seller_id))
	if err != nil {
		fmt.Println(err)
	}
}