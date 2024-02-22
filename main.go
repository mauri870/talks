package main

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	txttemplate "text/template"

	"golang.org/x/tools/present"
)

var (
	presentTemplates = map[string]*template.Template{
		".article": parsePresentTemplate("article.tmpl"),
		".slide":   parsePresentTemplate("slides.tmpl"),
	}
	homeArticle = loadHomeArticle()

	playCompileURL = "https://play.golang.org/compile"

	ErrNotFound = errors.New("not found")
)

func main() {
	http.Handle("/", handlerFunc(serveRoot))
	http.Handle("/compile", handlerFunc(serveCompile))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./present/static"))))
	http.Handle("/code/", http.StripPrefix("/code/", http.FileServer(http.Dir("./code"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("./images"))))
	http.HandleFunc("/play.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./present/play.js")
	})
	present.PlayEnabled = true

	if err := http.ListenAndServe(":4000", nil); err != nil {
		slog.Error("Error running server", err)
		os.Exit(1)
	}
}

func playable(c present.Code) bool {
	return present.PlayEnabled && c.Play && c.Ext == ".go"
}

func parsePresentTemplate(name string) *template.Template {
	t := present.Template()
	t = t.Funcs(template.FuncMap{"playable": playable})
	if _, err := t.ParseFiles("present/templates/"+name, "present/templates/action.tmpl"); err != nil {
		panic(err)
	}
	t = t.Lookup("root")
	if t == nil {
		panic("root template not found for " + name)
	}
	return t
}

func loadHomeArticle() []byte {
	const fname = "assets/home.article"
	fcontents, err := os.ReadFile(fname)
	if err != nil {
		panic(err)
	}

	tmpl, err := txttemplate.New("home").Parse(string(fcontents))
	if err != nil {
		panic(err)
	}

	slides, err := filepath.Glob("./*.slide")
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	tmpl.Execute(&buf, slides)

	doc, err := present.Parse(&buf, fname, 0)
	if err != nil {
		panic(err)
	}
	buf.Reset()
	if err := renderPresentation(&buf, fname, doc); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func renderPresentation(w io.Writer, fname string, doc *present.Doc) error {
	t := presentTemplates[path.Ext(fname)]
	if t == nil {
		return errors.New("unknown template extension")
	}
	data := struct {
		*present.Doc
		Template     *template.Template
		PlayEnabled  bool
		NotesEnabled bool
	}{doc, t, true, true}
	return t.Execute(w, &data)
}

type handlerFunc func(http.ResponseWriter, *http.Request) error

func (f handlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := f(w, r)
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case os.IsNotExist(err):
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func serveRoot(w http.ResponseWriter, r *http.Request) error {
	switch {
	case r.Method != "GET" && r.Method != "HEAD":
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil
	case r.URL.Path == "/":
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(homeArticle)
		return err
	default:
		return servePresentation(w, r)
	}
}

func servePresentation(w http.ResponseWriter, r *http.Request) error {
	filename := filepath.Base(r.URL.Path[1:])

	if !strings.HasSuffix(filename, ".article") && !strings.HasSuffix(filename, ".slide") {
		w.WriteHeader(404)
		return nil
	}

	parser := &present.Context{
		ReadFile: func(name string) ([]byte, error) {
			return os.ReadFile(name)
		},
	}

	f, err := os.Open(filename)
	if err != nil {
		return ErrNotFound
	}

	doc, err := parser.Parse(f, filename, 0)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := renderPresentation(&buf, filename, doc); err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(buf.Bytes())
	return err
}

func serveCompile(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.PostForm(playCompileURL, r.Form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	_, err = io.Copy(w, resp.Body)
	return err
}
