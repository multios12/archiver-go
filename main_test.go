package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type argData struct {
	src    string
	dist   string
	ok     string
	format string
	err    string
}

func TestValidateArgs(t *testing.T) {

	var args []string
	argDatas := []argData{
		{src: "./testdata/【コミック】テストデータ 第1巻.zip", dist: "./", format: "webp"},
		{src: "./testdata/", dist: "./", format: "webp"},
		{src: "./testdata/", dist: "./", format: "png"},
		{src: "./testdata/", dist: "./", format: "jpeg"},
	}
	for _, d := range argDatas {
		args = []string{d.src, d.dist}
		flag.CommandLine.Set("f", d.format)
		if err := validateArgs(args); err != nil {
			println(d.src)
			t.Fail()
		}
	}

	errDatas := []argData{
		{src: "./testdata/notfound", dist: "", format: "webp", err: "処理対象ファイルまたはディレクトリ、保存先ディレクトリを指定してください"},
		{src: "./testdata/notfound", dist: "./", format: "webp", err: "処理対象ファイルまたはディレクトリが見つかりません"},
		{src: "./testdata/【コミック】テストデータ 第1巻.zip", dist: "./notfound", format: "webp", err: "保存先ディレクトリが存在しません"},
		{src: "./testdata/【コミック】テストデータ 第1巻.zip", dist: "./testdata/【コミック】テストデータ 第1巻.zip", format: "webp", err: "保存先ディレクトリが不正です"},
		{src: "./testdata/【コミック】テストデータ 第1巻.zip", dist: "./", ok: "./notfound", format: "webp", err: "移動先ディレクトリが存在しません"},
		{src: "./testdata/【コミック】テストデータ 第1巻.zip", dist: "./", format: "lzh", err: "画像フォーマットが不正です"},
	}

	for _, d := range errDatas {
		if d.dist == "" {
			args = []string{d.src}
		} else {
			args = []string{d.src, d.dist}
		}
		flag.CommandLine.Set("f", d.format)
		flag.CommandLine.Set("o", d.ok)
		if err := validateArgs(args); err.Error() != d.err {
			println(d.err)
			t.Fail()
		}
	}
}

func TestTransaction(t *testing.T) {
	srcpath = "./testdata/【コミック】テストデータ 第1巻.zip"
	distpath = os.TempDir()
	format = "webp"

	transaction()
}

func TestReadZipFiles(t *testing.T) {
	files := readZipFiles("./testdata")
	if len(files) != 1 {
		t.Fail()
	}
}

func TestCompressFile(t *testing.T) {
	format = "webp"
	b, _, _ := compressFile("./testdata/【コミック】テストデータ 第1巻.zip")
	if b.Len() == 0 {
		t.Fail()
	}

	format = "png"
	b, _, _ = compressFile("./testdata/【コミック】テストデータ 第1巻.zip")
	if b.Len() == 0 {
		t.Fail()
	}

	format = "jpeg"
	b, _, _ = compressFile("./testdata/【コミック】テストデータ 第1巻.zip")
	if b.Len() == 0 {
		t.Fail()
	}
}

func TestReadFilesFromZip(t *testing.T) {
	_, f, err := readFilesFromZip("./testdata/【コミック】テストデータ 第1巻.zip")
	if err != nil {
		t.Fail()
	}
	if len(f) == 0 {
		t.Fail()
	}

	_, f, err = readFilesFromZip("./testdata/test.txt")
	if err == nil {
		t.Fail()
	}
}

func TestSave(t *testing.T) {
	dir, _ := ioutil.TempDir("", "aaa")
	filename := filepath.Join(dir, "test.zip")
	b, _ := ioutil.ReadFile("./testdata/【コミック】テストデータ 第1巻.zip")

	_, err := save(filename, bytes.NewBuffer(b))
	if err != nil {
		t.Fail()
	}
}

func TestIsImageTest(t *testing.T) {
	if !isImage("filename.png") {
		t.Fail()
	}

	if !isImage("filename.png") {
		t.Fail()
	}

	if !isImage("filename.webp") {
		t.Fail()
	}

	if isImage("filename.bmp") {
		t.Fail()
	}
}

func TestCreateNotDuplicateFilename(t *testing.T) {
	var filename string
	filename = createNotDuplicateFilename("./abc12345678901234.zip")
	if filename != "./abc.zip" {
		t.Fail()
	}

	filename = createNotDuplicateFilename("./testdata/【コミック】テストデータ 第1巻.zip")
	if filename == "./testdata/【コミック】テストデータ 第1巻.zip" {
		t.Fail()
	}
}
