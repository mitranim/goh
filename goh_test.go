package goh

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
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

func TestXmlDoc(t *testing.T) {
	bytes, err := xml.Marshal(XmlDoc{
		Encoding: "utf-8",
		Val:      `text`,
	})
	try(err)

	eq(`<?xml version="1.0" encoding="utf-8"?><string>text</string>`, string(bytes))
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
