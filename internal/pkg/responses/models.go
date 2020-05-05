package responses

import "net/http"

type HttpError struct {
	Message string `json:"message"`
}

type HttpResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []HttpError `json:"errors"`
}

type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func (rec *StatusRecorder) WriteHeader(code int) {
	rec.Status = code
	rec.ResponseWriter.WriteHeader(code)
}
