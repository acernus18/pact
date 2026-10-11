package pact

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func writeSystemError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Response[any]{Code: -1, Message: "System Error", Data: nil})
}

func writeError(w http.ResponseWriter, err error) {
	elems := strings.Split(err.Error(), ":")
	if len(elems) == 2 {
		code, e := strconv.Atoi(elems[0])
		if e == nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(Response[any]{
				Code:    code,
				Message: elems[1],
				Data:    nil,
			})
		} else {
			writeSystemError(w)
		}
	}
}
