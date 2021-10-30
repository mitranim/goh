## Overview

`goh` = **Go** **H**TTP **H**andlers.

Utility types that represent a not-yet-sent HTTP response as a value (status, header, body) with _no added abstractions_. All types implement `http.Hander`.

Recommended in conjunction with [`github.com/mitranim/rout`](https://github.com/mitranim/rout): router with support for returning responses as `http.Handler`.

Small and dependency-free.

See API docs at https://pkg.go.dev/github.com/mitranim/goh.

## Usage

```golang
import "github.com/mitranim/goh"

type Val struct {
  One int64 `json:"one"`
  Two int64 `json:"two"`
}

type ErrJson struct {
  Error error `json:"error"`
}

// Simple string example.
func handler(req *http.Request) http.Handler {
  return goh.StringOk(`response text`)
}

// String example with status and headers.
func handler(req *http.Request) http.Handler {
  return goh.String{
    Status: http.StatusCreated,
    Header: http.Header{`Content-Type`: {`text/html`}},
    Body:   `<body>response text</body>`,
  }
}

// Simple JSON example.
func handler(req *http.Request) http.Handler {
  return goh.JsonOk(Val{10, 20})
}

// JSON example with custom error handler.
func handler(req *http.Request) http.Handler {
  return goh.Json{
    Body: Val{10, 20},
    ErrFunc: writeErrAsJson,
  }
}

// You can customize the default error handler.
func init() {
  goh.ErrHandlerDefault = writeErrAsJson
}

// Example custom error handler.
// Should be provided to response types as `ErrFunc: writeErrAsJson`.
func writeErrAsJson(
  rew http.ResponseWriter, req *http.Request, wrote bool, err error,
) {
  if err == nil {
    return
  }

  if !wrote {
    rew.WriteHeader(http.StatusInternalServerError)
    err = json.NewEncoder(rew).Encode(ErrJson{err})
    if err != nil {
      // Logged below.
      err = fmt.Errorf(`secondary error while writing error response: %w`, err)
    }
  }

  if err != nil {
    fmt.Fprintf(os.Stderr, "%+v\n", err)
  }
}
```

## Changelog

### `v0.1.7`

Added file-serving facilities:

  * `File`
  * `Dir`
  * `Filter`
  * `FilterFunc`
  * `AllowDors`

This provides richer file-serving functionality than `net/http`, including the ability to serve paths from a folder selectively, or being able to "try file" and fall back on something else.

### `v0.1.6`

`Json.TryBytes` and `Xml.TryBytes` no longer panic on nil header.

### `v0.1.5`

Added `Json.TryBytes` and `Xml.TryBytes` for pre-encoding static responses.

### `v0.1.4`

Added `.Res()` methods for request → response signatures.

### `v0.1.3`

Added `Err`, `Handler`, `Respond`.

### `v0.1.2`

`Redirect` no longer writes the HTTP status before invoking `http.Redirect`.

### `v0.1.1`

Optional support for `<?xml?>`.

## License

https://unlicense.org

## Misc

I'm receptive to suggestions. If this library _almost_ satisfies you but needs changes, open an issue or chat me up. Contacts: https://mitranim.com/#contacts
