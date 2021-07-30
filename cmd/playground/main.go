package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ariary/AravisFS/pkg/ubac"
	"github.com/golang/gddo/httputil/header"
)

//HELPER

type malformedRequest struct {
	status int
	msg    string
}

func (mr *malformedRequest) Error() string {
	return mr.msg
}

// Use to mitigate the Decoder limitation (error handling ,etc)
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &malformedRequest{status: http.StatusUnsupportedMediaType, msg: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576) //limit the size of the request body

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // exclude extra unexpected fields in clientJSON

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, msg: msg}

		default:
			return err
		}
	}
	//Decode accept {..JSON1..}{..JSON2..} --> avoid it
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{status: http.StatusBadRequest, msg: msg}
	}

	return nil
}

///// REMOTE PKG (SHARED )
type BodyLs struct {
	ResourceName string `json:"name"`
}

func createBodyLs(path string, resourceName string) BodyLs {

	b := &BodyLs{
		ResourceName: resourceName}
	return *b
}

///// UBAC
//return all available endpoints
func Endpoints(w http.ResponseWriter, r *http.Request) {
	endpoints := "endpoints\n"
	endpoints += "ls\n"
	fmt.Fprintf(w, endpoints)
}

// LS PART

// handler for ls function. Waiting request with JSON body with this structure {"name":"..."}
// where name is the name of the resource on which we apply the ls
// test it example:
// curl http://127.1:4444/ls -H "Content-Type: application/json" --request POST --data '{"name":"AAAAAAAAAAAAAAAA6ihdrw4ttG+sj+eQMnlA237KVk6le4+fERV81xq4dTDo0PnkM3M="}'
func RemoteLs(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//like this we could use path in RemoteLs Handler
		fmt.Println(path)

		var body BodyLs

		err := decodeJSONBody(w, r, &body)
		if err != nil {
			var mr *malformedRequest
			if errors.As(err, &mr) {
				http.Error(w, mr.msg, mr.status)
			} else {
				log.Println(err.Error())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		content, resourceType, err := ubac.Ls(body.ResourceName, path)
		if err != nil {
			fmt.Fprintf(w, err.Error())
			return
		}

		response := resourceType + ":" + content
		fmt.Fprintf(w, response)
	}
}

//Start an http server waiting for action request over encrypted fs (pointed by path)
func UbacListen(port int, path string) {
	mux := http.NewServeMux()
	//Add handlers
	mux.HandleFunc("/endpoints", Endpoints)
	mux.HandleFunc("/ls", RemoteLs(path)) //ls

	log.Println("Waiting for remote command over encrypted fs (", path, ") on port", port, ":...")
	err := http.ListenAndServe(":"+strconv.Itoa(port), mux)
	log.Fatal(err)
}
func main() {
	UbacListen(4444, "test/arafs/encrypted.arafs")
}
