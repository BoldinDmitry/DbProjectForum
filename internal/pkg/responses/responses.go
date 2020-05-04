package responses

import (
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"net/http"
)

func SendServerError(errorMessage string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	log.Err(errors.New(errorMessage))
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
