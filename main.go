package main

import (
	"fmt"
	"encoding/xml"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

    "github.com/gorilla/mux"
)

/* Example request

curl -X POST "http://webservice.correios.com.br:80/service/rastro" \
-H "Content-Type: text/xml;charset=UTF-8" \
-H "SOAPAction: \"buscaEventosLista\"" \
-d "<?xml version=\"1.0\" encoding=\"UTF-8\"?><SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:ns1=\"http://resource.webservice.correios.com.br/\"><SOAP-ENV:Body><ns1:buscaEventosLista><usuario>ECT</usuario><senha>SRO</senha><tipo>L</tipo><resultado>T</resultado><lingua>101</lingua><objetos>TTE123XXX</objetos></ns1:buscaEventosLista></SOAP-ENV:Body></SOAP-ENV:Envelope>"

*/

const SOAP_URL = "http://webservice.correios.com.br:80/service/rastro"

const SOAP_ENVELOPE = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ns1="http://resource.webservice.correios.com.br/">
	<SOAP-ENV:Body>
		<ns1:buscaEventosLista>
			<usuario>ECT</usuario>
			<senha>SRO</senha>
			<tipo>L</tipo>
			<resultado>T</resultado>
			<lingua>101</lingua>
			<objetos>%s</objetos>
		</ns1:buscaEventosLista>
	</SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

type Evento struct {
	Data string `xml:"data" json:"data"`
	Hora string `xml:"hora" json:"hora"`
	Descricao string `xml:"descricao" json:"descricao"`
}

type Objeto struct {
	Numero string `xml:"numero" json:"numero"`
	Erro string `xml:"erro" json:"erro"`
	Nome string `xml:"nome" json:"nome"`
	Evento *Evento `xml:"evento" json:"evento"`
}

type SoapResponse struct {
	XMLName xml.Name
	Objeto *Objeto `xml:"return>objeto" json:"objeto"`
}

type SoapFault struct {
	Faultcode string `xml:"faultcode" json:"faultcode"`
	Faultstring string `xml:"faultstring" json:"faultstring"`
	Detail string `xml:"detail" json:"detail"`
}

type SoapBody struct {
	Fault SoapFault
	Response SoapResponse `xml:"buscaEventosListaResponse"`
}

type SoapEnvelope struct {
	XMLName xml.Name
	Body SoapBody
}

func GetSoapEnvelope(trackingNumber string) (*SoapEnvelope, error) {
	soapRequestContent := fmt.Sprintf(SOAP_ENVELOPE, trackingNumber)
	timeout := time.Duration(30 * time.Second)
	httpClient := &http.Client{
		Timeout: timeout,
	}
	resp, err := httpClient.Post(SOAP_URL, "text/xml; charset=utf-8", bytes.NewBufferString(soapRequestContent))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		return nil, e
	}
	in := string(b)
	parser := xml.NewDecoder(bytes.NewBufferString(in))
	envelope := new(SoapEnvelope)
	err = parser.DecodeElement(&envelope, nil)
	if err != nil {
		return nil, err
	}
	return envelope, nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
    respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    response, _ := json.Marshal(payload)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
}

type RetTrackingNumbers struct {
	Objetos []*Objeto `json:"objetos"`
}

func doGetTrackingNumbers(trackingNumbers []string) (*RetTrackingNumbers, error) {
	ret := new(RetTrackingNumbers)

	length := len(trackingNumbers)
	if length > 0 {
		codes := make(chan bool, length)

		var err error
		mutexRet := &sync.Mutex{}
		mutexErr := &sync.Mutex{}
		for _, cod := range trackingNumbers {
			go func(trackingNumber string) {
				var errSoap error
				if err == nil {
					var env *SoapEnvelope
					env, errSoap = GetSoapEnvelope(trackingNumber)
					if errSoap == nil {
						if env.Body.Fault.Faultstring != "" {
							errSoap = fmt.Errorf("Faultcode: %s. Faultstring: %s. Detail: %s", env.Body.Fault.Faultcode, env.Body.Fault.Faultstring, env.Body.Fault.Detail)
						} else {
							mutexRet.Lock()
							ret.Objetos = append(ret.Objetos, env.Body.Response.Objeto)
							mutexRet.Unlock()
						}
					}
				}
				mutexErr.Lock()
				err = errSoap
				mutexErr.Unlock()
				codes <- true
			}(cod)
		}

		for i := 0; i < length; i++ {
			<-codes
		}

		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func GetTrackingNumbers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackingNumbers := strings.Split(strings.TrimSpace(vars["tracking_numbers"]), ";")

	ret, err := doGetTrackingNumbers(trackingNumbers)

	if err != nil {
		log.Printf("GetTrackingNumbers: %v\n", err.Error())
		respondWithError(w, http.StatusInternalServerError,  "Error retrieving tracking numbers.")
	} else {
		respondWithJSON(w, http.StatusOK, ret);
	}
}

func main() {
    router := mux.NewRouter()
    router.HandleFunc("/tracking_numbers/{tracking_numbers}", GetTrackingNumbers).Methods("GET")

    log.Fatal(http.ListenAndServe(":8000", router))
}
