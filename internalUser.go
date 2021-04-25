package main

import (
	"encoding/json"
	"fmt"
	api "github.com/mrc0de/BitProphet-Go/CoinbaseAPI"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/////////////////////////////
// src: https://docs.pro.coinbase.com
////////////////////////////////////////////////////////////////
//All REST requests must contain the following headers:
//
//CB-ACCESS-KEY The api key as a string.
//CB-ACCESS-SIGN The base64-encoded signature (see Signing a Message).
//CB-ACCESS-TIMESTAMP A timestamp for your request.
//CB-ACCESS-PASSPHRASE The passphrase you specified when creating the API key.
//
//All request bodies should have content type application/json and be valid JSON.

// The CB-ACCESS-SIGN header is generated by creating a sha256 HMAC using the base64-decoded secret key
// on the prehash string timestamp + method + requestPath + body (where + represents string concatenation)
// and base64-encode the output.
//
//The timestamp value is the same as the CB-ACCESS-TIMESTAMP header.
//The body is the request body string or omitted if there is no request body (typically for GET requests).
//The method should be UPPER CASE.
////////////////////////////////////////////////////////////////

type InternalUserStat struct {
	Currency  string
	Balance   float64
	Available float64
	Hold      float64
}

type InternalUserOrder struct {
	Price         string    `json:"price"`
	Size          string    `json:"size"`
	ProductId     string    `json:"product_id"`
	Side          string    `json:"side"`
	Stp           string    `json:"stp"`
	Type          string    `json:"type"`
	TimeInForce   string    `json:"time_in_force"`
	PostOnly      bool      `json:"post_only"`
	CreatedAt     time.Time `json:"created_at"`
	FillFees      string    `json:"fill_fees"`
	FilledSize    string    `json:"filled_size"`
	ExecutedValue string    `json:"executed_value"`
	Status        string    `json:"status"`
	Settled       bool      `json:"settled"`
}

func InternalUserStats(w http.ResponseWriter, r *http.Request) {
	// fetch stats for internal user
	// /stats/user/default
	// PUBLIC!! DO NOT OUTPUT ANY SENSITIVE DATA!!!
	if !Config.BPInternalAccount.Enabled {
		http.Error(w, fmt.Sprintf("Not Allowed"), http.StatusForbidden)
		return
	}
	// Buys and matching sells with brief activity analysis
	// Transaction Steps:
	// Get the keys and secret and passphrase (from somewhere, config, env, etc, using config file for starters)
	// Determine the URL and the body contents (if any)
	// take timestamp of now
	// Produce Signature
	// Write Headers (w/ signature etc)
	// Write body (if any, most have none)
	// Post and fetch reply
	// Clean reply and ONLY RETURN NON SENSITIVE DATA to users
	// By default, this is the ONLY account that 'brags' but could be enabled on other users, if desired
	// If this works well enough, I might never make other users muuuhahahahaha
	///////////////////////////////////////////////////////////////////////////
	logger.Printf("[PUBLIC]   [InternalUserStats]   [PUBLIC]")
	req := api.NewSecureRequest("list_accounts", Config.CBVersion) // create the req
	req.Credentials.Key = Config.BPInternalAccount.AccessKey       // setup it's creds
	req.Credentials.Passphrase = Config.BPInternalAccount.PassPhrase
	req.Credentials.Secret = Config.BPInternalAccount.Secret
	reply, err := req.Process(logger) // process request
	if err != nil {
		logger.Printf("[InternalUserStats] ERROR: %s", err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{Error: "You broke something"})
		return
	}
	var accList []api.CoinbaseAccount
	err = json.Unmarshal(reply, &accList)
	if err != nil {
		logger.Printf("[InternalUserStats] ERROR: %s", err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{Error: "You broke something"})
		return
	}

	logger.Printf("[InternalUserStats] Found %d Accounts", len(accList))
	var relevantAccounts []api.CoinbaseAccount
	for _, coin := range Config.BPInternalAccount.DefaultCoins {
		for _, acc := range accList {
			if acc.Currency == coin || acc.Currency == Config.BPInternalAccount.NativeCurrency {
				relevantAccounts = append(relevantAccounts, acc)
			}
		}
	}
	logger.Printf("[InternalUserStats] Found %d Relevant Accounts", len(relevantAccounts))
	var AccountStats []InternalUserStat
	for z, a := range relevantAccounts {
		var stat InternalUserStat
		logger.Printf("[InternalUserStats] [%d] Coin %s", z, a.Currency)
		stat.Currency = a.Currency
		if a.Currency == "USD" {
			a.Balance = a.Balance[:strings.Index(a.Balance, ".")+3]
		}
		logger.Printf("[InternalUserStats] [%d] Balance: %s", z, a.Balance)
		stat.Balance, err = strconv.ParseFloat(a.Balance, 64)
		if err != nil {
			logger.Printf("[InternalUserStats] ERROR: %s", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(struct {
				Error string `json:"error"`
			}{Error: "You broke something"})
			return
		}
		if a.Currency == "USD" {
			a.Available = a.Available[:strings.Index(a.Available, ".")+3]
		}
		logger.Printf("[InternalUserStats] [%d] Available: %s", z, a.Available)
		stat.Available, err = strconv.ParseFloat(a.Available, 64)
		if err != nil {
			logger.Printf("[InternalUserStats] ERROR: %s", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(struct {
				Error string `json:"error"`
			}{Error: "You broke something"})
			return
		}
		if a.Currency == "USD" {
			a.Hold = a.Hold[:strings.Index(a.Hold, ".")+3]
		}
		logger.Printf("[InternalUserStats] [%d] Held: %s", z, a.Hold)
		stat.Hold, err = strconv.ParseFloat(a.Hold, 64)
		if err != nil {
			logger.Printf("[InternalUserStats] ERROR: %s", err)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(struct {
				Error string `json:"error"`
			}{Error: "You broke something"})
			return
		}
		logger.Printf("[InternalUserStats] [%d] Enabled: %t", z, a.TradingEnabled)
		logger.Printf("[InternalUserStats] ----\t----\t----\t----")
		AccountStats = append(AccountStats, stat)
	}
	// Stats done, get orders
	reqOrders := api.NewSecureRequest("list_orders", Config.CBVersion) // create the req
	reqOrders.Credentials.Key = Config.BPInternalAccount.AccessKey     // setup it's creds
	reqOrders.Credentials.Passphrase = Config.BPInternalAccount.PassPhrase
	reqOrders.Credentials.Secret = Config.BPInternalAccount.Secret
	replyOrders, err := reqOrders.Process(logger) // process request
	if err != nil {
		logger.Printf("[InternalUserStats] ERROR: %s", err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{Error: "You broke something"})
		return
	}
	var orderList []InternalUserOrder
	err = json.Unmarshal(replyOrders, &orderList)
	if err != nil {
		logger.Printf("[InternalUserStats] ERROR: %s", err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{Error: "You broke something"})
		return
	}
	logger.Printf("[InternalUserStats] Orders Found: %d", len(orderList))

	/// output
	var stats struct {
		Accounts []InternalUserStat
		Orders   []InternalUserOrder
	}
	stats.Accounts = AccountStats
	stats.Orders = orderList
	w.Header()["Content-Type"] = []string{"application/json"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}