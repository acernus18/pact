# pact

A tiny Go library that maps HTTP requests onto named handlers.

`pact` turns a single POST route of the shape `/{resource}/{action}` into a
registry lookup: you register a handler per `resource` + `action` pair, and the
dispatcher decodes the body, calls the handler and writes back a uniform JSON
envelope. There are no dependencies beyond the standard library.

- Route by `resource` / `action` instead of one path per endpoint
- One response envelope for every endpoint (`code`, `message`, `data`)
- Generic payloads via type parameters
- Business errors carried as an application-level code, not an HTTP status

## Install

```bash
go get github.com/acernus18/pact
```

Requires Go 1.26 or newer.

## Quick start

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/acernus18/pact"
)

func main() {
	dispatcher := new(pact.Dispatcher)

	dispatcher.Register("user", "get", func(payload string) (any, error) {
		// Decode the incoming body: {"token": "...", "data": {"id": 1}}
		request, err := pact.ParsePayload[struct {
			ID int `json:"id"`
		}](payload)
		if err != nil {
			return nil, err
		}
		if request.Data.ID == 0 {
			// Business errors use the "code:message" convention.
			return nil, fmt.Errorf("40001:invalid user id")
		}
		return map[string]any{"id": request.Data.ID, "name": "Ada"}, nil
	})

	http.HandleFunc("/api/{resource}/{action}", dispatcher.Dispatch())
	http.ListenAndServe(":8080", nil)
}
```

Call it:

```bash
curl -X POST http://localhost:8080/api/user/get \
  -H 'Content-Type: application/json' \
  -d '{"token":"","data":{"id":1}}'
```

```json
{
  "code": 0,
  "message": "SUC",
  "data": {
    "id": 1,
    "name": "Ada"
  }
}
```

## How a request is handled

1. Anything other than `POST` is rejected with a system error.
2. `resource` and `action` are read from the path with `r.PathValue`, so the
   route you register must declare both placeholders, e.g.
   `/api/{resource}/{action}`.
3. The whole body is read and handed to the handler as a `string`.
4. The handler returns `(any, error)`.
    - No error: `200 OK` with `{"code":0,"message":"SUC","data":<result>}`.
    - Error shaped like `"<code>:<message>"`: `200 OK` with that `code` and
      `message`.
    - Any other error: `200 OK` with `{"code":-1,"message":"System Error"}`.

Every response is HTTP `200` with `Content-Type: application/json`; the outcome
lives in the body's `code` field (`0` = success, `-1` = system error).

## API

### Types

```go
// The response envelope written for every request.
type Response[T any] struct {
Code    int    `json:"code"`
Message string `json:"message"`
Data    T      `json:"data"`
}

// The envelope expected in a request body.
type Payload[T any] struct {
Token string `json:"token"`
Data  T      `json:"data"`
}
```

Wire keys are lowercase: `code`, `message`, `data` and `token`.

### `ParsePayload[T any](message string) (Payload[T], error)`

Unmarshals a request body into `Payload[T]`. It decodes only — it does not
validate the token or reject unknown fields.

### `Close(closer io.Closer, handler func(error))`

Closes `closer` and calls `handler` only if closing fails, so the common
`defer`-based cleanup of response bodies, files and sockets stays a one-liner.

### `type Responder = func(payload string) (any, error)`

The handler signature: raw request body in, response `Data` (or an error) out.

### `Dispatcher`

```go
func (d *Dispatcher) Register(resource, action string, responder Responder)
func (d *Dispatcher) Dispatch() func (w http.ResponseWriter, r *http.Request)
```

- `Register` adds or replaces the handler for a `resource`/`action` pair. The
  registry is created lazily, so the zero value `new(Dispatcher)` (or
  `var d Dispatcher`) is ready to use.
- `Dispatch` returns the `http.HandlerFunc` to mount on your mux. Requests for
  an unregistered pair produce a system error.

Pass the returned function to `http.HandleFunc` / `http.Handle` as in the
example above.

## Error convention

Business errors are strings of exactly two colon-separated parts:

```go
return nil, fmt.Errorf("40001:user not found")
// -> {"code":40001,"message":"user not found","data":null}
```

The message is everything between the first and the second colon; a value with
more or fewer than two parts (including a message that itself contains `:`) is
treated as a malformed error and falls back to the generic system error.

## Layout

Everything lives in a single file, `pact.go`, split into two blocks by the
`// Common Utils` comment:

| Block        | Contents                                                                           |
|--------------|------------------------------------------------------------------------------------|
| Core         | `Response`, `Payload`, `Responder`, `Dispatcher`, `writeError`, `writeSystemError` |
| Common Utils | `ParsePayload`, `Close`                                                            |

## Publish Process

### Add tag

```
git tag v0.0.1
git push origin v0.0.1

Total 0 (delta 0), reused 0 (delta 0), pack-reused 0 (from 0)
To github.com:acernus18/pact.git
 * [new tag]         v0.0.1 -> v0.0.1
```

### Notify Go to cache it

```
git config --global http.proxy http://127.0.0.1:7890
git config --global https.proxy http://127.0.0.1:7890

go list -m github.com/acernus18/pact@v0.0.1

git config --global --unset http.proxy
git config --global --unset https.proxy
git config --global --list
```
