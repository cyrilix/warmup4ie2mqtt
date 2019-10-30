package warmup4ie

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"testing"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{})
}

func initThermostat(t *testing.T) *Device {
	device, err := NewDevice(email, password)
	if err != nil || device == nil {
		t.Errorf("unexpected error: %v", err)
	}
	return device
}

func TestNewDevice(t *testing.T) {
	device, err := NewDevice(email, password)
	if err != nil || device == nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDevice_GetLocations(t *testing.T) {
	device := initThermostat(t)

	if _, err := device.ListLocations(); err != nil {
		fmt.Printf("Test failed: %v", err)
		t.Fail()
	}
}

func TestDevice_GetRooms(t *testing.T) {
	device := initThermostat(t)

	if rooms, err := device.ListRooms(); err != nil {
		fmt.Printf("Test failed: %v", err)
		t.Fail()
	} else {
		for _, room := range *rooms {
			fmt.Printf("%+v\n", room)
		}
	}
}

func TestRunMode_Marshal(t *testing.T) {
	content, err := json.Marshal(&struct {
		ModeForced RunMode
		ModeAway   RunMode
		ModeOff    RunMode
		ModeFixed  RunMode
		ModeFrost  RunMode
		ModeProg   RunMode
	}{
		ModeForced: RunModeForced,
		ModeAway:   RunModeAway,
		ModeOff:    RunModeOff,
		ModeFixed:  RunModeFixed,
		ModeFrost:  RunModeFrost,
		ModeProg:   RunModeProg,
	})
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}

	expectedJson := "{\"ModeForced\":2,\"ModeAway\":5,\"ModeOff\":0,\"ModeFixed\":3,\"ModeFrost\":4,\"ModeProg\":1}"
	if string(content) != expectedJson {
		t.Errorf("invalid json marshalling:\nexpected=%s\n  actual=%s", expectedJson, string(content))
	}
	fmt.Println(string(content))

}

func TestRunMode_UnmarshalJSON(t *testing.T) {
	content := "{\"ModeForced\":2,\"ModeAway\":5,\"ModeOff\":0,\"ModeFixed\":3,\"ModeFrost\":4,\"ModeProg\":1}"
	goContent := struct {
		ModeForced RunMode
		ModeAway   RunMode
		ModeOff    RunMode
		ModeFixed  RunMode
		ModeFrost  RunMode
		ModeProg   RunMode
	}{}
	err := json.Unmarshal([]byte(content), &goContent)
	if err != nil {
		t.Errorf("unmarshalling error: %v\n", err)
	}

	r := []struct {
		expected RunMode
		actual   RunMode
	}{
		{RunModeAway, goContent.ModeAway},
		{RunModeFixed, goContent.ModeFixed},
		{RunModeForced, goContent.ModeForced},
		{RunModeFrost, goContent.ModeFrost},
		{RunModeOff, goContent.ModeOff},
		{RunModeProg, goContent.ModeProg},
	}

	for _, res := range r {
		if res.expected != res.actual {
			t.Errorf("invalid conversion, %v should be %v\n", res.actual, res.expected)
		}
	}
}

func TestTemperature_GetValue(t *testing.T) {
	temp := Temperature{RawTemperature: 123}
	if temp.GetValue() != 12.3 {
		t.Errorf("bad conversion, expected: 12.3, actual: %d", temp)
	}

	temp = Temperature{0}
	if temp.GetValue() != 0.0 {
		t.Errorf("bad conversion, expected: 0.0, actual: %d", temp)
	}
}

func TestTemperature_String(t *testing.T) {
	temp := Temperature{}
	if temp.String() != "0.0°C" {
		t.Errorf("unexpected string for nil temperature: %v\n", temp.String())
	}

	temp = Temperature{112}
	if temp.String() != "11.2°C" {
		t.Errorf("unexpected string for temperature: %v\n, expected 11.2°C", temp.String())
	}
}

func TestTemperature_UnmarshalJSON(t *testing.T) {
	content := `{ "temperature": 234}`
	goContent := struct{ Temperature Temperature }{}

	err := json.Unmarshal([]byte(content), &goContent)
	if err != nil {
		t.Errorf("unable to unmarshall temperature: %+v\n", err)
	}
	if goContent.Temperature.RawTemperature != 234 {
		t.Errorf("bad raw temperature: %d, expected %d", goContent.Temperature.RawTemperature, 234)
	}
}

func TestTemperature_MarshalJSON(t *testing.T) {
	type JsonResult struct {
		Temperature Temperature `json:"Temperature,Temperature"`
	}
	content, err := json.Marshal(&JsonResult{Temperature: Temperature{RawTemperature: 456}})
	if err != nil {
		t.Errorf("unable to marshall temperature: %+v\n", err)
	}
	expected := `{"Temperature":456}`
	if string(content) != expected {
		t.Errorf("bad marshalling,\nexpected: %v\n  actual: %v\n", expected, string(content))
	}
}

func TestRetrieveAccessToken(t *testing.T) {
	expectedToken := "ekneknejgnel"
	var returnJsonTokenHandler = func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("App-Token") != AppToken {
			t.Errorf("bad App-token: %s", r.Header.Get("App-Token"))
		}
		if r.Method != "POST" {
			t.Errorf("bad method: %s, expected POST", r.Method)
		}
		w.WriteHeader(200)
		_, err := fmt.Fprintf(w, "{\"status\":{\"result\":\"success\"},\"response\":{\"method\":\"userLogin\",\"token\":\"%s\",\"mobileName\":null},\"message\":{\"duration\":\"0.082\"}}", expectedToken)
		if err != nil {
			t.Errorf("unable to write response: %v", err)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(returnJsonTokenHandler))
	defer server.Close()

	token, err := retrieveAccesToken(&http.Client{}, server.URL, "email@test", "passowrd")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if token != expectedToken {
		t.Errorf("bad token, expected '%s', actual '%s'", expectedToken, token)
	}
}

func TestDevice_ListRooms(t *testing.T) {
	token := "gkhgkTokenhgj"

	var returnJsonRoomsHandler = func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Header: %v", r.Header)
		for k, v := range defaultHeaders {
			if r.Header.Get(k) != v[0] {
				t.Errorf("expected header %s not found or bad value: expected=%v, actual=%v", k, v[0], r.Header.Get(k))
			}
		}
		if r.Header.Get("warmup-authorization") != token {
			t.Errorf("bad token used for authentication: %s, expected %s", r.Header["warmup-authorization"], token)
		}

		w.WriteHeader(200)
		_, err := fmt.Fprint(w, `{"data":{"user":{"currentLocation":{"id":1234,"name":"Home","rooms":[{"id":5678,"roomName":"Room1","runModeInt":1,"targetTemp":220,"currentTemp":235,"thermostat4ies":[{"minTemp":50,"maxTemp":300}]},{"id":91234,"roomName":"Room2","runModeInt":3,"targetTemp":210,"currentTemp":230,"thermostat4ies":[{"minTemp":50,"maxTemp":300}]}]}}},"status":"success"}`)
		if err != nil {
			t.Errorf("unable to write response: %v", err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(returnJsonRoomsHandler))
	defer server.Close()

	device := Device{
		graphqlUrl: server.URL,
		email:      "email@test.com",
		token:      token,
		client:     &http.Client{},
	}
	rooms, err := device.ListRooms()
	if err != nil {
		t.Errorf("unexpected error: %v\n", err)
	}
	if len(*rooms) != 2 {
		t.Errorf("2 rooms expected, actual: %d", len(*rooms))
	}
	room1 := (*rooms)[0]
	if room1.Id != 5678 {
		t.Errorf("invalid id expected:%d , actual:%d", 5678, room1.Id)
	}
	if room1.Name != "Room1" {
		t.Errorf("invalid name expected:%v , actual:%v", "Room1", room1.Name)
	}
	if room1.CurrentTemp.GetValue() != 23.5 {
		t.Errorf("invalid temperature expected:%v , actual:%v", 23.5, room1.CurrentTemp)
	}
	if room1.TargetTemp.GetValue() != 22. {
		t.Errorf("invalid temperature expected:%v , actual:%v", 22., room1.TargetTemp)
	}
	if room1.RunMode != RunModeProg {
		t.Errorf("invalid runmod expected:%v , actual:%v", RunModeProg, room1.RunMode)
	}

	room2 := (*rooms)[1]
	if room2.Id != 91234 {
		t.Errorf("invalid id expected:%d , actual:%d", 91234, room2.Id)
	}
	if room2.Name != "Room2" {
		t.Errorf("invalid name expected:%v , actual:%v", "Room2", room2.Name)
	}

}
