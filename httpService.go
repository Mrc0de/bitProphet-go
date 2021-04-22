package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mrc0de/BitProphet-Go/CoinbaseAPI"
	api "github.com/mrc0de/BitProphet-Go/CoinbaseAPI" //shit like this is why we cant have nice things....
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type httpService struct {
	HTTPServer        *http.Server
	HTTPServerChannel chan *httpServiceEvent
	WsService         *WebSocketService
	Quit              bool
}

type httpServiceEvent struct {
	Time      time.Time
	RemoteIP  net.IP
	EventType string
	EventData string
}

func (h *httpService) Init() {
	defer func() {
		if r := recover(); r != nil {
			logger.Printf("[httpService.Init] [UNHANDLED_ERROR]: %s", r)
			debug.PrintStack()
			os.Exit(1)
		}
	}()

	// HTTP(s) Service
	logger.Printf("[httpService.Init] Starting WWW Service [HTTPS]")
	r := mux.NewRouter().StrictSlash(true)
	// routes
	r.HandleFunc("/", WWWHome).Methods("GET")
	r.HandleFunc("/stats/user/default", processTimeout(InternalUserStats, 10*time.Second)).Methods("GET")
	//r.HandleFunc("/stats/market/{id}", WWWHome).Methods("POST")
	r.Use(h.LogRequest)

	// Websocket Setup
	h.WsService.WebServer = h
	go h.WsService.WsHub.run()
	r.HandleFunc("/ws", func(w2 http.ResponseWriter, r2 *http.Request) {
		websocketUpgrade(h.WsService.WsHub, w2, r2)
	})

	logger.Printf("%v", h.HTTPServer)
	h.HTTPServer = &http.Server{
		Addr:         Config.Web.Listen + ":443",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		if err := h.HTTPServer.ListenAndServeTLS(Config.Web.CertFile, Config.Web.KeyFile); err != nil {
			logger.Printf("[httpService.Init] %s", err)
		}
	}()
	logger.Println("[httpService.Init] [Started HTTPS]")

	// Start http redirect to https
	go func() {
		if err := http.ListenAndServe("0.0.0.0:80", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			logger.Printf("[%s] Redirecting %s from HTTP to HTTPS", r.RequestURI, r.RemoteAddr)
			http.Redirect(rw, r, "https://"+r.Host+r.URL.String(), http.StatusFound) // 302 doesnt get cached
		})); err != nil {
			logger.Printf("[StartRedirectToHTTPS] %s", err)
		}
	}()

}

// request logger
func (h *httpService) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("[Request] [URI: %s] [Host: %s] [Len: %d] [RemoteAddr: %s] \r\n\t\t\t[UserAgent: %s]",
			r.RequestURI, r.Host, r.ContentLength, r.RemoteAddr, r.UserAgent())
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Printf("[Request] Error Reading Body: %s", err)
		} else if len(body) != 0 {
			logger.Printf("[Request] [Body:%s]", body)
		}
		next.ServeHTTP(w, r)
	})
}

// route handlers
func WWWHome(w http.ResponseWriter, r *http.Request) {
	var data struct {
		WsHost string
	}
	data.WsHost = r.Host
	logger.Printf("[%s] %s %s", r.RequestURI, r.Method, r.RemoteAddr)
	tmpl := template.Must(template.ParseFiles(Config.Web.Path+"/home.tmpl", Config.Web.Path+"/base.tmpl"))
	err := tmpl.Execute(w, &data)
	if err != nil {
		logger.Printf("Error Parsing Template: %s", err)
	}
}

func processTimeout(h http.HandlerFunc, duration time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), duration)
		defer cancel()
		r = r.WithContext(ctx)
		processDone := make(chan bool)
		go func() {
			h(w, r)
			processDone <- true
		}()
		select {
		case <-ctx.Done():
			w.Write([]byte(`{"error": "process timeout"}`))
		case <-processDone:
		}
	}
}

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
	replyOrders, err := req.Process(logger) // process request
	if err != nil {
		logger.Printf("[InternalUserStats] ERROR: %s", err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{Error: "You broke something"})
		return
	}
	logger.Printf("replyOrders: %s", replyOrders)
	var orderList []api.CoinbaseOrder
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

	w.Header()["Content-Type"] = []string{"application/json"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(AccountStats)
}
