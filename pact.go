package pact

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Response[T any] struct {
	Code    int
	Message string
	Data    T
}

type Payload[T any] struct {
	Token string
	Data  T
}

func ParsePayload[T any](message string) (Payload[T], error) {
	var result Payload[T]
	err := json.Unmarshal([]byte(message), &result)
	return result, err
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
		}
		w.Header().Set("Content-Type", "application/json")
		if e := json.NewEncoder(w).Encode(Response[any]{
			Code: 0, Message: "SUC", Data: response,
		}); e != nil {
			writeSystemError(w)
		}
	}
}
