package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type M map[string]interface{}

func response(w http.ResponseWriter, statusCode int, data interface{}) {
	responseB, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("error marshalling response: %v", err)))
		return
	}

	w.WriteHeader(statusCode)
	_, _ = w.Write(responseB)
}
