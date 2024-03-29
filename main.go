package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

const page = `<html>
<head>
    <title>Upload file</title>
</head>
<body>
<form enctype="multipart/form-data" action="/common-mark-upload" method="post">
      <input type="file" name="upload-file" />
      <input type="hidden" name="token" value="{{.}}"/>
      <input type="submit" value="upload" />
</form>
</body>
</html>
`

func handleErr(w http.ResponseWriter, stage string, err error, code int) {
	log.Printf(stage+": %v\n", err)
	fmt.Fprintf(w, stage+": %v\n", err)
	http.Error(w, err.Error(), code)
}

func upload(w http.ResponseWriter, req *http.Request) {
	log.Printf("method: %v (from %s)\n", req.Method, req.RemoteAddr)
	switch req.Method {
	case "GET":
		crutime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(crutime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		t, err := template.New("upload").Parse(page)
		if err != nil {
			handleErr(w, "error parsing upload-page", err, http.StatusInternalServerError)
			return
		}

		t.Execute(w, token)

	case "POST":
		req.ParseMultipartForm(500 << 20)
		file, handler, err := req.FormFile("upload-file")
		if err != nil {
			handleErr(w, "error parsing form-file", err, http.StatusInternalServerError)
			return
		}
		defer file.Close()
		hdata, err := cnvToHTML(file)
		file.Seek(0, 0)
		if err != nil {
			handleErr(w, "error converting to HTML", err, http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(
			w,
			`
 <style>
 %s
 </style>
`,
			css,
		)

		io.Copy(w, hdata)

		os.MkdirAll("./uploads", 0755)

		now := time.Now().Unix()
		f, err := os.Create(fmt.Sprintf(
			"./uploads/%10d-%s", now,
			handler.Filename,
		))
		if err != nil {
			log.Printf("error creating file: %v\n", err)
			http.Error(
				w,
				"error creating file: ["+err.Error()+"]",
				http.StatusInternalServerError,
			)
			return
		}
		defer f.Close()
		_, err = io.Copy(f, file)
		if err != nil {
			log.Printf("error uploading file [%s]: %v\n",
				handler.Filename,
				err,
			)
			http.Error(
				w,
				fmt.Sprintf("error uploading file [%s]: %v\n",
					handler.Filename,
					err,
				),
				http.StatusInternalServerError,
			)
			return
		}

		err = f.Close()
		if err != nil {
			log.Printf("error closing file [%s]: %v\n",
				handler.Filename,
				err,
			)
			http.Error(
				w,
				fmt.Sprintf("error closing file [%s]: %v\n",
					handler.Filename,
					err,
				),
				http.StatusInternalServerError,
			)
			return
		}

	default:
		http.Error(
			w,
			"invalid request-method ["+req.Method+"]",
			http.StatusBadRequest,
		)
		return
	}
}

func cnvToHTML(r io.Reader) (io.Reader, error) {
	input, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	unsafe := blackfriday.MarkdownCommon(input)
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return bytes.NewReader(html), err
}

func main() {
	http.HandleFunc("/common-mark-upload", upload)

	log.SetPrefix("[cmark-srv] ")
	log.Printf("listening on http://localhost:7777/common-mark-upload ...\n")
	log.Printf("exit: %v\n", http.ListenAndServe(":7777", nil))
}
