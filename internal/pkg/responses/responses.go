package responses

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"net/http"
)

func SendServerError(errorMessage string, ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(http.StatusInternalServerError)
	fmt.Printf("{level: error, message: %s}", errors.New(errorMessage))
}

func SendResponse(code int, data interface{}, ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(code)

	serializedData, err := json.Marshal(data)
	if err != nil {
		SendServerError(err.Error(), ctx)
		return
	}
	ctx.SetBody(serializedData)
}

func SendResponseOK(data interface{}, ctx *fasthttp.RequestCtx) {
	SendResponse(200, data, ctx)
}
