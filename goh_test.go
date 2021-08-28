package goh

import (
	"encoding/xml"
	"fmt"
	"net/http"
	ht "net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

var (
	_ = http.Handler(Reader{})
	_ = http.Handler(Bytes{})
	_ = http.Handler(String{})
	_ = http.Handler(Json{})
	_ = http.Handler(Xml{})
	_ = http.Handler(Redirect{})
)

var (
	_ = ResFunc(Reader{}.Res)
	_ = ResFunc(Bytes{}.Res)
	_ = ResFunc(String{}.Res)
	_ = ResFunc(Json{}.Res)
	_ = ResFunc(Xml{}.Res)
	_ = ResFunc(Redirect{}.Res)
)

var (
	headSrc = http.Header{`one`: {`two`}, `three`: {`four`}}
	headExp = http.Header{`One`: {`two`}, `Three`: {`four`}}
)

func TestErr_empty(t *testing.T) {
	rew := ht.NewRecorder()
	Err(nil).ServeHTTP(rew, nil)

	eq(500, rew.Code)
	eq(`unknown error`, rew.Body.String())
}

func TestErr_full(t *testing.T) {
	rew := ht.NewRecorder()
	Err(fmt.Errorf(`fail`)).ServeHTTP(rew, nil)

	eq(500, rew.Code)
	eq(`fail`, rew.Body.String())
}

func TestHead_empty(t *testing.T) {
	rew := ht.NewRecorder()
	eq(200, rew.Code)

	Head{}.Write(rew)

	eq(200, rew.Code)
	eq(0, len(rew.Result().Header))
}

func TestHead_full(t *testing.T) {
	rew := ht.NewRecorder()
	Head{Status: 201, Header: headSrc}.Write(rew)

	eq(201, rew.Code)
	eq(headExp, rew.Result().Header)
}

func TestReader(t *testing.T) {
	rew := ht.NewRecorder()

	const src = `hello world`
	Reader{Status: 201, Header: headSrc, Body: strings.NewReader(src)}.ServeHTTP(rew, nil)

	eq(201, rew.Code)
	eq(headExp, rew.Result().Header)
	eq(src, rew.Body.String())
}

func TestBytes(t *testing.T) {
	rew := ht.NewRecorder()

	const src = `hello world`
	Bytes{Status: 201, Header: headSrc, Body: []byte(src)}.ServeHTTP(rew, nil)

	eq(201, rew.Code)
	eq(headExp, rew.Result().Header)
	eq(src, rew.Body.String())
}

func TestString(t *testing.T) {
	rew := ht.NewRecorder()

	const src = `hello world`
	String{Status: 201, Header: headSrc, Body: src}.ServeHTTP(rew, nil)

	eq(201, rew.Code)
	eq(headExp, rew.Result().Header)
	eq(src, rew.Body.String())
}

func TestJson(t *testing.T) {
	rew := ht.NewRecorder()

	headExp := headExp.Clone()
	headExp.Set(`content-type`, `application/json`)

	type T struct {
		Val string `json:"val"`
	}

	Json{Status: 201, Header: headSrc, Body: T{`hello world`}}.ServeHTTP(rew, nil)

	eq(201, rew.Code)
	eq(headExp, rew.Result().Header)
	eq(`{"val":"hello world"}`, strings.TrimSpace(rew.Body.String()))
}

func TestXml(t *testing.T) {
	rew := ht.NewRecorder()

	headExp := headExp.Clone()
	headExp.Set(`content-type`, `application/xml`)

	type T struct {
		XMLName xml.Name
		Val     string `xml:"val"`
	}

	Xml{Status: 201, Header: headSrc, Body: T{xml.Name{Local: `tag`}, `hello world`}}.ServeHTTP(rew, nil)

	eq(201, rew.Code)
	eq(headExp, rew.Result().Header)
	eq(`<tag><val>hello world</val></tag>`, strings.TrimSpace(rew.Body.String()))
}

func TestRedirect(t *testing.T) {
	rew := ht.NewRecorder()
	req := ht.NewRequest(http.MethodPost, `/`, nil)

	headExp := headExp.Clone()
	headExp.Set(`location`, `/three`)

	Redirect{Status: 301, Header: headSrc, Link: `/three`}.ServeHTTP(rew, req)

	eq(301, rew.Code)
	eq(headExp, rew.Result().Header)
	eq(``, rew.Body.String())
}

func TestXmlDoc(t *testing.T) {
	bytes, err := xml.Marshal(XmlDoc{
		Encoding: "utf-8",
		Val:      `text`,
	})
	try(err)

	eq(`<?xml version="1.0" encoding="utf-8"?><string>text</string>`, string(bytes))
}

func TestHandler_error(t *testing.T) {
	handler := Handler(func() http.Handler { panic("fail") })
	eq(StringWith(http.StatusInternalServerError, `fail`), handler)
}

func TestHandler_success(t *testing.T) {
	handler := Handler(func() http.Handler { return StringOk(`ok`) })
	eq(StringOk(`ok`), handler)
}

func eq(exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		panic(fmt.Errorf("expected:\n%#v\ngot:\n%#v\n", exp, act))
	}
}

func try(err error) {
	if err != nil {
		panic(err)
	}
}
