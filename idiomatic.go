package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
)

func FindBackends(config interface{}) map[string]backend {
	paths := make(map[string]backend)

	t := reflect.TypeOf(config)
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		name := strings.ToLower(m.Name)

		b := backend{}
		b.Method = m
		if err := b.Cache(); err != nil {
			panic(err)
		}
		paths[name] = b
	}
	return paths
}

type backend struct {
	reflect.Method
	errPos *int
	valPos *int

	dstType  reflect.Type
	pthTypes []reflect.Type
}

func (b backend) String() string {
	return fmt.Sprintf(`%s{err:%t, val:%t, dst:%s, pth:%d}`,
		b.Name, b.errPos != nil, b.valPos != nil, b.dstType.Name(), len(b.pthTypes))
}

func (b *backend) Cache() error {
	// find possible input values
	for i := 1; i < b.Method.Type.NumIn(); i++ {
		b.pthTypes = append(b.pthTypes, b.Method.Type.In(i))
	}
	if len(b.pthTypes) > 0 {
		b.dstType = b.pthTypes[len(b.pthTypes)-1]
		b.pthTypes = b.pthTypes[:len(b.pthTypes)-1]
	}
	var e error
	eType := reflect.TypeOf(&e).Elem()

	// find possible error or value returns
	for i := 0; i < b.Method.Type.NumOut(); i++ {
		t := b.Method.Type.Out(i)
		if t.Implements(eType) {
			pos := i
			b.errPos = &pos
		} else {
			pos := i
			b.valPos = &pos
		}
	}
	return nil
}

type backendSet map[string]map[string]backend

type BackendResponse interface{}

func New() backendSet {
	return make(backendSet)
}

func (bs backendSet) String() string {
	paths := []string{}
	for name, b := range bs {
		methods := []string{}
		for methodName, back := range b {
			methods = append(methods, fmt.Sprintf(`%s: %s`, methodName, back))
		}
		paths = append(paths, name+"/"+strings.Join(methods, ","))
	}
	return strings.Join(paths, ", ")
}

func (bs backendSet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res, err := bs.CallBackend(strings.TrimLeft(r.URL.RequestURI(), "/"), r.Body)
	r.Body.Close()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(res)
}

func (back *backendSet) AddBackend(config interface{}) {
	name := strings.ToLower(reflect.TypeOf(config).Name())
	(*back)[name] = FindBackends(config)
}

func (back backendSet) CallBackendString(path string, payload string) (res BackendResponse, err error) {
	return back.CallBackend(path, bytes.NewBufferString(payload))
}

func (back backendSet) CallBackend(path string, payload io.Reader) (res BackendResponse, err error) {
	parts := strings.Split(path, `/`)
	m, ok := back[parts[0]][parts[1]]
	if !ok {
		return res, fmt.Errorf(`path "%s" not found`, path)
	}
	parts = parts[2:]
	inputs := []reflect.Value{reflect.Zero(m.Type.In(0))}

	if len(m.pthTypes) > 0 {
		if len(parts) < len(m.pthTypes) {
			return res, fmt.Errorf(`path "%s" not enough args, expecting %d`, path, len(m.pthTypes))
		}
		for i, t := range m.pthTypes {
			s := parts[i]
			instance := reflect.New(t)
			json.Unmarshal([]byte(`"`+s+`"`), instance.Interface())
			inputs = append(inputs, instance.Elem())
		}
	}

	if m.dstType != nil {
		v := reflect.New(m.dstType)

		if payload != nil {
			if err = json.NewDecoder(payload).Decode(v.Interface()); err != nil {
				return res, fmt.Errorf(`path "%s" got invalid json: %s`, path, err)
			}
		}

		inputs = append(inputs, v.Elem())
	}

	if len(inputs) != m.Func.Type().NumIn() {
		return res, fmt.Errorf(`expected %d inputs, got %d`, m.Func.Type().NumIn(), len(inputs))
	}

	outputs := m.Func.Call(inputs)
	log.Printf("EXEC %s(%s) = %s", m.Name, inputs, outputs)
	if m.errPos != nil {
		if asErr, ok := outputs[*m.errPos].Interface().(error); ok && asErr != nil {
			return res, asErr
		}
	}

	if m.valPos != nil {
		res = outputs[*m.valPos].Interface()
	}
	return
}

var a = New()
var defaultBackend = &a

func AddBackend(c interface{}) { defaultBackend.AddBackend(c) }
func CallBackend(a string, b io.Reader) (BackendResponse, error) {
	return defaultBackend.CallBackend(a, b)
}

var Handler = defaultBackend
