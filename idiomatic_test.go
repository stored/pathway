package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

type Test struct {
	// put fields to specify auth, logging, etc
}

func (Test) Echo(in string) string { // put the request/response format here
	log.Println("i got called")
	return "i got called"
}

func (Test) Adder(in []int) int {
	i := 0
	for _, v := range in {
		i += v
	}
	return i
}

func (Test) PathAdder(a, b, c int, _ interface{}) int {
	return Test{}.Adder([]int{a, b, c})
}

func TestReflection(t *testing.T) {
	h := FindBackends(Test{})
	t.Log(h)
	if _, ok := h["echo"]; !ok {
		t.Error("missing test/echo path")
		return
	}
}

func TestGoodPaths(t *testing.T) {
	AddBackend(Test{})
	t.Log("attempting multiple good calls")
	for _, v := range []struct {
		string
		io.Reader
	}{
		{"test/echo", bytes.NewBufferString(`"ping"`)},
		{"test/adder", bytes.NewBufferString(`[1,2]`)},
		{"test/pathadder/1/2/3", nil},
		{"test/adder", nil},
	} {
		_, err := CallBackend(v.string, v.Reader)
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
	}
}

func TestBadPaths(t *testing.T) {
	AddBackend(Test{})
	t.Log("attempting multiple bad calls")
	for _, v := range []struct {
		string
		io.Reader
	}{
		{"test/echo", bytes.NewBufferString(`{broken`)},
		{"test/missing", nil},
		{"boop/adder", nil},
	} {
		_, err := CallBackend(v.string, v.Reader)
		if err == nil {
			t.Errorf("expected an error for %s, got none", v.string)
		}
	}
}

func TestHttp(t *testing.T) {
	AddBackend(Test{})
	http.Handle("/", Handler)
	r, _ := http.NewRequest("POST", "/test/adder", bytes.NewBufferString(`[1,2,3,4]`))
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, r)
	t.Log("res", w.Code, "body", w.Body.String())
	if w.Code != 200 {
		t.Error("w.Code != 200")
	}
}
