package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	_ "github.com/lib/pq"
)

var (
	db  *sql.DB
	err error
)

const (
	user     = "postgres"
	password = "pass"
	dbname   = "postgres"
)

type data struct {
	URL      string `json:"url"`
	SellerID int    `json:"seller_id"`
}

type getOffersReq struct {
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

type searchResponce struct {
	Offers []offer `json:"offers"`
}

func offersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		var search getOffersReq
		var offers []offer
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		json.Unmarshal(body, &search)

		var query []string
		if search.OfferID != 0 {
			query = append(query, fmt.Sprintf("id=%d", search.OfferID))
		}
		if search.SellerID != 0 {
			query = append(query, fmt.Sprintf("seller_id=%d", search.SellerID))
		}
		query = append(query, fmt.Sprintf(`"name" LIKE %s`, "'%"+search.NameSerch+"%'"))

		rows, err := db.Query(`SELECT * FROM "offer" WHERE ` + fmt.Sprint(strings.Join(query, " AND ")))
		for rows.Next() {
			var offer offer
			err = rows.Scan(&offer.OfferID, &offer.Name, &offer.Price, &offer.Quantity, &offer.Available, &offer.SellerID)
			if err != nil {
				fmt.Println(err)
			}
			offers = append(offers, offer)
		}

		jOffers, _ := json.Marshal(offers)
		fmt.Fprintln(w, string(jOffers))

	}
	if r.Method == "POST" {
		var data data
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
			if !hasSeller(data.SellerID, db) {
				insertSeller(data.SellerID, db)
			}
			logID := insertTaskLog(data.URL, data.SellerID, db)
			fmt.Fprintf(w, "Processing started, your task ID is %d\n", logID)
			go process(data.URL, data.SellerID, logID)
		}
	}
}

type infoRequest struct {
	TaskID int `json:"task_id"`
}

type infoResponse struct {
	TaskID        int    `json:"task_id"`
	Status        string `json:"status"`
	ElapsedTime   string `json:"elapsed_time"`
	LinesParsed   int    `json:"lines_parsed"`
	NewOffers     int    `json:"new_offers"`
	UpdatedOffers int    `json:"updated_offers"`
	Errors        int    `json:"errors"`
}
type infoResponseError struct {
	Err string `json:"error"`
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		var reqData infoRequest
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		json.Unmarshal(body, &reqData)

		log, hasTask := getTaskLog(int64(reqData.TaskID), db)

		if !hasTask {
			respData := infoResponseError{"incorrect task_id"}
			jData, err := json.Marshal(respData)
			if err != nil {
				fmt.Fprintln(w, "Internal server error")
				return
			}
			w.Write(jData)
			return
		}
		jData, err := json.Marshal(log)
		if err != nil {
			fmt.Fprintln(w, "Internal server error")
			return
		}
		w.Write(jData)
	}
}

type offer struct {
	OfferID   int     `json:"offer_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int64   `json:"quantity"`
	Available bool    `json:"available"`
	SellerID  int     `json:"seller_id"`
}

func parseExcel(file *excelize.File) ([]offer, int) {
	var offers []offer
	numberOfErrors := 0

	rows, err := file.GetRows("data")
	if err != nil {
		fmt.Println(err)
		return offers, 1
	}

	for _, row := range rows {
		o := offer{}
		offerID, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			numberOfErrors++
			continue
		}
		o.OfferID = int(offerID)
		o.Name = row[1]
		o.Price, err = strconv.ParseFloat(row[2], 64)
		if err != nil {
			numberOfErrors++
			continue
		}
		o.Quantity, err = strconv.ParseInt(row[3], 10, 64)
		if err != nil {
			numberOfErrors++
			continue
		}
		o.Available, err = strconv.ParseBool(row[4])
		if err != nil {
			numberOfErrors++
			continue
		}

		if o.OfferID < 0 || o.Name == "" || o.Price < 0 || o.Quantity < 0 {
			numberOfErrors++
			continue
		}

		offers = append(offers, o)
	}
	return offers, numberOfErrors
}

func homePage(w http.ResponseWriter, r *http.Request) {

}

func process(url string, sellerID int, logID int64) {
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		updateTaskLog(logID, "ERROR: Parsing error", "", 0, 0, 0, 1)
		return
	}
	defer resp.Body.Close()

	f, err := excelize.OpenReader(resp.Body)
	if err != nil {
		updateTaskLog(logID, "ERROR: Parsing error", "", 0, 0, 0, 1)
		return
	}

	offers, numberOfErrors := parseExcel(f)

	updates := 0
	inserts := 0
	for i := 0; i < len(offers); i++ {
		offer, hasOffer := getOffer(offers[i].OfferID, sellerID, db)
		if !hasOffer {
			insertOffer(offers[i], sellerID, db)
			inserts++
			continue
		}
		if *offer != offers[i] {
			updateOffer(offers[i], sellerID, db)
			updates++
		}
	}
	t := time.Now()
	elapsed := t.Sub(start)

	updateTaskLog(logID, "Finished", fmt.Sprint(elapsed), len(offers), inserts, updates, numberOfErrors)
}

func updateOffer(offer offer, sellerID int, db *sql.DB) {
	_, err := db.Exec(`UPDATE offer SET name=$1, price=$2, quantity=$3, available=$4 WHERE id=$5 AND seller_id=$6`,
		offer.Name, offer.Price, offer.Quantity, offer.Available, offer.OfferID, sellerID)
	if err != nil {
		fmt.Println(err)
	}
}

func getOffer(offerID int, sellerID int, db *sql.DB) (*offer, bool) {
	var offer offer
	var currentSellerID int

	err := db.QueryRow(
		`SELECT * from "offer" WHERE  id=$1 AND seller_id=$2`, offerID, sellerID,
	).Scan(&offer.OfferID, &offer.Name, &offer.Price, &offer.Quantity, &offer.Available, &currentSellerID)
	if err != nil {
		return nil, false
	}
	return &offer, true
}

func hasSeller(sellerID int, db *sql.DB) bool {
	var currentSellerID int
	err := db.QueryRow(
		`SELECT * from "seller" WHERE id=$1`, sellerID,
	).Scan(&currentSellerID)
	if err != nil {
		return false
	}
	return true
}

func insertOffer(offer offer, sellerID int, db *sql.DB) {
	_, err := db.Exec(
		`INSERT INTO public.offer
		(id, name, price, quantity, available, seller_id)
		VALUES($1, $2, $3, $4, $5, $6);`,
		offer.OfferID, offer.Name, offer.Price, offer.Quantity, offer.Available, sellerID,
	)
	if err != nil {
		fmt.Println(err)
	}
}

func insertSeller(sellerID int, db *sql.DB) {
	_, err := db.Exec(fmt.Sprintf(`INSERT INTO "seller" ("id") VALUES(%d)`, sellerID))
	if err != nil {
		fmt.Println(err)
	}
}

func insertTaskLog(url string, sellerID int, db *sql.DB) int64 {
	var lid int64
	err := db.QueryRow(`INSERT INTO "task_log" ("status", "url", "seller_id") VALUES('Processing...', $1, $2) RETURNING id`, url, sellerID).Scan(&lid)
	if err != nil {
		fmt.Println(err)
	}
	return lid
}

func updateTaskLog(
	id int64,
	status string,
	elapsedTime string,
	linesParsed int,
	newOffers int,
	updatedOffers int,
	errors int,
) {
	_, err := db.Exec(
		`UPDATE task_log SET status=$1, elapsed_time=$2, lines_parsed=$3, new_offers=$4, updated_offers=$5, errors=$6 WHERE id=$7`,
		status, elapsedTime, linesParsed, newOffers, updatedOffers, errors, id)
	if err != nil {
		fmt.Println(err)
	}
}

func getTaskLog(logID int64, db *sql.DB) (*infoResponse, bool) {
	var url string
	var sellerID int
	l := infoResponse{}
	err := db.QueryRow(
		`SELECT * from "task_log" WHERE  id=$1`, logID).Scan(&l.TaskID, &url, &sellerID, &l.Status, &l.ElapsedTime, &l.LinesParsed, &l.NewOffers, &l.UpdatedOffers, &l.Errors)
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	return &l, true
}
