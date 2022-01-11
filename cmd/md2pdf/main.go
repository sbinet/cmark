package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/russross/blackfriday"
)

func main() {
	raw, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	mdExtensions := 0 |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH |
		blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
		blackfriday.EXTENSION_DEFINITION_LISTS

	ltx := blackfriday.LatexRenderer(0)
	tex := blackfriday.Markdown(raw, ltx, mdExtensions)
	tmpdir, err := ioutil.TempDir("", "md2pdf-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	err = ioutil.WriteFile(path.Join(tmpdir, "out.tex"), tex, 0644)
	if err != nil {
		log.Fatal(err)
	}

	buf := new(bytes.Buffer)
	cmd := exec.Command("pdflatex", "-shell-escape", "-output-directory="+tmpdir, path.Join(tmpdir, "out.tex"))
	cmd.Stdout = buf
	cmd.Stderr = buf
	err = cmd.Run()
	if err != nil {
		log.Printf("error running %s %s:\n%v\n",
			cmd.Path, strings.Join(cmd.Args, " "),
			string(buf.Bytes()),
		)
		log.Fatal(err)
	}

	src, err := ioutil.ReadFile(path.Join(tmpdir, "out.pdf"))
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile("out.pdf", src, 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile("out.tex", tex, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
