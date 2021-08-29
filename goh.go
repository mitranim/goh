/*
goh = Go Http Handlers.

Utility types that represent a not-yet-sent HTTP response as a value
(status, header, body) with NO added abstractions or interfaces. All types
implement `http.Hander`.

See `readme.md` for examples.
*/
package goh

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

/*
Signature of a "request->response" function. All Goh handler types have a method
`.Res` that conforms to this signature.
*/
type ResFunc = func(*http.Request) http.Handler

/*
Signature of an error handler function provided by user code to the various
`http.Handler` types in this library, such as `String`.

When nil, `ErrHandlerDefault` is used.

The `wrote` parameter indicates whether the response writer has been written to.
If `false`, the handler should write an error response. If `true`, or if
sending the error response has failed, the handler should log the resulting
error to the server log.
*/
type ErrFunc = func(rew http.ResponseWriter, req *http.Request, wrote bool, err error)

/*
Default error handler, used by `http.Handler` types when no `ErrFunc` was
provided. May be overridden globally.
*/
var ErrHandlerDefault = ErrHandler

/*
Default implementation of `ErrFunc`. Used by `http.Handler` types, such as
`String`, when no `ErrFunc` was provided by user code.

If possible, writes the error to the response writer as plain text. If not, logs
the error to the standard error stream.

When implementing a custom error handler, use this function's source as an
example.
*/
func ErrHandler(rew http.ResponseWriter, _ *http.Request, wrote bool, err error) {
	if err == nil {
		return
	}

	if !wrote {
		rew.WriteHeader(http.StatusInternalServerError)
		_, err = fmt.Fprint(rew, err)
		if err != nil {
			// Logged below.
			err = fmt.Errorf(`secondary error while writing error response: %w`, err)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
}

/*
Makes an extremely simple `http.Handler` that serves the error's message as
plain text. The status is always 500.
*/
func Err(err error) String {
	return StringWith(http.StatusInternalServerError, errMsg(err))
}

/*
The head part of each `http.Handler` implementation in this package.
Pseudo-embedded in the various handler types, obtained via `.Head()` methods.

Goh uses pseudo-embedding instead of actual embedding because Go doesn't allow
promoted fields to be used at the top level of embedding type. Literals of
various handler types would have to use `Head: Head{}`.

See `Reader`, `Bytes`, `String`, `Json`, `Xml`.
*/
type Head struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
}

/*
Writes the header and HTTP status (if any) to the provided writer. Called
internally by the various handler types. You shouldn't have to call this
yourself, unless implementing a new type.

This must be called exactly once, and only before writing the body.
*/
func (self Head) Write(rew http.ResponseWriter) {
	self.writeHeaders(rew)
	if self.Status > 0 {
		rew.WriteHeader(self.Status)
	}
}

func (self Head) writeHeaders(rew http.ResponseWriter) {
	target := rew.Header()
	for key, vals := range self.Header {
		target.Del(key)
		for _, val := range vals {
			target.Add(key, val)
		}
	}
}

func (self Head) handleErr(rew http.ResponseWriter, req *http.Request, wrote bool, err error) {
	if err == nil {
		return
	}
	if self.ErrFunc == nil {
		self.ErrFunc = ErrHandlerDefault
	}
	self.ErrFunc(rew, req, wrote, err)
}

/*
HTTP handler that copies a response from a reader.

Caution: if the reader is also `io.Closer`, it must be closed in your code.
This type does NOT attempt that.
*/
type Reader struct {
	Status  int
	Header  http.Header
	Body    io.Reader
	ErrFunc ErrFunc
}

// Implement `http.Handler`.
func (self Reader) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().Write(rew)

	if self.Body != nil {
		_, err := io.Copy(rew, self.Body)
		if err != nil {
			err = fmt.Errorf(`failed to copy response from reader: %w`, err)
			self.Head().handleErr(rew, req, true, err)
		}
	}
}

// Returns the pseudo-embedded `Head` part.
func (self Reader) Head() Head { return Head{self.Status, self.Header, self.ErrFunc} }

// Conforms to `ResFunc`.
func (self Reader) Res(*http.Request) http.Handler { return self }

/*
HTTP handler that writes bytes. Note: for sending a string, use `String`,
avoiding a bytes-to-string conversion.
*/
type Bytes struct {
	Status  int
	Header  http.Header
	Body    []byte
	ErrFunc ErrFunc
}

// Implement `http.Handler`.
func (self Bytes) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().Write(rew)

	_, err := rew.Write(self.Body)
	if err != nil {
		err = fmt.Errorf(`failed to write response bytes: %w`, err)
		self.Head().handleErr(rew, req, true, err)
	}
}

// Returns the pseudo-embedded `Head` part.
func (self Bytes) Head() Head { return Head{self.Status, self.Header, self.ErrFunc} }

// Conforms to `ResFunc`.
func (self Bytes) Res(*http.Request) http.Handler { return self }

// Shortcut for `BytesWith(http.StatusOK, body)`.
func BytesOk(body []byte) Bytes {
	return BytesWith(http.StatusOK, body)
}

// Shortcut for `Bytes` with specific status and body.
func BytesWith(status int, body []byte) Bytes {
	return Bytes{Status: status, Body: body}
}

/*
HTTP handler that writes a string. Note: for sending bytes, use `Bytes`,
avoiding a string-to-bytes conversion.
*/
type String struct {
	Status  int
	Header  http.Header
	Body    string
	ErrFunc ErrFunc
}

// Implement `http.Handler`.
func (self String) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().Write(rew)

	_, err := io.WriteString(rew, self.Body)
	if err != nil {
		err = fmt.Errorf(`failed to write response string: %w`, err)
		self.Head().handleErr(rew, req, true, err)
	}
}

// Returns the pseudo-embedded `Head` part.
func (self String) Head() Head { return Head{self.Status, self.Header, self.ErrFunc} }

// Conforms to `ResFunc`.
func (self String) Res(*http.Request) http.Handler { return self }

// Shortcut for `StringWith(http.StatusOK, body)`.
func StringOk(body string) String {
	return StringWith(http.StatusOK, body)
}

// Shortcut for `String` with specific status and body.
func StringWith(status int, body string) String {
	return String{Status: status, Body: body}
}

/*
HTTP handler that automatically sets the appropriate JSON headers and encodes
its body as JSON. Currently does not support custom encoder options; if you
need that feature, open an issue or PR.
*/
type Json struct {
	Status  int
	Header  http.Header
	Body    interface{}
	ErrFunc ErrFunc
}

// Implement `http.Handler`.
func (self Json) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	rew.Header().Set("Content-Type", "application/json")
	self.Head().Write(rew)

	writer := spyingWriter{Writer: rew}
	err := json.NewEncoder(&writer).Encode(self.Body)
	if err != nil {
		err = fmt.Errorf(`failed to write response as JSON: %w`, err)
		self.Head().handleErr(rew, req, writer.wrote, err)
	}
}

// Returns the pseudo-embedded `Head` part.
func (self Json) Head() Head { return Head{self.Status, self.Header, self.ErrFunc} }

// Conforms to `ResFunc`.
func (self Json) Res(*http.Request) http.Handler { return self }

/*
Converts to `Bytes` by encoding the body and adding the appropriate content type
header. Panics on encoding errors. Should be used in root scope to pre-encode a
static response:

	import "github.com/mitranim/goh"

	var someRes = goh.XmlOk(someValue).TryBytes()
*/
func (self Json) TryBytes() Bytes {
	body, err := json.Marshal(self.Body)
	if err != nil {
		panic(err)
	}

	head := self.Header.Clone()
	head.Set("Content-Type", "application/json")

	return Bytes{
		Status:  self.Status,
		Header:  head,
		Body:    body,
		ErrFunc: self.ErrFunc,
	}
}

// Shortcut for `JsonWith(http.StatusOK, body)`.
func JsonOk(body interface{}) Json {
	return JsonWith(http.StatusOK, body)
}

// Shortcut for `Json` with specific status and body.
func JsonWith(status int, body interface{}) Json {
	return Json{Status: status, Body: body}
}

/*
HTTP handler that automatically sets the appropriate XML headers and encodes
its body as XML. Currently does not support custom encoder options; if you
need that feature, open an issue or PR.

Caution: this does NOT prepend the processing instruction `<?xml?>`. When you
don't need to specify the encoding, this instruction is entirely skippable.
When you need to specify the encoding, wrap `.Body` in the utility type
`XmlDoc` provided by this package.
*/
type Xml Json

// Implement `http.Handler`.
func (self Xml) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	rew.Header().Set("Content-Type", "application/xml")
	self.Head().Write(rew)

	writer := spyingWriter{Writer: rew}
	err := xml.NewEncoder(&writer).Encode(self.Body)
	if err != nil {
		err = fmt.Errorf(`failed to write response as XML: %w`, err)
		self.Head().handleErr(rew, req, writer.wrote, err)
	}
}

// Returns the pseudo-embedded `Head` part.
func (self Xml) Head() Head { return Head{self.Status, self.Header, self.ErrFunc} }

// Conforms to `ResFunc`.
func (self Xml) Res(*http.Request) http.Handler { return self }

/*
Converts to `Bytes` by encoding the body and adding the appropriate content type
header. Panics on encoding errors. Should be used in root scope to pre-encode a
static response:

	import "github.com/mitranim/goh"

	var someRes = goh.XmlOk(someValue).TryBytes()
*/
func (self Xml) TryBytes() Bytes {
	body, err := xml.Marshal(self.Body)
	if err != nil {
		panic(err)
	}

	head := self.Header.Clone()
	head.Set("Content-Type", "application/xml")

	return Bytes{
		Status:  self.Status,
		Header:  head,
		Body:    body,
		ErrFunc: self.ErrFunc,
	}
}

// Shortcut for `XmlWith(http.StatusOK, body)`.
func XmlOk(body interface{}) Xml {
	return XmlWith(http.StatusOK, body)
}

// Shortcut for `Xml` with specific status and body.
func XmlWith(status int, body interface{}) Xml {
	return Xml{Status: status, Body: body}
}

/*
HTTP handler that performs an HTTP redirect.
*/
type Redirect struct {
	Status  int
	Header  http.Header
	Link    string
	ErrFunc ErrFunc
}

// Implement `http.Handler`.
func (self Redirect) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().writeHeaders(rew)
	http.Redirect(rew, req, self.Link, self.Status)
}

// Returns the pseudo-embedded `Head` part.
func (self Redirect) Head() Head { return Head{self.Status, self.Header, self.ErrFunc} }

// Conforms to `ResFunc`.
func (self Redirect) Res(*http.Request) http.Handler { return self }

// Shortcut for `Redirect` with specific status and body.
func RedirectWith(status int, link string) Redirect {
	return Redirect{Status: status, Link: link}
}

/*
Utility type for use together with `Xml`. When encoded as XML, this prepends the
`<?xml?>` header with version 1.0 and the specified encoding, if any. Example
usage:

	myXmlDoc := SomeType{SomeField: someValue}

	res := goh.XmlOk(goh.XmlDoc{
		Encoding: "utf-8",
		Val: myXmlDoc,
	})

Eventual output:

	<?xml version="1.0" encoding="utf-8"?>
	<SomeType ...>
*/
type XmlDoc struct {
	Encoding string
	Val      interface{}
}

func (self XmlDoc) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	inst := xmlVersionInst
	if self.Encoding != "" {
		inst = append(inst, ` encoding=`...)
		inst = strconv.AppendQuote(inst, self.Encoding)
	}

	err := enc.EncodeToken(xml.ProcInst{
		Target: `xml`,
		Inst:   inst,
	})
	if err != nil {
		return err
	}

	return enc.Encode(self.Val)
}

/*
Runs the provided function, returning the resulting `http.Handler`. Catches
panics and converts them to a simple error responder via `Err`.
*/
func Handler(fun func() http.Handler) (out http.Handler) {
	defer recHandler(&out)
	return fun()
}

/*
Shortcut for serving the response generated by the provided function. Catches
panics, serving the resulting errors as plain text via `Err`.
*/
func Respond(rew http.ResponseWriter, req *http.Request, fun func() http.Handler) {
	Handler(fun).ServeHTTP(rew, req)
}

var xmlVersionInst = []byte(`version="1.0"`)

type spyingWriter struct {
	io.Writer
	wrote bool
}

func (self *spyingWriter) Write(chunk []byte) (int, error) {
	self.wrote = true
	return self.Writer.Write(chunk)
}

func errMsg(err error) (msg string) {
	if err != nil {
		msg = err.Error()
	}
	if msg == `` {
		msg = `unknown error`
	}
	return
}

func recHandler(ptr *http.Handler) {
	val := recover()
	if val == nil {
		return
	}

	err, _ := val.(error)
	if err == nil {
		err = fmt.Errorf(`%v`, val)
	}

	*ptr = Err(err)
}
