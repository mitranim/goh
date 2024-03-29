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
  goh.HandleErr = writeErrAsJson
}

// Example custom error handler.
// Should be provided to response types as `ErrFunc: writeErrAsJson`.
func writeErrAsJson(
  rew http.ResponseWriter, req *http.Request, err error, wrote bool,
) {
  if err == nil {
    return
  }

  if !wrote {
    rew.WriteHeader(http.StatusInternalServerError)
    err := json.NewEncoder(rew).Encode(ErrJson{err})
    if err != nil {
      fmt.Fprintf(os.Stderr, "secondary error while writing error response: %+v\n", err)
      return
    }
  }

  fmt.Fprintf(os.Stderr, "error while writing HTTP response: %+v\n", err)
}
```

## Changelog

### `v0.1.11`

Breaking:

* Renamed `.MaybeHan` to `.HanOpt` in various types.

Added:

* `File.ServedHTTP`.
* `File.Exists`.
* `File.Existing`.
* `Dir.ServedHTTP`.
* `Dir.Resolve`.
* `Dir.Allow`.
* `Dir.File`.
* `HttpHandlerOpt` (implemented by `File` and `Dir`).

Fixed:

* `File` no longer writes its `.Header` when the target file is not found.

### `v0.1.10`

`Json` and `Xml` now support pretty-printing via field `.Indent`.

### `v0.1.9`

Cosmetic renaming and minor cleanup. Renamed `ErrHandlerDefault` → `HandleErr`, `ErrHandler` → `WriteErr`, tweaked argument order in `ErrFunc`, tweaked default error handling in `WriteErr`, tweaked error messages.

### `v0.1.8`

Lexicon change: "Res" → "Han" for functions that return `http.Handler`.

Add `TryJsonBytes`.

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
