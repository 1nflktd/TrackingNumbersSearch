package main

import (
	"fmt"
	"encoding/xml"
	"net/http"
	"bytes"
	"io/ioutil"
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
	Data string `xml:"data"`
	Hora string `xml:"hora"`
	Descricao string `xml:"descricao"`
}

type Objeto struct {
	Numero string `xml:"numero"`
	Erro string `xml:"erro"`
	Nome string `xml:"nome"`
	Evento Evento `xml:"evento"`
}

type SoapResponse struct {
	XMLName xml.Name
	Objeto Objeto `xml:"return>objeto"`
}

type SoapFault struct {
	Faultcode string `xml:"faultcode"`
	Faultstring string `xml:"faultstring"`
	Detail string `xml:"detail"`
}

type SoapBody struct {
	Fault SoapFault
	Response SoapResponse `xml:"buscaEventosListaResponse"`
}

type SoapEnvelope struct {
	XMLName xml.Name
	Body SoapBody
}

func GetSoapEnvelope(codigoRastreamento string) (*SoapEnvelope, error) {
	soapRequestContent := fmt.Sprintf(SOAP_ENVELOPE, codigoRastreamento)
	httpClient := new(http.Client)
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

func consultarCodigosRastreamento(codigosRastreamento []string) {
	codigos := make(chan bool, len(codigosRastreamento))

	for _, cod := range codigosRastreamento {
		go func(codigoRastreamento string) {
			env, err := GetSoapEnvelope(codigoRastreamento)
			if err == nil {
				fmt.Printf("Numero: %s\n Erro: %s Evento: %v\n",
					env.Body.Response.Objeto.Numero,
					env.Body.Response.Objeto.Erro,
					env.Body.Response.Objeto.Evento)
			} else {
				fmt.Printf("Erro ao obter codigo de rastreamento %s: %s\n", codigoRastreamento, err)
			}
			codigos <- true
		}(cod)
	}

	for i := 0; i < len(codigosRastreamento); i++ {
		<-codigos
	}
}

func main() {
	codigosRastreamento := []string{
		"XFASDASDXX",
		"cvaSDF23123",
	}

	consultarCodigosRastreamento(codigosRastreamento)
}
