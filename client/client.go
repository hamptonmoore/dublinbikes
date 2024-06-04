package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type DublinBikesClient struct {
	accountID              string
	accountEmail           string
	accountPassword        string
	accessToken            string
	refreshToken           string
	oauthOAuthRefreshToken string
	oauthIDToken           string
	oauthExpiresIn         int
	oauthScope             string
	oauthTokenType         string
}

type ClientTokensResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func NewDublinBikesClient(account_id string, email string, password string) (*DublinBikesClient, error) {
	bikeclient := DublinBikesClient{
		accountID:       account_id,
		accountEmail:    email,
		accountPassword: password,
	}

	err := bikeclient.generateClientTokens()
	if err != nil {
		fmt.Printf("Error generating client tokens: %v\n", err)
		return nil, err
	}

	code, err := bikeclient.login()
	if err != nil {
		fmt.Printf("Error logging in: %v\n", err)
		return nil, err
	}

	err = bikeclient.exchangeCodeForToken(code)
	if err != nil {
		fmt.Printf("Error exchanging code for token: %v\n", err)
		return nil, err
	}

	err = bikeclient.RefreshAccessToken()
	if err != nil {
		fmt.Printf("Error refreshing access token: %v\n", err)
		return nil, err
	}

	return &bikeclient, nil
}

func (client *DublinBikesClient) generateClientTokens() error {
	url := "https://api.cyclocity.fr/auth/environments/PRD/client_tokens"
	data := map[string]string{
		"code": "vls.web.dublin:PRD",
		"key":  "0398667a307bbd0d8258a8c9b81dc11657aacae406a1b406a6b26b26ecc7f60e",
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response status: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	var authResp ClientTokensResponse
	err = json.Unmarshal(body, &authResp)
	if err != nil {
		return fmt.Errorf("error unmarshaling response body: %v", err)
	}

	client.accessToken = authResp.AccessToken
	client.refreshToken = authResp.RefreshToken

	return nil
}

func (client *DublinBikesClient) login() (string, error) {
	loginURL := fmt.Sprintf("https://api.cyclocity.fr/identities/users/login?takn=%s&email=%s&password=%s&redirect_uri=%s",
		client.accessToken, url.QueryEscape(client.accountEmail), url.QueryEscape(client.accountPassword), url.QueryEscape("https://www.dublinbikes.ie/openid_connect_login"))

	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	httpClient := &http.Client{
		// Follow redirect, but do not automatically
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Stop after first redirect
			return http.ErrUseLastResponse
		},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("unexpected status code: %v", resp.Status)
	}

	locationHeader := resp.Header.Get("Location")
	if locationHeader == "" {
		return "", fmt.Errorf("no Location header found in response")
	}

	locationURL, err := url.Parse(locationHeader)
	if err != nil {
		return "", fmt.Errorf("error parsing Location URL: %v", err)
	}

	code := locationURL.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("no code found in Location URL")
	}

	return code, nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

func (client *DublinBikesClient) exchangeCodeForToken(code string) error {
	url := fmt.Sprintf("https://api.cyclocity.fr/identities/token?grant_type=authorization_code&code=%s&redirect_uri=%s",
		url.QueryEscape(code), url.QueryEscape("https://www.dublinbikes.ie/openid_connect_login"))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Taknv1 %s", client.accessToken))
	req.Header.Set("Content-Type", "application/json")

	clientHTTP := &http.Client{}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response status: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return fmt.Errorf("error unmarshaling response body: %v", err)
	}

	client.accessToken = tokenResp.AccessToken
	client.oauthOAuthRefreshToken = tokenResp.RefreshToken
	client.oauthIDToken = tokenResp.IDToken
	client.oauthExpiresIn = tokenResp.ExpiresIn
	client.oauthScope = tokenResp.Scope
	client.oauthTokenType = tokenResp.TokenType

	return nil
}

func (client *DublinBikesClient) RefreshAccessToken() error {
	url := "https://api.cyclocity.fr/auth/access_tokens"
	data := map[string]string{
		"refreshToken": client.refreshToken,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	clientHTTP := &http.Client{}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response status: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	var tokenResp ClientTokensResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return fmt.Errorf("error unmarshaling response body: %v", err)
	}

	client.accessToken = tokenResp.AccessToken

	return nil
}

type Trip struct {
	ID              string `json:"id"`
	MovementRef     string `json:"movementRef"`
	SubscriptionID  string `json:"subscriptionId"`
	SubscriptionRef string `json:"subscriptionRef"`
	ContractName    string `json:"contractName"`
	AccountID       string `json:"accountId"`
	Status          string `json:"status"`
	BikeNumber      int    `json:"bikeNumber"`
	StartDateTime   string `json:"startDateTime"`
	StartStation    int    `json:"startStation"`
	EndDateTime     string `json:"endDateTime"`
	EndStation      int    `json:"endStation"`
	StartStand      int    `json:"startStand"`
	EndStand        int    `json:"endStand"`
	Duration        int    `json:"duration"`
	RewardsEarned   int    `json:"rewardsEarned"`
	RewardsSpent    int    `json:"rewardsSpent"`
	Price           int    `json:"price"`
	Discount        int    `json:"discount"`
	ReducedPrice    int    `json:"reducedPrice"`
	Litigious       bool   `json:"litigious"`
	IsSpecial       bool   `json:"isSpecial"`
	IsRated         bool   `json:"isRated"`
}

func (client *DublinBikesClient) GetTrips() ([]Trip, error) {
	err := client.RefreshAccessToken()
	if err != nil {
		return nil, fmt.Errorf("error refreshing access token: %v", err)
	}

	url := fmt.Sprintf("https://api.cyclocity.fr/contracts/dublin/accounts/%s/trips", client.accountID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Identity", client.oauthIDToken)
	req.Header.Set("Authorization", fmt.Sprintf("Taknv1 %s", client.accessToken))
	req.Header.Set("Accept", "application/json, text/plain, */*")

	clientHTTP := &http.Client{}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response status: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var trips []Trip
	err = json.Unmarshal(body, &trips)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response body: %v", err)
	}

	return trips, nil
}
