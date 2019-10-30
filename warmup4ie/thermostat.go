package warmup4ie

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

const (
	AppId      = "WARMUP-APP-V001"
	AppToken   = "M=;He<Xtg\"$}4N%5k{$:PD+WA\"]D<;#PriteY|VTuA>_iyhs+vA\"4lic{6-LqNM:"
	tokenUrl   = "https://api.warmup.com/apps/app/v1"
	graphqlUrl = "https://apil.warmup.com/graphql"
)

type RunMode int

const (
	// Heater stopped
	RunModeOff RunMode = iota
	// Programation enabled
	RunModeProg
	// Heated forced for limited time
	RunModeForced
	RunModeFixed
	RunModeFrost
	RunModeAway
)

func (r *RunMode) UnmarshalJSON(content []byte) error {
	value, err := strconv.Atoi(string(content))
	if err != nil {
		return err
	}
	*r = RunMode(value)
	return nil
}

func (r *RunMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(*r))
}

type Thermostat interface {
	ListLocations() (*[]Location, error)
	ListRooms() (*[]Room, error)
}

type Device struct {
	graphqlUrl string
	email      string
	token      string
	client     *http.Client
}

func NewDevice(email string, password string) (*Device, error) {
	client := &http.Client{}
	token, err := retrieveAccesToken(client, tokenUrl, email, password)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve access token: %w", err)
	}
	return &Device{graphqlUrl: graphqlUrl, client: &http.Client{}, email: email, token: token}, nil
}

func retrieveAccesToken(client *http.Client, url string, email string, password string) (string, error) {
	type requestToken struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Method   string `json:"method"`
		AppId    string `json:"appId"`
	}
	type bodyToken struct {
		Request requestToken `json:"request"`
	}
	body, err := json.Marshal(
		bodyToken{
			requestToken{
				email,
				password,
				"userLogin",
				AppId,
			},
		})

	if err != nil {
		return "", fmt.Errorf("unable to build json request: %w", err)
	}
	response, err := runHTTPtokenRequest(url, body, client)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	return parseToken(response)
}

func runHTTPtokenRequest(url string, body []byte, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}
	req.Header = defaultHeaders
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func parseToken(response *http.Response) (string, error) {
	jsonResponse := &struct {
		Status *struct {
			Result string
		}
		Response *struct {
			Token string
		}
	}{}
	jsonContent, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(jsonContent, jsonResponse); err != nil {
		return "", err
	}
	if response.StatusCode != 200 || jsonResponse.Status.Result != "success" {
		return "", fmt.Errorf("invalid response from Warmup server, code: %v", response.Status)
	}
	return jsonResponse.Response.Token, nil
}

func (d *Device) ListLocations() (*[]Location, error) {
	body := strings.NewReader(fmt.Sprintf(`{
"account": {
    "email": "%s",
    "token": "%s"
},
"request": {
    "method": "getLocations"
}
}`, d.email, d.token))

	var response LocationResponse
	if err := d.postRequest(tokenUrl, nil, body, &response); err != nil {
		return nil, err
	}

	if response.Status.Result != "success" {
		return nil, fmt.Errorf("failed to fetch locations from warmup server: %v", response)
	}

	return &response.Message.GetLocations.Result.Data.User.Locations, nil
}

func (d *Device) ListRooms() (*[]Room, error) {
	body := strings.NewReader(`{
"query": "query QUERY{ user{ currentLocation: location { id name rooms{ id roomName runModeInt targetTemp currentTemp thermostat4ies {minTemp maxTemp}}  }}  } "
}`)

	var response RoomResponse
	headers := []*customHeader{
		{key: "warmup-authorization", value: d.token},
	}
	if err := d.postRequest(d.graphqlUrl, headers, body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("failed to fetch rooms from warmup server: %v", response)
	}

	return &response.Data.User.CurrentLocation.Rooms, nil
}

type LocationResponse struct {
	Status *struct {
		Result string
	}
	Response *struct {
		Method    string
		Locations []HomeLocation
	}
	Message *struct {
		GetLocations *struct {
			Result *struct {
				Data *struct {
					User *struct {
						Id        int
						Locations []Location
					}
				}
				Status string
			}
		}
		Duration float32 `json:",string"`
	}
}

type HomeLocation struct {
	Id            int
	Name          string
	Latitude      float64 `json:",string"`
	Longitude     float64 `json:",string"`
	CountryCode   string
	Timezone      string
	Currency      int
	TempFormat    bool
	SmartGeo      bool
	LocZone       int
	LocMode       string
	HolStart      string
	HolEnd        string
	HolTemp       int
	Zone          int
	ZoneOffset    string
	ZoneDirection bool
	ZoneTime      int
	MainRoom      int
	GeoMode       int
	Now           string
	FenceArray    []int
}

type Location struct {
	Id          int
	Name        string
	GeoLocation *struct {
		Latitude  float64 `json:",string"`
		Longitude float64 `json:",string"`
	}
	Holiday *struct {
		HolStart string
		HolEnd   string
		HolTemp  int
	}
	Address *struct {
		OwmCityId   int
		CountryCode string
		Timezone    string
		Currency    int
		Address1    string
		Address2    string
		Town        string
		Postcode    string
	}
	LocZone *struct {
		Zone     int
		Offset   string
		IsHoming bool
		Time     string
	}
	Settings *struct {
		MainRoom     int
		IsFahrenheit bool
		IsSmartGeo   bool
	}
	LocMode    string
	LocModeInt int
	FenceArray []int
	GeoModeInt int
}

type RoomResponse struct {
	Status string
	Data   *struct {
		User *struct {
			CurrentLocation *struct {
				Id    int
				Name  string
				Rooms []Room
			}
		}
	}
}
type Temperature struct {
	RawTemperature int
}

/** Return temperature in celcius degrees */
func (t *Temperature) GetValue() float32 {
	return float32(t.RawTemperature) / 10.
}
func (t *Temperature) String() string {
	return fmt.Sprintf("%.1f°C", t.GetValue())
}
func (t *Temperature) UnmarshalJSON(content []byte) error {
	value, err := strconv.Atoi(string(content))
	if err != nil {
		return err
	}
	t.RawTemperature = value
	return nil
}
func (t *Temperature) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.RawTemperature)
}

type Room struct {
	Id             int
	Name           string  `json:"roomName"`
	RunMode        RunMode `json:"runModeInt"`
	TargetTemp     Temperature
	CurrentTemp    Temperature
	Thermostat4IES []struct {
		MinTemp Temperature
		MaxTemp Temperature
	}
}

type JsonResponse struct {
	Status *struct {
		Result string
	}
	Response *struct {
		ErrorCode int
	}
	Message string
}

var defaultHeaders = http.Header{
	"user-agent":      {"WARMUP_APP"},
	"accept-encoding": {"br, gzip, deflate"},
	"accept":          {"*/*"},
	"Connection":      {"keep-alive"},
	"content-type":    {"application/json"},
	"app-token":       {AppToken},
	"app-version":     {"1.8.1"},
	"accept-language": {"de-de"},
}

type customHeader struct {
	key   string
	value string
}

func (d *Device) postRequest(url string, headers []*customHeader, body io.Reader, response interface{}) error {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	defaultHeaders.Clone()
	req.Header = defaultHeaders.Clone()
	for _, h := range headers {
		req.Header.Add(h.key, h.value)
	}
	// Force user-agent (no list that starts with golang default value)
	req.Header.Set("user-agent", defaultHeaders.Get("user-agent"))

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid http status: %d", resp.StatusCode)
	}

	jsonContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read content %w", err)
	}
	log.Debugf("%v\n", string(jsonContent))
	err = json.Unmarshal(jsonContent, response)
	if err != nil {
		return fmt.Errorf("unable to unmarshal json content %s: %w", jsonContent, err)
	}
	return nil
}
