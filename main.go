package p

import (
	"net/http"

	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/sirupsen/logrus"

	"github.com/nftcloner/backend/internal/handler"
)

var mux = newMux()

func Entrypoint(w http.ResponseWriter, r *http.Request) {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetLevel(logrus.TraceLevel)

	mux.ServeHTTP(w, r)
}

func newMux() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("/metadata/update", handler.UpdateMetadata)
	return m
}
