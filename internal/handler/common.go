package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
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

func errorResponse(w http.ResponseWriter, statusCode int, err error, fields ...logrus.Fields) {
	if len(fields) > 0 {
		logrus.WithFields(fields[0]).Error(err)
	} else {
		logrus.Error(err)
	}
	response(w, statusCode, M{"error": err.Error()})
}
