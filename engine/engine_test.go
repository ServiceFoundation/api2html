package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestEngineFactory_New_koConfigParser(t *testing.T) {
	expectedErr := fmt.Errorf("boooom")
	ef := EngineFactory{
		Parser: func(path string) (Config, error) {
			if path != "something" {
				t.Errorf("unexpected path: %s", path)
			}
			return Config{}, expectedErr
		},
	}
	if _, err := ef.New("something", true); err == nil {
		t.Error("expecting error")
	} else if err != expectedErr {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func TestEngineFactory_New_ok(t *testing.T) {
	if err := ioutil.WriteFile("test_tmpl", []byte("hi, {{Extra.name}}!"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if err := ioutil.WriteFile("test_lyt", []byte("-{{{content}}}-"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	defer os.Remove("test_tmpl")
	defer os.Remove("test_lyt")
	expectedCfg := Config{
		Pages: []Page{
			{
				URLPattern: "/a",
				Layout:     "b",
				Template:   "a",
				Extra: map[string]interface{}{
					"name": "stranger",
				},
			},
		},
		Templates: map[string]string{"a": "test_tmpl"},
		Layouts:   map[string]string{"b": "test_lyt"},
	}
	templateStore := NewTemplateStore()
	ef := DefaultEngineFactory
	ef.Parser = func(path string) (Config, error) {
		if path != "something" {
			t.Errorf("unexpected path: %s", path)
		}
		return expectedCfg, nil
	}
	ef.TemplateStoreFactory = func() *TemplateStore { return templateStore }
	ef.MustachePageFactory = func(e *gin.Engine, ts *TemplateStore) MustachePageFactory {
		if ts != templateStore {
			t.Errorf("unexpected template store: %v", ts)
		}
		return NewMustachePageFactory(e, ts)
	}

	e, err := ef.New("something", true)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}

	time.Sleep(200 * time.Millisecond)

	assertResponse(t, e, "/a", http.StatusOK, "-hi, stranger!-")
	assertResponse(t, e, "/b", http.StatusNotFound, default404Tmpl)
}

func TestNew(t *testing.T) {
	if err := ioutil.WriteFile("test_tmpl", []byte("hi, {{Extra.name}}!"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	defer os.Remove("test_tmpl")
	if err := ioutil.WriteFile("test_lyt", []byte("-{{{content}}}-"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	defer os.Remove("test_lyt")
	if err := os.Mkdir("static", 0777); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	defer os.RemoveAll("static")
	if err := ioutil.WriteFile("static/s.txt", []byte("12345"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if err := ioutil.WriteFile("static/404", []byte("404"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if err := ioutil.WriteFile("static/500", []byte("500"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if err := ioutil.WriteFile("static/robots.txt", []byte("robots.txt"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if err := ioutil.WriteFile("static/sitemap.xml", []byte("sitemap.xml"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if err := os.Mkdir("public", 0777); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	defer os.RemoveAll("public")
	if err := ioutil.WriteFile("public/public.js", []byte("public"), 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	cfg := Config{
		Pages: []Page{
			{
				URLPattern: "/a",
				Layout:     "b",
				Template:   "a",
				Extra: map[string]interface{}{
					"name": "stranger",
				},
			},
		},
		StaticTXTContent: []string{"s.txt"},
		Templates:        map[string]string{"a": "test_tmpl"},
		Layouts:          map[string]string{"b": "test_lyt"},
		Robots:           true,
		Sitemap:          true,
		PublicFolder: &PublicFolder{
			Path:   "./public",
			Prefix: "/js",
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}
	if err := ioutil.WriteFile("public/config.json", data, 0644); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}

	e, err := New("public/config.json", false)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}

	time.Sleep(200 * time.Millisecond)

	assertResponse(t, e, "/a", http.StatusOK, "-hi, stranger!-")
	assertResponse(t, e, "/b", http.StatusNotFound, "404")
	assertResponse(t, e, "/robots.txt", http.StatusOK, "robots.txt")
	assertResponse(t, e, "/sitemap.xml", http.StatusOK, "sitemap.xml")
	assertResponse(t, e, "/js/public.js", http.StatusOK, "public")
	assertResponse(t, e, "/s.txt", http.StatusOK, "12345")
}

func assertResponse(t *testing.T, e *gin.Engine, url string, status int, body string) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Errorf("[%s] unexpected error: %s", url, err.Error())
		return
	}
	e.ServeHTTP(w, req)
	if statusCode := w.Result().StatusCode; statusCode != status {
		t.Errorf("[%s] unexpected status code: %d (%v)", url, statusCode, w.Result())
	}

	data, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Errorf("[%s] unexpected error: %s (%v)", url, err.Error(), w.Result())
		return
	}
	w.Result().Body.Close()

	if string(data) != body {
		t.Errorf("[%s] unexpected body: %s (%v)", url, string(data), w.Result())
	}
}
