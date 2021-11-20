package goh

import (
	"encoding/xml"
	"fmt"
	"net/http"
	ht "net/http/httptest"
	"net/url"
	"os"
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
	_ = http.Handler(File{})
	_ = http.Handler(Dir{})
	_ = http.Handler(NotFound{})
)

var (
	_ = Han(Reader{}.Han)
	_ = Han(Bytes{}.Han)
	_ = Han(String{}.Han)
	_ = Han(Json{}.Han)
	_ = Han(Xml{}.Han)
	_ = Han(Redirect{}.Han)
	_ = Han(File{}.Han)
	_ = Han(File{}.MaybeHan)
	_ = Han(Dir{}.Han)
	_ = Han(Dir{}.MaybeHan)
	_ = Han(NotFound{}.Han)
)

type JsonVal struct {
	Val string `json:"val"`
}

type XmlVal struct {
	XMLName xml.Name
	Val     string `xml:"val"`
}

var (
	headSrc = http.Header{`One`: {`two`}, `three`: {`four`}}
	headExp = http.Header{`One`: {`two`}, `three`: {`four`}}
)

func TestErr_empty(t *testing.T) {
	rew := ht.NewRecorder()
	Err(nil).ServeHTTP(rew, nil)

	eq(t, 500, rew.Code)
	eq(t, `unknown error`, rew.Body.String())
}

func TestErr_full(t *testing.T) {
	rew := ht.NewRecorder()
	Err(fmt.Errorf(`fail`)).ServeHTTP(rew, nil)

	eq(t, 500, rew.Code)
	eq(t, `fail`, rew.Body.String())
}

func TestTryJsonBytes(t *testing.T) {
	eq(
		t,
		Bytes{
			Status: http.StatusOK,
			Header: http.Header{`Content-Type`: {`application/json`}},
			Body:   []byte(`{"val":"one"}`),
		},
		TryJsonBytes(JsonVal{`one`}),
	)
}

func TestHead_empty(t *testing.T) {
	rew := ht.NewRecorder()
	eq(t, 200, rew.Code)

	Head{}.Write(rew)

	eq(t, 200, rew.Code)
	eq(t, 0, len(rew.Result().Header))
}

func TestHead_full(t *testing.T) {
	rew := ht.NewRecorder()
	Head{Status: 201, Header: headSrc}.Write(rew)

	eq(t, 201, rew.Code)
	eq(t, headExp, rew.Result().Header)
}

func TestReader(t *testing.T) {
	rew := ht.NewRecorder()

	const src = `hello world`
	Reader{Status: 201, Header: headSrc, Body: strings.NewReader(src)}.ServeHTTP(rew, nil)

	eq(t, 201, rew.Code)
	eq(t, headExp, rew.Result().Header)
	eq(t, src, rew.Body.String())
}

func TestBytes(t *testing.T) {
	rew := ht.NewRecorder()

	const src = `hello world`
	Bytes{Status: 201, Header: headSrc, Body: []byte(src)}.ServeHTTP(rew, nil)

	eq(t, 201, rew.Code)
	eq(t, headExp, rew.Result().Header)
	eq(t, src, rew.Body.String())
}

func TestString(t *testing.T) {
	rew := ht.NewRecorder()

	const src = `hello world`
	String{Status: 201, Header: headSrc, Body: src}.ServeHTTP(rew, nil)

	eq(t, 201, rew.Code)
	eq(t, headExp, rew.Result().Header)
	eq(t, src, rew.Body.String())
}

func TestJson(t *testing.T) {
	rew := ht.NewRecorder()

	headExp := headExp.Clone()
	headExp.Set(`content-type`, `application/json`)

	Json{Status: 201, Header: headSrc, Body: JsonVal{`hello world`}}.ServeHTTP(rew, nil)

	eq(t, 201, rew.Code)
	eq(t, headExp, rew.Result().Header)
	eq(t, `{"val":"hello world"}`, strings.TrimSpace(rew.Body.String()))
}

func TestJson_TryBytes_nil_head(t *testing.T) {
	res := Json{
		Status:  201,
		Body:    JsonVal{`hello world`},
		ErrFunc: ErrHandler,
	}.TryBytes()

	headExp := http.Header{}
	headExp.Set(`content-type`, `application/json`)

	eq(t, 201, res.Status)
	eq(t, headExp, res.Header)
	eq(t, `{"val":"hello world"}`, string(res.Body))
	eq(t, ptr(ErrHandler), ptr(res.ErrFunc))
}

func TestJson_TryBytes_non_nil_head(t *testing.T) {
	res := Json{
		Status:  201,
		Header:  headSrc,
		Body:    JsonVal{`hello world`},
		ErrFunc: ErrHandler,
	}.TryBytes()

	headExp := headSrc.Clone()
	headExp.Set(`content-type`, `application/json`)

	eq(t, 201, res.Status)
	eq(t, headExp, res.Header)
	eq(t, `{"val":"hello world"}`, string(res.Body))
	eq(t, ptr(ErrHandler), ptr(res.ErrFunc))
}

func TestXml(t *testing.T) {
	rew := ht.NewRecorder()

	headExp := headExp.Clone()
	headExp.Set(`content-type`, `application/xml`)

	Xml{Status: 201, Header: headSrc, Body: XmlVal{xml.Name{Local: `tag`}, `hello world`}}.ServeHTTP(rew, nil)

	eq(t, 201, rew.Code)
	eq(t, headExp, rew.Result().Header)
	eq(t, `<tag><val>hello world</val></tag>`, strings.TrimSpace(rew.Body.String()))
}

func TestXml_TryBytes_nil_head(t *testing.T) {
	res := Xml{
		Status:  201,
		Body:    XmlVal{xml.Name{Local: `tag`}, `hello world`},
		ErrFunc: ErrHandler,
	}.TryBytes()

	headExp := http.Header{}
	headExp.Set(`content-type`, `application/xml`)

	eq(t, 201, res.Status)
	eq(t, headExp, res.Header)
	eq(t, `<tag><val>hello world</val></tag>`, string(res.Body))
	eq(t, ptr(ErrHandler), ptr(res.ErrFunc))
}

func TestXml_TryBytes_non_nil_head(t *testing.T) {
	res := Xml{
		Status:  201,
		Header:  headSrc,
		Body:    XmlVal{xml.Name{Local: `tag`}, `hello world`},
		ErrFunc: ErrHandler,
	}.TryBytes()

	headExp := headSrc.Clone()
	headExp.Set(`content-type`, `application/xml`)

	eq(t, 201, res.Status)
	eq(t, headExp, res.Header)
	eq(t, `<tag><val>hello world</val></tag>`, string(res.Body))
	eq(t, ptr(ErrHandler), ptr(res.ErrFunc))
}

func TestRedirect(t *testing.T) {
	rew := ht.NewRecorder()
	req := ht.NewRequest(http.MethodPost, `/`, nil)

	headExp := headExp.Clone()
	headExp.Set(`location`, `/three`)

	Redirect{Status: 301, Header: headSrc, Link: `/three`}.ServeHTTP(rew, req)

	eq(t, 301, rew.Code)
	eq(t, headExp, rew.Result().Header)
	eq(t, ``, rew.Body.String())
}

func TestXmlDoc(t *testing.T) {
	bytes, err := xml.Marshal(XmlDoc{
		Encoding: "utf-8",
		Val:      `text`,
	})
	try(err)

	eq(t, `<?xml version="1.0" encoding="utf-8"?><string>text</string>`, string(bytes))
}

func TestFile(t *testing.T) {
	t.Run(`missing`, func(t *testing.T) {
		testFile404(t, File{Path: `0589a8bfe3854d499c5e3beef89660c1`})
	})

	t.Run(`dir rather than file`, func(t *testing.T) {
		testFile404(t, File{Path: `.`})
	})

	t.Run(`ends with slash`, func(t *testing.T) {
		testFile404(t, File{Path: `readme.md/`})
	})

	t.Run(`exists`, func(t *testing.T) {
		testFileOk(t, File{Path: `readme.md`}, Head{Status: 200})
	})

	t.Run(`use head`, func(t *testing.T) {
		testFileOk(t, File{Status: 202, Path: `readme.md`}, Head{Status: 202, Header: http.Header{}})
	})
}

func testFile404(t testing.TB, file File) {
	eq(t, nil, file.MaybeHan(nil))
	eq(t, file, file.Han(nil))

	rew := ht.NewRecorder()
	file.ServeHTTP(rew, nil)

	eq(t, http.StatusNotFound, rew.Code)
}

func testFileOk(t testing.TB, file File, head Head) {
	eq(t, file, file.MaybeHan(nil))
	eq(t, file, file.Han(nil))

	rew := ht.NewRecorder()
	file.ServeHTTP(rew, pathReq(`b824014bb44242c5b20b1706a5e0a930`))

	eq(t, head.Status, rew.Code)
	if head.Header != nil {
		eq(t, head.Header, rew.Result().Header)
	}
	eq(t, readFile(file.Path), rew.Body.Bytes())
}

func TestDir(t *testing.T) {
	try(os.Chdir(`..`))
	t.Cleanup(func() { os.Chdir(`goh`) })

	t.Run(`without filter`, func(t *testing.T) {
		dir := Dir{Path: `goh`}

		t.Run(`missing`, func(t *testing.T) {
			testDir404(t, dir, pathReq(`c5ba8aa69fff421fb4ae48c6361fa7e2`))
		})

		t.Run(`dir rather than file`, func(t *testing.T) {
			testDir404(t, dir, pathReq(`goh/.git`))
		})

		t.Run(`ends with slash`, func(t *testing.T) {
			testDir404(t, dir, pathReq(`readme.md/`))
		})

		t.Run(`redundant nesting`, func(t *testing.T) {
			testDir404(t, dir, pathReq(`goh/readme.md`))
		})

		t.Run(`wrong dir`, func(t *testing.T) {
			testDir404(t, Dir{Path: `68482a6a782445679d53786e6c08b8bd`}, pathReq(`readme.md`))
		})

		t.Run(`exists`, func(t *testing.T) {
			testDirOk(t, dir, pathReq(`readme.md`), `goh/readme.md`)
		})
	})

	t.Run(`with filter`, func(t *testing.T) {
		t.Run(`not exists and not allowed`, func(t *testing.T) {
			filter := FilterFunc(func(string) bool { return false })
			testDir404(t, Dir{Path: `goh`, Filter: filter}, pathReq(`ffce2ef7d9cc415cab31b1d716e12c1c`))
		})

		t.Run(`not exists and allowed`, func(t *testing.T) {
			filter := FilterFunc(func(string) bool { return true })
			testDir404(t, Dir{Path: `goh`, Filter: filter}, pathReq(`ffce2ef7d9cc415cab31b1d716e12c1c`))
		})

		t.Run(`exists and not allowed`, func(t *testing.T) {
			filter := FilterFunc(func(string) bool { return false })
			testDir404(t, Dir{Path: `goh`, Filter: filter}, pathReq(`readme.md`))
		})

		t.Run(`exists and allowed`, func(t *testing.T) {
			filter := FilterFunc(func(string) bool { return true })
			testDirOk(t, Dir{Path: `goh`, Filter: filter}, pathReq(`readme.md`), `goh/readme.md`)
		})
	})
}

func testDir404(t testing.TB, dir Dir, req *http.Request) {
	eq(t, nil, dir.MaybeHan(req))
	eq(t, NotFound{}, dir.Han(req))

	rew := ht.NewRecorder()
	dir.ServeHTTP(rew, req)

	eq(t, http.StatusNotFound, rew.Code)
}

func testDirOk(t testing.TB, dir Dir, req *http.Request, expPath string) {
	eq(t, File{Path: expPath}, dir.MaybeHan(req))
	eq(t, File{Path: expPath}, dir.Han(req))

	testFileOk(t, File{Path: expPath}, Head{Status: http.StatusOK})

	rew := ht.NewRecorder()
	dir.ServeHTTP(rew, req)

	eq(t, http.StatusOK, rew.Code)
	eq(t, readFile(expPath), rew.Body.Bytes())
}

func Test_isSubpath(t *testing.T) {
	test := func(exp bool, sub, sup string) {
		t.Helper()
		eq(t, exp, isSubpath(sub, sup))
	}

	test(false, ``, ``)
	test(false, `one`, `one`)
	test(false, `one`, `/one`)
	test(false, `/one`, `one`)
	test(false, `/one`, `/one`)
	test(false, `one/`, `one`)
	test(false, `one/`, `one/`)
	test(false, `one`, `onetwo`)
	test(false, `one/`, `onetwo`)
	test(true, `one`, `one/`)
	test(true, `one`, `one/two`)
	test(true, `one`, `one/two/three`)
	test(true, `one`, `one/two/three.four`)
}

func TestAllowDirs(t *testing.T) {
	test := func(exp bool, dirs AllowDirs, path string) {
		t.Helper()
		eq(t, exp, dirs.Allow(path))
	}

	test(false, AllowDirs{}, ``)
	test(false, AllowDirs{}, `one`)
	test(false, AllowDirs{}, `one/`)

	test(false, AllowDirs{`one`}, `two`)
	test(false, AllowDirs{`one`}, `one`)
	test(true, AllowDirs{`one`}, `one/`)
	test(true, AllowDirs{`one`}, `one/two`)

	test(false, AllowDirs{`one`, `two`}, `three`)
	test(false, AllowDirs{`one`, `two`}, `three/`)
	test(true, AllowDirs{`one`, `two`}, `one/`)
	test(true, AllowDirs{`one`, `two`}, `one/two`)
	test(true, AllowDirs{`one`, `two`}, `two/`)
	test(true, AllowDirs{`one`, `two`}, `two/three`)
}

func TestHandler_error(t *testing.T) {
	handler := Handler(func() http.Handler { panic("fail") })
	eq(t, StringWith(http.StatusInternalServerError, `fail`), handler)
}

func TestHandler_success(t *testing.T) {
	handler := Handler(func() http.Handler { return StringOk(`ok`) })
	eq(t, StringOk(`ok`), handler)
}

func eq(t testing.TB, exp, act interface{}) {
	t.Helper()
	if !reflect.DeepEqual(exp, act) {
		t.Fatalf(`
expected (detailed):
	%#[1]v
actual (detailed):
	%#[2]v
expected (simple):
	%[1]v
actual (simple):
	%[2]v
`, exp, act)
	}
}

func try(err error) {
	if err != nil {
		panic(err)
	}
}

func ptr(val interface{}) uintptr {
	return reflect.ValueOf(val).Pointer()
}

func pathUrl(path string) *url.URL      { return &url.URL{Path: path} }
func pathReq(path string) *http.Request { return &http.Request{URL: pathUrl(path)} }

func readFile(path string) []byte {
	val, err := os.ReadFile(path)
	try(err)
	return val
}
