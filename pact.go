package pact

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func writeSystemError(w http.ResponseWriter) {
	// [BugFix]: add header before write
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Response[any]{Code: -1, Message: "System Error", Data: nil})
}

func writeError(w http.ResponseWriter, err error) {
	elems := strings.Split(err.Error(), ":")
	if len(elems) == 2 {
		code, e := strconv.Atoi(elems[0])
		if e == nil {
			// [BugFix]: add header before write
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(Response[any]{
				Code:    code,
				Message: elems[1],
				Data:    nil,
			})
		} else {
			writeSystemError(w)
		}
	} else {
		// [BugFix]: return system error if Error format is not correct
		writeSystemError(w)
	}
}

type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type Payload[T any] struct {
	Token string `json:"token"`
	Data  T      `json:"data"`
}

type Responder = func(payload string) (any, error)

type Dispatcher struct {
	registry map[string]map[string]Responder
}

func (dispatcher *Dispatcher) Register(resource, action string, responder Responder) {
	if dispatcher.registry == nil {
		dispatcher.registry = make(map[string]map[string]Responder)
	}
	resourceResponders, present := dispatcher.registry[resource]
	if !present {
		resourceResponders = make(map[string]Responder)
	}
	resourceResponders[action] = responder
	dispatcher.registry[resource] = resourceResponders
}

func (dispatcher *Dispatcher) Dispatch() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeSystemError(w)
			return
		}
		resource := r.PathValue("resource")
		action := r.PathValue("action")
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			writeSystemError(w)
			return
		}
		responder, present := dispatcher.registry[resource][action]
		if !present {
			writeSystemError(w)
			return
		}
		response, err := responder(string(payload))
		if err != nil {
			writeError(w, err)
			// [BugFix]: didnt return after writeError
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if e := json.NewEncoder(w).Encode(Response[any]{
			Code: 0, Message: "SUC", Data: response,
		}); e != nil {
			writeSystemError(w)
		}
	}
}

// Common Utils

func Close(closer io.Closer, handler func(error)) {
	if err := closer.Close(); err != nil {
		handler(err)
	}
}

func ParsePayload[T any](message string) (Payload[T], error) {
	var result Payload[T]
	err := json.Unmarshal([]byte(message), &result)
	return result, err
}

// Database Utils

type IScanner interface {
	Scan(rows *sql.Rows) error
}

func Query[T any, PT interface {
	*T
	IScanner
}](db *sql.DB, stmt string, args ...any) ([]T, error) {
	rows, queryErr := db.Query(stmt, args...)
	if queryErr != nil {
		return nil, queryErr
	}
	defer Close(rows, func(e error) {
		log.Printf("Error closing rows - %s : %v", stmt, e)
	})
	result := make([]T, 0)
	for rows.Next() {
		var item T
		if err := PT(&item).Scan(rows); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
