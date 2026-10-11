package pact

import (
	"net/http"
	"testing"
)

func TestDispatcher(t *testing.T) {

	dispatcher := new(Dispatcher)
	dispatcher.Register("health", "check", func(payload string) (any, error) {
		return ParsePayload[string](payload)
	})

	http.HandleFunc("/test/{resource}/{action}", dispatcher.Dispatch())

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
