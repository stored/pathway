package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
)

const (
	defaultBaseURL = "https://your-internal-service/"
)

type service struct {
	client *Client
}

type EchoService service

type APICall struct {
	Payload json.RawMessage
}

func dumpRequest(r *http.Request) {
	output, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println("Error dumping request:", err)
		return
	}
	log.Println(string(output))
}

func trimLeftChar(s string) string {
	for i := range s {
		if i > 0 {
			return s[i:]
		}
	}
	return s[:0]
}

func translatePathToMethod(m string) string {
	methodStartRe := regexp.MustCompile("[a-z]+")
	methodPartsRe := regexp.MustCompile("-[a-z]+")
	var methodBuffer bytes.Buffer
	methodStart := methodStartRe.FindString(m)
	methodBuffer.WriteString(strings.Title(methodStart))
	methodParts := methodPartsRe.FindAllString(m, len(methodStart)-1)
	for i, methodPart := range methodParts {
		methodParts[i] = strings.Title(trimLeftChar(methodPart))
	}
	methodBuffer.WriteString(strings.Join(methodParts, ""))
	return methodBuffer.String()
}

func APIGateway(w http.ResponseWriter, request *http.Request) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	apiCall := new(APICall)
	if err := json.Unmarshal(body, &apiCall.Payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	methodParts := strings.Split(mux.Vars(request)["method"], "/")
	resource := strings.Title(methodParts[0])
	method := translatePathToMethod(methodParts[1])
	apiClient := NewClient()
	data, _, err := apiClient.CallMethodByName(resource, method, apiCall.Payload)
	parsedData, err := json.Marshal(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	log.Println(string(parsedData))
	io.WriteString(w, string(parsedData))
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"alive": true}`)
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/health", HealthCheckHandler)
	router.HandleFunc("/api/{method:.*}", APIGateway)
	log.Println("Running API gateway at port 4000")
	log.Fatal(http.ListenAndServe(":4000", router))
}
