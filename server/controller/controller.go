package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/goserg/Golang-merchant-API/parser"
)

//Controller это контроллер для обработки html запросов
type Controller struct {
	db *sql.DB
}

type infoRequest struct {
	TaskID int64 `json:"task_id"`
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

type getOffersReq struct {
	OfferID   int    `json:"offer_id"`
	SellerID  int    `json:"seller_id"`
	NameSerch string `json:"name_search"`
}

type postOffersRequest struct {
	URL      string `json:"url"`
	SellerID int    `json:"seller_id"`
	Async    bool   `json:"async"`
}

//NewController создает новый контроллер
func NewController(db *sql.DB) *Controller {
	return &Controller{db}
}

//InfoHandler обработка запросов /info
func (c *Controller) InfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		var reqData infoRequest
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		json.Unmarshal(body, &reqData)
		c.provideInfo(reqData.TaskID, w, r)
	}
}

func (c *Controller) provideInfo(id int64, w http.ResponseWriter, r *http.Request) {
	log, hasTask := c.getTaskLog(id)

	if !hasTask {
		respData := infoResponseError{"incorrect task_id"}
		jData, err := json.Marshal(respData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Internal server error")
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write(jData)
		return
	}
	jData, err := json.Marshal(log)
	if err != nil {
		fmt.Fprintln(w, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jData)
}

//HomePage обработка запросов /
func (c *Controller) HomePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<html>Привет. Это API для загрузки данных по товарам в базу данных."+
		"Документацию можно почитать на "+
		"<a href='https://github.com/goserg/Golang-merchant-API'>https://github.com/goserg/Golang-merchant-API</a></html>",
	)
}

//OffersHandler обработка запросов /offers
func (c *Controller) OffersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		c.getOfferHandler(w, r)
	}
	if r.Method == http.MethodPost {
		c.postOfferHandler(w, r)
	}
}

func (c *Controller) getOfferHandler(w http.ResponseWriter, r *http.Request) {
	var search getOffersReq
	var offers []parser.Offer
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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

	rows, err := c.db.Query(`SELECT * FROM "offer" WHERE ` + fmt.Sprint(strings.Join(query, " AND ")))
	for rows.Next() {
		var offer parser.Offer
		err = rows.Scan(&offer.OfferID, &offer.Name, &offer.Price, &offer.Quantity, &offer.Available, &offer.SellerID)
		if err != nil {
			fmt.Println(err)
		}
		offers = append(offers, offer)
	}
	if len(offers) == 0 {
		respData := infoResponseError{"No match"}
		jData, err := json.Marshal(respData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Internal server error")
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write(jData)
		return
	}

	jOffers, _ := json.Marshal(offers)
	w.WriteHeader(http.StatusOK)
	w.Write(jOffers)
}

func (c *Controller) postOfferHandler(w http.ResponseWriter, r *http.Request) {
	var data postOffersRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	json.Unmarshal(body, &data)
	if data.URL != "" && data.SellerID != 0 {
		if err = c.db.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "ERROR: Unable to connect to the Data Base")
			return
		}
		if !c.hasSeller(data.SellerID) {
			c.insertSeller(data.SellerID)
		}
		logID := c.insertTaskLog(data.URL, data.SellerID)
		if data.Async {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Processing started, your task ID is %d\n", logID)
			go c.process(data.URL, data.SellerID, logID)
			return
		}
		c.process(data.URL, data.SellerID, logID)
		c.provideInfo(logID, w, r)
	}
}

func (c *Controller) getTaskLog(logID int64) (*infoResponse, bool) {
	var url string
	var sellerID int
	l := infoResponse{}
	err := c.db.QueryRow(
		`SELECT * from "task_log" WHERE  id=$1`, logID).Scan(&l.TaskID, &url, &sellerID, &l.Status, &l.ElapsedTime, &l.LinesParsed, &l.NewOffers, &l.UpdatedOffers, &l.Errors)
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	return &l, true
}

func (c *Controller) hasSeller(sellerID int) bool {
	var currentSellerID int
	err := c.db.QueryRow(
		`SELECT * from "seller" WHERE id=$1`, sellerID,
	).Scan(&currentSellerID)
	if err != nil {
		return false
	}
	return true
}

func (c *Controller) insertSeller(sellerID int) {
	_, err := c.db.Exec(fmt.Sprintf(`INSERT INTO "seller" ("id") VALUES(%d)`, sellerID))
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Controller) insertTaskLog(url string, sellerID int) int64 {
	var lid int64
	err := c.db.QueryRow(`INSERT INTO "task_log" ("status", "url", "seller_id") VALUES('Processing...', $1, $2) RETURNING id`, url, sellerID).Scan(&lid)
	if err != nil {
		fmt.Println(err)
	}
	return lid
}

func (c *Controller) process(url string, sellerID int, logID int64) {
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		c.updateTaskLog(logID, "ERROR: Parsing error. Cannot load file", "", 0, 0, 0, 1)
		return
	}
	defer resp.Body.Close()

	f, err := parser.OpenReader(resp.Body)
	if err != nil {
		c.updateTaskLog(logID, "ERROR: Parsing error. Cannot read xlsx file", "", 0, 0, 0, 1)
		return
	}
	offers, numberOfErrors := parser.ParseExcel(f)

	updates := 0
	inserts := 0
	for i := 0; i < len(offers); i++ {
		offer, hasOffer := c.getOffer(offers[i].OfferID, sellerID)
		if !hasOffer {
			c.insertOffer(offers[i], sellerID)
			inserts++
			continue
		}
		if *offer != offers[i] {
			c.updateOffer(offers[i], sellerID)
			updates++
		}
	}
	t := time.Now()
	elapsed := t.Sub(start)

	c.updateTaskLog(logID, "Finished", fmt.Sprint(elapsed), len(offers), inserts, updates, numberOfErrors)
}

func (c *Controller) updateTaskLog(
	id int64,
	status string,
	elapsedTime string,
	linesParsed int,
	newOffers int,
	updatedOffers int,
	errors int,
) {
	_, err := c.db.Exec(
		`UPDATE task_log SET status=$1, elapsed_time=$2, lines_parsed=$3, new_offers=$4, updated_offers=$5, errors=$6 WHERE id=$7`,
		status, elapsedTime, linesParsed, newOffers, updatedOffers, errors, id)
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Controller) updateOffer(offer parser.Offer, sellerID int) {
	_, err := c.db.Exec(`UPDATE offer SET name=$1, price=$2, quantity=$3, available=$4 WHERE id=$5 AND seller_id=$6`,
		offer.Name, offer.Price, offer.Quantity, offer.Available, offer.OfferID, sellerID)
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Controller) getOffer(offerID int, sellerID int) (*parser.Offer, bool) {
	var offer parser.Offer
	var currentSellerID int

	err := c.db.QueryRow(
		`SELECT * from "offer" WHERE  id=$1 AND seller_id=$2`, offerID, sellerID,
	).Scan(&offer.OfferID, &offer.Name, &offer.Price, &offer.Quantity, &offer.Available, &currentSellerID)
	if err != nil {
		return nil, false
	}
	return &offer, true
}

func (c *Controller) insertOffer(offer parser.Offer, sellerID int) {
	_, err := c.db.Exec(
		`INSERT INTO public.offer
		(id, name, price, quantity, available, seller_id)
		VALUES($1, $2, $3, $4, $5, $6);`,
		offer.OfferID, offer.Name, offer.Price, offer.Quantity, offer.Available, sellerID,
	)
	if err != nil {
		fmt.Println(err)
	}
}
