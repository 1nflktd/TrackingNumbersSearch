package main_test

import (
    "testing"
    "net/http"
    "net/http/httptest"
    "os"
    "encoding/json"

    "github.com/gorilla/mux"

    "."
)

var testServeMux *mux.Router

func TestMain(m *testing.M) {
    testServeMux = mux.NewRouter()
    testServeMux.HandleFunc("/tracking_numbers/{tracking_numbers}", main.GetTrackingNumbers).Methods("GET")

    code := m.Run()

    os.Exit(code)
}

func executeRequest(req *http.Request, h http.Handler) *httptest.ResponseRecorder {
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)

    return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
    if expected != actual {
        t.Errorf("Expected response code %d. Got %d\n", expected, actual)
    }
}

func TestGetTrackingNumbersInvalidNumber(t *testing.T) {
    req, _ := http.NewRequest("GET", "/tracking_numbers/XX123CC", nil)
    response := executeRequest(req, testServeMux)

    checkResponseCode(t, http.StatusOK, response.Code)

    var retBody main.RetTrackingNumbers
    err := json.Unmarshal(response.Body.Bytes(), &retBody)
    if err != nil {
        t.Errorf(err.Error())
        return
    }

    if retBody.Objetos[0].Erro == "" {
        t.Errorf("Expecting error message.")
    }
}

func TestGetTrackingNumbersCorrect(t *testing.T) {
    req, _ := http.NewRequest("GET", "/tracking_numbers/XX1233FF;GGW122;BBBF;PN848933136BR", nil)
    response := executeRequest(req, testServeMux)

    checkResponseCode(t, http.StatusOK, response.Code)

    var retBody main.RetTrackingNumbers
    err := json.Unmarshal(response.Body.Bytes(), &retBody)
    if err != nil {
        t.Errorf(err.Error())
        return
    }

    for _, ret := range retBody.Objetos {
        if ret.Numero == "PN848933136BR"{
            if ret.Evento == nil {
                t.Errorf("Expecting evento.")
            }
            break
        }
    }


}
