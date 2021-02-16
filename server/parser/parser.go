package parser

import (
	"io"
	"strconv"

	"github.com/360EntSecGroup-Skylar/excelize"
)

//Offer структуа
type Offer struct {
	OfferID   int     `json:"offer_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int64   `json:"quantity"`
	Available bool    `json:"available"`
	SellerID  int     `json:"seller_id"`
}

//OpenReader открывает xlsx файл из тела респонса
func OpenReader(body io.ReadCloser) (*excelize.File, error) {
	return excelize.OpenReader(body)
}

//ParseExcel парсит xlsx файл
func ParseExcel(file *excelize.File) ([]Offer, int) {
	var offers []Offer
	numberOfErrors := 0

	rows := file.GetRows("data")

	for _, row := range rows {
		o := Offer{}
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
