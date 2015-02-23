package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var Client JSONClient

// Create Client to communicate with GOKMS
func CreateClient() {
	authKey := os.Getenv("GOKMSCLI_AUTHKEY")
	baseUrl := os.Getenv("GOKMSCLI_URL")

	if authKey == "" || baseUrl == "" {
		Exit("Enivronmental Variable: GOKMSCLI_AUTHKEY or GOKMSCLI_URL are empty!  You must set these values!", 2)
	}

	Client = JSONClient{Client: http.DefaultClient, Endpoint: baseUrl, AuthKey: authKey}
}

// JSONClient is the underlying client for JSON APIs.
type JSONClient struct {
	Client   *http.Client
	Endpoint string
	// authKey is the key used for authentication
	AuthKey string
}

// Do sends an HTTP request and returns an HTTP response, following policy
// (e.g. redirects, cookies, auth) as configured on the client.
func (c *JSONClient) Do(method, uri string, req, resp interface{}) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(method, c.Endpoint+uri, bytes.NewReader(b))
	if err != nil {
		return err
	}
	httpReq.Header.Set("User-Agent", "GO-KMS-CLI")
	httpReq.Header.Set("Content-Type", "application/json")

	httpReq = c.SetAuth(httpReq, method, uri)

	httpResp, err := c.Client.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			return err
		}
		if len(bodyBytes) == 0 {
			return APIError{
				StatusCode: httpResp.StatusCode,
				Message:    httpResp.Status,
			}
		}
		var jsonErr jsonErrorResponse
		if err := json.Unmarshal(bodyBytes, &jsonErr); err != nil {
			return err
		}
		return jsonErr.Err(httpResp.StatusCode)
	}

	if resp != nil {
		return json.NewDecoder(httpResp.Body).Decode(resp)
	}
	return nil
}

// SetAuth will set kms auth headers
func (c *JSONClient) SetAuth(request *http.Request, method string, resource string) *http.Request {

	date := time.Now().UTC().Format(time.RFC1123) // UTC time
	request.Header.Add("x-kms-date", date)

	authRequestKey := fmt.Sprintf("%s\n%s\n%s", method, date, resource)

	hmac := GetHmac256(authRequestKey, c.AuthKey)

	//fmt.Printf("SharedKey: %s HMAC: %s RequestKey: \n%s\n", SharedKey, hmac, authRequestKey)

	request.Header.Add("Authorization", hmac)

	return request
}

type jsonErrorResponse struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

func (e jsonErrorResponse) Err(StatusCode int) error {
	return APIError{
		StatusCode: StatusCode,
		Type:       e.Type,
		Message:    e.Message,
	}
}

// An APIError is an error returned by an AWS API.
type APIError struct {
	StatusCode int // HTTP status code e.g. 200
	Type       string
	Code       string
	Message    string
	RequestID  string
	HostID     string
	Specifics  map[string]string
}

func (e APIError) Error() string {
	return e.Message
}
