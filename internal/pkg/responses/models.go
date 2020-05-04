package responses

type HttpError struct {
	Message string `json:"message"`
}

type HttpResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []HttpError `json:"errors"`
}
