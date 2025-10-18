package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/joho/godotenv"
)

var apiKey = os.Getenv("APCA_API_KEY_ID")
var apiSecret = os.Getenv("APCA_API_SECRET_KEY")
var baseURL = os.Getenv("APCA_BASE_URL")

func main() {
	godotenv.Load()

	client := alpaca.NewClient(alpaca.ClientOpts{

		APIKey:    apiKey,
		APISecret: apiSecret,
		BaseURL:   baseURL,
	})
	acct, err := client.GetAccount()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", *acct)

	//call data function retrieving market data
	data()
}

// streaming stock market data
func data() {

	url := "https://data.alpaca.markets/v2/stocks/trades/latest?symbols=AAPL"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("APCA-API-KEY-ID", apiKey)
	req.Header.Add("APCA-API-SECRET-KEY", apiSecret)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("failed to read response body: %v", err)
	}

	fmt.Println(string(body))
}
