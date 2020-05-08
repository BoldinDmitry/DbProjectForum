package responses

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func SendServerError(errorMessage string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Printf("{level: error, message: %s}", errors.New(errorMessage))
}

func SendResponse(code int, data interface{}, w http.ResponseWriter) {
	w.WriteHeader(code)

	serializedData, err := json.Marshal(data)
	if err != nil {
		SendServerError(err.Error(), w)
		return
	}
	_, err = w.Write(serializedData)
	if err != nil {
		SendServerError(err.Error(), w)
		return
	}
}

func SendResponseOK(data interface{}, w http.ResponseWriter) {
	SendResponse(200, data, w)
}
