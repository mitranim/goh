## Overview

`goh` = **Go** **H**TTP **H**andlers.

Utility types that represent a not-yet-sent HTTP response as a value (status, header, body) with _no added abstractions or interfaces_. All types implement `http.Hander`.

See API docs at https://pkg.go.dev/github.com/mitranim/goh.

## Usage

```golang
import "github.com/mitranim/goh"

// Simple example. Implicitly uses default error handler.
func handler(rew http.ResponseWriter, req *http.Request) {
  res := goh.StringOk("response text")
  res.ServeHTTP(rew, req)
}

// Simple example with status and headers.
// Implicitly uses default error handler.
func handler(rew http.ResponseWriter, req *http.Request) {
  res := goh.String{
    Status: http.StatusCreated,
    Header: http.Header{"Content-Type": {"text/html"}},
    Body:   "<body>response text</body>",
  }
  res.ServeHTTP(rew, req)
}

// Simple JSON example.
// Implicitly uses default error handler.
func handler(rew http.ResponseWriter, req *http.Request) {
  res := goh.JsonOk(Val{10, 20})
  res.ServeHTTP(rew, req)
}

// Example with custom error handler.
func handler(rew http.ResponseWriter, req *http.Request) {
  res := goh.Json{
    Body: Val{10, 20},
    ErrFunc: writeErrAsJson,
  }
  res.ServeHTTP(rew, req)
}

// Example with custom error handler.
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

// Can replace the default error handler globally. Don't abuse this power.
func init() {
  goh.ErrHandlerDefault = writeErrAsJson
}

type Val struct {
  One int64 `json:"one"`
  Two int64 `json:"two"`
}

type ErrJson struct {
  Error error `json:"error"`
}
```

## Changelog

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
