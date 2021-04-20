package CoinbaseAPI

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SecureRequest struct {
	Url           string
	RequestName   string
	RequestMethod string
	RequestBody   string
	Timestamp     time.Time
	Credentials   *CoinbaseCredentials
}

func NewSecureRequest(RequestName string) *SecureRequest {
	return &SecureRequest{
		Url:           UrlForRequestName(RequestName),
		RequestName:   RequestName,
		RequestMethod: "GET", // default, change as needed
		Timestamp:     time.Now(),
		Credentials: &CoinbaseCredentials{
			Key:        "",
			Passphrase: "",
			Secret:     "",
		},
	}
}

type CoinbaseCredentials struct {
	// PRIVATE! NEVER EXPOSE!
	Key        string
	Passphrase string
	Secret     string
}

type CoinbaseAccount struct {
	// PRIVATE! NEVER EXPOSE directly!
	AccountID      string  `json:"id"`
	Currency       string  `json:"currency"`
	Balance        float64 `json:"balance"`
	Available      float64 `json:"available"`
	Hold           float64 `json:"hold"`
	ProfileID      string  `json:"profile_id"`
	TradingEnabled bool    `json:"trading_enabled"`
}

func UrlForRequestName(name string) string {
	switch strings.ToLower(name) {
	case "list_accounts":
		{
			return "/accounts"
		}
	default:
		{
			return ""
		}
	}
}

func (s *SecureRequest) Process(logger *log.Logger) ([]byte, error) {
	var (
		err   error
		req   *http.Request
		reply []byte
	)
	if len(s.RequestBody) < 1 {
		req, err = http.NewRequest(s.RequestMethod, "https://api.pro.coinbase.com"+s.Url, nil)
	} else {
		req, err = http.NewRequest(s.RequestMethod, "https://api.pro.coinbase.com"+s.Url, bytes.NewBuffer([]byte(s.RequestBody)))
	}
	if err != nil {
		if logger != nil {
			logger.Printf("[SecureRequest::Process] Error creating request: %s", err)
		}
		return reply, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CB-ACCESS-KEY", s.Credentials.Key)
	req.Header.Set("CB-ACCESS-TIMESTAMP", string(s.Timestamp.Unix()))
	req.Header.Set("CB-ACCESS-PASSPHRASE", s.Credentials.Passphrase)
	// Generate the signature
	// decode Base64 secret
	var sec []byte
	num, err := base64.StdEncoding.Decode(sec, []byte(s.Credentials.Secret))
	if err != nil {
		if logger != nil {
			logger.Printf("Error decoding secret: %s", err)
		}
		return reply, err
	}
	if logger != nil {
		logger.Printf("[SecureRequest::Process] Decoded Secret Length: %d", num)
	}
	// Create SHA256 HMAC w/ secret
	h := hmac.New(sha256.New, sec)
	// write timestamp
	h.Write([]byte(strconv.FormatInt(s.Timestamp.Unix(), 10)))
	//write method
	h.Write([]byte(s.RequestMethod))
	//write path
	h.Write([]byte(s.Url))
	//write body (if any)
	if len(s.RequestBody) > 1 {
		h.Write([]byte(s.RequestBody))
	}
	var sha []byte
	num = hex.Encode(sha, h.Sum(nil))
	if logger != nil {
		logger.Printf("[SecureRequest::Process] Encode Signature Length: %d", num)
	}
	// encode the result to base64
	var shaEnc []byte
	base64.StdEncoding.Encode(shaEnc, sha)
	req.Header.Set("CB-ACCESS-SIGN", string(sha))

	// Send
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux i686; rv:10.0) Gecko/20100101 Firefox/10.0")
	c := &http.Client{}
	c.Timeout = 20 * time.Second
	resp, err := c.Do(req)
	if err != nil {
		if logger != nil {
			logger.Printf("Error reading response: %s", err)
		}
		return reply, err
	}
	defer resp.Body.Close()

	reply, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.Printf("Error reading body: %s", err)
		}
		return reply, err
	}
	return reply, err
}
