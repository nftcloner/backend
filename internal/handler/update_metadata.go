package handler

import "net/http"

func UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	response(w, http.StatusOK, M{"ok": "ok"})
}
