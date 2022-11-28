/*
goh = Go Http Handlers.

Utility types that represent a not-yet-sent HTTP response as a value
(status, header, body) with NO added abstractions. All types implement
`http.Hander`.

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
	"path/filepath"
	"strconv"
	"strings"
)

const (
	HeadType  = `Content-Type`
	TypeJson  = `application/json`
	TypeForm  = `application/x-www-form-urlencoded`
	TypeMulti = `multipart/form-data`
)

/*
Signature of a "request->response" function. All Goh handler types have a method
`.Han` that conforms to this signature.
*/
type Han = func(*http.Request) http.Handler

/*
Signature of an error handler function provided by user code to the various
`http.Handler` types in this library, such as `String`. When nil,
`goh.HandleErr` is used.

The `wrote` parameter indicates whether the response writer has been written to.
If `false`, the handler should write an error response. If `true`, or if
sending the error response has failed, the handler should log the resulting
error to the server log.
*/
type ErrFunc = func(rew http.ResponseWriter, req *http.Request, err error, wrote bool)

/*
Default error handler, used by various `http.Handler` types in this package when
no `.ErrFunc` was provided. May be overridden globally.
*/
var HandleErr = WriteErr

/*
Default implementation of `goh.ErrFunc`. Used by `http.Handler` types, such as
`goh.String`, when no `goh.ErrFunc` was provided by user code. If possible,
writes the error to the response writer as plain text. If not, logs the error
to the standard error stream. When implementing a custom error handler, use
this function's source as an example.
*/
func WriteErr(rew http.ResponseWriter, _ *http.Request, err error, wrote bool) {
	if err == nil {
		return
	}

	if !wrote {
		rew.WriteHeader(http.StatusInternalServerError)
		_, inner := io.WriteString(rew, err.Error())
		if inner == nil {
			return
		}

		fmt.Fprintf(
			os.Stderr,
			"unexpected error while writing HTTP response: %+v\n"+
				"unexpected secondary error while writing error response: %+v\n",
			err, inner,
		)
		return
	}

	fmt.Fprintf(os.Stderr, "unexpected error while writing HTTP response: %+v\n", err)
}

/*
Makes an extremely simple `http.Handler` that serves the error's message as
plain text. The status is always 500.
*/
func Err(err error) String {
	return StringWith(http.StatusInternalServerError, errMsg(err))
}

/*
Shortcut for `goh.JsonOk(val).TryBytes()`. Should be used for pre-encoded
handlers defined as global variables. Should NOT be used for
dynamically-generated responses.
*/
func TryJsonBytes(val interface{}) Bytes { return JsonOk(val).TryBytes() }

/*
The head part of each `http.Handler` implementation in this package.
Pseudo-embedded in the various handler types, obtained via `.Head()` methods.

Goh uses pseudo-embedding instead of actual embedding because Go doesn't allow
promoted fields to be used at the top level of embedding type. With embedding,
literals of various handler types would have to use `Head: Head{}`.
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

	/**
	The status `http.StatusOK` is implicit, and writing it should be equivalent to
	writing no status at all. However, there are some unwanted "smarts" inside
	the Go HTTP library, where writing status 200 suppresses the writing of
	default HEADERS following it. One example is `http.ServeFile`.
	*/
	if self.Status != 0 && self.Status != http.StatusOK {
		rew.WriteHeader(self.Status)
	}
}

func (self Head) writeHeaders(rew http.ResponseWriter) {
	target := rew.Header()
	for key, vals := range self.Header {
		target[key] = vals
	}
}

func (self Head) errFunc() ErrFunc {
	if self.ErrFunc != nil {
		return self.ErrFunc
	}
	return HandleErr
}

/*
HTTP handler that copies a response from a reader.

Caution: if the reader is also `io.Closer`, it must be closed in your code.
This type does NOT attempt that.
*/
type Reader struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
	Body    io.Reader
}

// Returns the pseudo-embedded `goh.Head` part.
func (self Reader) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self Reader) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().Write(rew)

	if self.Body != nil {
		_, err := io.Copy(rew, self.Body)
		if err != nil {
			err = fmt.Errorf(`[goh] failed to copy response from reader: %w`, err)
			self.Head().errFunc()(rew, req, err, true)
		}
	}
}

// Conforms to `goh.Han`.
func (self Reader) Han(*http.Request) http.Handler { return self }

/*
HTTP handler that writes bytes. Note: for sending a string, use `goh.String`,
avoiding a bytes-to-string conversion.
*/
type Bytes struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
	Body    []byte
}

// Returns the pseudo-embedded `goh.Head` part.
func (self Bytes) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self Bytes) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().Write(rew)

	_, err := rew.Write(self.Body)
	if err != nil {
		err = fmt.Errorf(`[goh] failed to write response bytes: %w`, err)
		self.Head().errFunc()(rew, req, err, true)
	}
}

// Conforms to `goh.Han`.
func (self Bytes) Han(*http.Request) http.Handler { return self }

// Shortcut for `goh.BytesWith(http.StatusOK, body)`.
func BytesOk(body []byte) Bytes {
	return BytesWith(http.StatusOK, body)
}

// Shortcut for `goh.Bytes` with specific status and body.
func BytesWith(status int, body []byte) Bytes {
	return Bytes{Status: status, Body: body}
}

/*
HTTP handler that writes a string. Note: for sending bytes, use `goh.Bytes`,
avoiding a string-to-bytes conversion.
*/
type String struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
	Body    string
}

// Returns the pseudo-embedded `goh.Head` part.
func (self String) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self String) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().Write(rew)

	_, err := io.WriteString(rew, self.Body)
	if err != nil {
		err = fmt.Errorf(`[goh] failed to write response string: %w`, err)
		self.Head().errFunc()(rew, req, err, true)
	}
}

// Conforms to `goh.Han`.
func (self String) Han(*http.Request) http.Handler { return self }

// Shortcut for `goh.StringWith(http.StatusOK, body)`.
func StringOk(body string) String {
	return StringWith(http.StatusOK, body)
}

// Shortcut for `goh.String` with specific status and body.
func StringWith(status int, body string) String {
	return String{Status: status, Body: body}
}

/*
HTTP handler that automatically sets the appropriate JSON headers and encodes
its body as JSON. The field `.Indent` is passed to the JSON encoder.
*/
type Json struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
	Indent  string
	Body    interface{}
}

// Returns the pseudo-embedded `goh.Head` part.
func (self Json) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self Json) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	rew.Header().Set(HeadType, TypeJson)
	self.Head().Write(rew)

	writer := spyingWriter{Writer: rew}
	enc := json.NewEncoder(&writer)
	enc.SetIndent(``, self.Indent)

	err := enc.Encode(self.Body)
	if err != nil {
		err = fmt.Errorf(`[goh] failed to write response as JSON: %w`, err)
		self.Head().errFunc()(rew, req, err, writer.wrote)
	}
}

// Conforms to `goh.Han`.
func (self Json) Han(*http.Request) http.Handler { return self }

/*
Converts to `goh.Bytes` by encoding the body and adding the appropriate content
type header. Panics on encoding errors. Should be used in root scope to
pre-encode a static response:

	import "github.com/mitranim/goh"

	var someHan = goh.JsonOk(someValue).TryBytes()
*/
func (self Json) TryBytes() Bytes {
	var body []byte
	var err error

	if self.Indent == `` {
		body, err = json.Marshal(self.Body)
	} else {
		body, err = json.MarshalIndent(self.Body, ``, self.Indent)
	}

	if err != nil {
		panic(err)
	}
	return bytesFrom(self.Head(), TypeJson, body)
}

// Shortcut for `goh.JsonWith(http.StatusOK, body)`.
func JsonOk(body interface{}) Json {
	return JsonWith(http.StatusOK, body)
}

// Shortcut for `goh.Json` with specific status and body.
func JsonWith(status int, body interface{}) Json {
	return Json{Status: status, Body: body}
}

/*
HTTP handler that automatically sets the appropriate XML headers and encodes its
body as XML. The field `.Indent` is passed to the JSON encoder.

Caution: this does NOT prepend the processing instruction `<?xml?>`. When you
don't need to specify the encoding, this instruction is entirely skippable.
When you need to specify the encoding, wrap `.Body` in the utility type
`goh.XmlDoc` provided by this package.
*/
type Xml Json

// Returns the pseudo-embedded `goh.Head` part.
func (self Xml) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self Xml) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	rew.Header().Set(HeadType, `application/xml`)
	self.Head().Write(rew)

	writer := spyingWriter{Writer: rew}
	enc := xml.NewEncoder(&writer)
	enc.Indent(``, self.Indent)

	err := enc.Encode(self.Body)
	if err != nil {
		err = fmt.Errorf(`[goh] failed to write response as XML: %w`, err)
		self.Head().errFunc()(rew, req, err, writer.wrote)
	}
}

// Conforms to `goh.Han`.
func (self Xml) Han(*http.Request) http.Handler { return self }

/*
Converts to `goh.Bytes` by encoding the body and adding the appropriate content
type header. Panics on encoding errors. Should be used in root scope to
pre-encode a static response:

	import "github.com/mitranim/goh"

	var someHan = goh.XmlOk(someValue).TryBytes()
*/
func (self Xml) TryBytes() Bytes {
	var body []byte
	var err error

	if self.Indent == `` {
		body, err = xml.Marshal(self.Body)
	} else {
		body, err = xml.MarshalIndent(self.Body, ``, self.Indent)
	}

	if err != nil {
		panic(err)
	}
	return bytesFrom(self.Head(), `application/xml`, body)
}

// Shortcut for `goh.XmlWith(http.StatusOK, body)`.
func XmlOk(body interface{}) Xml {
	return XmlWith(http.StatusOK, body)
}

// Shortcut for `goh.Xml` with specific status and body.
func XmlWith(status int, body interface{}) Xml {
	return Xml{Status: status, Body: body}
}

// HTTP handler that performs an HTTP redirect.
type Redirect struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
	Link    string
}

// Returns the pseudo-embedded `goh.Head` part.
func (self Redirect) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self Redirect) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().writeHeaders(rew)
	http.Redirect(rew, req, self.Link, self.Status)
}

// Conforms to `goh.Han`.
func (self Redirect) Han(*http.Request) http.Handler { return self }

// Shortcut for `goh.Redirect` with specific status and body.
func RedirectWith(status int, link string) Redirect {
	return Redirect{Status: status, Link: link}
}

/*
Utility type for use together with `goh.Xml`. When encoded as XML, this prepends
the `<?xml?>` header with version 1.0 and the specified encoding, if any.
Example usage:

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

// Implement `encoding/xml.Marshaler`, prepending the `<?xml?>` processing
// instruction, with the specified encoding if available.
func (self XmlDoc) MarshalXML(enc *xml.Encoder, _ xml.StartElement) error {
	inst := xmlVersionInst
	if self.Encoding != `` {
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
HTTP handler that always serves a file at a specific FS path. For each request,
it verifies that the file exists and delegates to `http.ServeFile`. If the file
doesn't exist, this responds with 404 without calling `http.ServeFile`,
avoiding its undesirable "smarts".

Unlike `http.ServeFile` and `http.FileServer`, responding with 404 is optional.
`goh.File.MaybeHan` returns a nil handler if the file is not found. You can use
this to "try" serving a file, and fall back on something else.
*/
type File struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
	Path    string
}

// Returns the pseudo-embedded `goh.Head` part.
func (self File) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self File) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Head().Write(rew)

	if fileExists(self.Path) {
		http.ServeFile(rew, req, self.Path)
	} else {
		NotFound{}.ServeHTTP(rew, req)
	}
}

// Conforms to `goh.Han`. Always returns non-nil.
func (self File) Han(*http.Request) http.Handler { return self }

// Conforms to `goh.Han`. Returns self if file exists, otherwise returns nil.
// Can be used to "try" serving a file.
func (self File) MaybeHan(*http.Request) http.Handler {
	if fileExists(self.Path) {
		return self
	}
	return nil
}

/*
HTTP handler that serves files out of a given directory. Similar to
`http.FileServer`, but without its undesirable "smarts". This will serve only
individual files, without directory listings or redirects. In addition, the
method `goh.Dir.MaybeHan` supports "try file" functionality, allowing you to
fall back on serving something else when a requested file is not found.

The status, header, and err func are copied to each `goh.File` used for each
response.
*/
type Dir struct {
	Status  int
	Header  http.Header
	ErrFunc ErrFunc
	Path    string
	Filter  Filter
}

// Returns the pseudo-embedded `goh.Head` part.
func (self Dir) Head() Head {
	return Head{self.Status, self.Header, self.ErrFunc}
}

// Implement `http.Handler`.
func (self Dir) ServeHTTP(rew http.ResponseWriter, req *http.Request) {
	self.Han(req).ServeHTTP(rew, req)
}

// Conforms to `goh.Han`. Always returns non-nil.
func (self Dir) Han(req *http.Request) http.Handler {
	res := self.MaybeHan(req)
	if res != nil {
		return res
	}
	return NotFound{}
}

// Conforms to `goh.Han`. Returns nil if requested file is not found.
func (self Dir) MaybeHan(req *http.Request) http.Handler {
	reqPath := strings.TrimPrefix(req.URL.Path, `/`)
	if strings.Contains(reqPath, `..`) || strings.HasSuffix(reqPath, `/`) {
		return nil
	}

	filePath := filepath.Join(self.Path, reqPath)
	if !self.allow(filePath) {
		return nil
	}

	return self.file(filePath).MaybeHan(req)
}

func (self Dir) allow(path string) bool {
	if self.Filter != nil {
		return self.Filter.Allow(filepath.ToSlash(path))
	}
	return true
}

func (self Dir) file(path string) File {
	return File{
		Status:  self.Status,
		Header:  self.Header,
		ErrFunc: self.ErrFunc,
		Path:    path,
	}
}

/*
Used by `goh.Dir` to allow or deny serving specific paths. The input to `.Allow`
is a normalized filesystem path that uses Unix-style forward slashes on both
Unix and Windows. The path starts with `goh.Dir.Path`. For example:

	dir := goh.Dir{Path: `static`}
	req := &http.Request{URL: &url.URL{Path: `/some_file`}}
	dir.Han(req)
	->
	dir.Filter.Allow(`static/some_file`)
*/
type Filter interface {
	Allow(string) bool
}

/*
Function type that implements `goh.Filter`. Example usage:

	goh.Dir{Path: `.`, Filter: goh.FilterFunc(regexp.MustCompile(`^status/`))}
*/
type FilterFunc func(string) bool

// Implement `goh.Filter` by calling itself.
func (self FilterFunc) Allow(val string) bool {
	if self != nil {
		return self(val)
	}
	return false
}

/*
Implements `goh.Filter` by requiring that the input path is contained within one
of the given directories. "Contained" means it begins with the directory path
followed by a path separator.
*/
type AllowDirs []string

// Implement `goh.Filter`.
func (self AllowDirs) Allow(val string) bool {
	for _, dir := range self {
		if isSubpath(dir, val) {
			return true
		}
	}
	return false
}

/*
Zero-sized handler that returns with 404 without any additional headers or body
content. Used internally by `goh.File`.
*/
type NotFound struct{}

func (NotFound) ServeHTTP(rew http.ResponseWriter, _ *http.Request) {
	rew.WriteHeader(http.StatusNotFound)
}

// Conforms to `goh.Han`, returning self.
func (self NotFound) Han(req *http.Request) http.Handler { return self }

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
	if err != nil {
		*ptr = Err(err)
		return
	}

	*ptr = StringWith(http.StatusInternalServerError, fmt.Sprint(val))
}

func bytesFrom(head Head, contentType string, body []byte) Bytes {
	if contentType != `` {
		if head.Header == nil {
			head.Header = http.Header{HeadType: {contentType}}
		} else {
			head.Header = head.Header.Clone()
			head.Header.Set(HeadType, contentType)
		}
	}

	return Bytes{
		Status:  head.Status,
		Header:  head.Header,
		ErrFunc: head.ErrFunc,
		Body:    body,
	}
}

func fileExists(path string) bool {
	stat, _ := os.Stat(path)
	return stat != nil && !stat.IsDir()
}

func isSubpath(sup, sub string) bool {
	return strings.HasPrefix(sub, sup) &&
		strings.HasPrefix(sub[len(sup):], `/`)
}
