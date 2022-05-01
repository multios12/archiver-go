package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/nfnt/resize"
)

var srcpath string  // 入力ファイル・ディレクトリパス
var distpath string // 出力ディレクトリパス
var okpath string   // リサイズに成功した処理対象ファイルを移動するパス
var maxHeight uint  // リサイズ後の画像の最大の高さ
var maxWidth uint   // リサイズ後の画像の最大の幅
var format string   // 保存時に使用する画像フォーマット

// 初期化
func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "ZIPファイルから画像を抜き出し、リサイズして再圧縮します\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "archiver source distination [options...]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Source      処理対象ファイルまたはディレクトリ\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Distination 保存先ディレクトリ\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options\n")
		flag.PrintDefaults()
		os.Exit(0)
	}
	srcpath = flag.Arg(0)
	distpath = flag.Arg(1)
	flag.UintVar(&maxHeight, "h", 1200, "リサイズ後の画像の`最大の高さ`")
	flag.UintVar(&maxWidth, "w", 3000, "リサイズ後の画像の`最大の幅`")
	flag.StringVar(&okpath, "o", "", "移動先ディレクトリ\n指定がある場合、処理成功後に処理対象ファイルを移動します")
	flag.StringVar(&format, "f", "webp", "保存時に使用する`画像フォーマット`\nwebp, png, jpeg")
}

// スタートアップポイント
func main() {
	if err := validateArgs(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "開始：%s\n", time.Now().Format("2006-01-02T15:04:05Z07:00"))
	transaction()
	fmt.Fprintf(os.Stdout, "終了：%s\n", time.Now().Format("2006-01-02T15:04:05Z07:00"))
}

// 引数の検証
func validateArgs(args []string) error {
	flag.Parse()

	if len(args) == 0 {
		flag.Usage()
	} else if len(args) < 2 {
		return errors.New("処理対象ファイルまたはディレクトリ、保存先ディレクトリを指定してください")
	} else if _, err := os.Stat(args[0]); err != nil {
		return errors.New("処理対象ファイルまたはディレクトリが見つかりません")
	} else if _, err := os.Stat(args[1]); err != nil {
		return errors.New("保存先ディレクトリが存在しません")
	} else if i, _ := os.Stat(args[1]); !i.IsDir() {
		return errors.New("保存先ディレクトリが不正です")
	} else if okpath != "" {
		if i, err := os.Stat(okpath); err != nil || !i.IsDir() {
			return errors.New("移動先ディレクトリが存在しません")
		}
	}
	if format != "webp" && format != "png" && format != "jpeg" {
		return errors.New("画像フォーマットが不正です")
	}
	return nil
}

func transaction() {
	var filenames []string
	if info, _ := os.Stat(srcpath); !info.IsDir() {
		filenames = append(filenames, srcpath)
	} else {
		filenames = readZipFiles(srcpath)
	}

	for _, filename := range filenames {
		b, count, err := compressFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stdout, "  ・失敗：%s：%s\n", filename, err)
			continue
		}
		distfilename := filepath.Base(filename)
		distfilename = filepath.Join(distpath, distfilename)
		distfilename, err = save(distfilename, b)
		if err != nil {
			fmt.Fprintf(os.Stdout, "  ・失敗：%s：ファイルが保存できませんでした(%s)\n", filename, err)
			continue
		}

		if okpath != "" {
			movefilename := filepath.Base(filename)
			movefilename = filepath.Join(okpath, filename)
			movefilename = createNotDuplicateFilename(movefilename)
			os.Rename(filename, movefilename)
		}

		info, _ := os.Stat(filename)
		fmt.Fprintf(os.Stdout, "  ・成功：%s：%dページ：%d → %dバイト\n", distfilename, count, info.Size(), b.Len())
	}
}

// 指定されたディレクトリから、zipファイルの一覧を取得して返す
func readZipFiles(srcpath string) (files []string) {
	infos, _ := ioutil.ReadDir(srcpath)
	for _, info := range infos {
		if info.IsDir() || strings.ToLower(filepath.Ext(info.Name())) != ".zip" {
			continue
		}
		filename := filepath.Join(srcpath, info.Name())
		files = append(files, filename)
	}
	return
}

// 指定されたファイルを圧縮し、圧縮ファイルのバイト配列を返す
func compressFile(zipFile string) (b *bytes.Buffer, count int, err error) {
	reader, files, err := readFilesFromZip(zipFile)
	if err != nil {
		return
	}
	defer reader.Close()

	b = new(bytes.Buffer)
	writer := zip.NewWriter(b)
	defer writer.Close()
	count = len(files)

	for _, file := range files {
		var i = fileImage{}
		i, err = NewZipInImage(file)
		if err == nil {
			i.Image = resize.Thumbnail(maxWidth, maxHeight, i.Image, resize.NearestNeighbor)
			err = i.Write(writer)
		}
		if err != nil {
			return
		}
	}

	return
}

// 指定されたzipファイルのzipReaderと画像ファイル配列を返す
func readFilesFromZip(filename string) (*zip.ReadCloser, []zip.File, error) {
	filename, _ = filepath.Abs(filename)
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, nil, errors.New("ファイルの読み込みに失敗しました")
	}

	var list []zip.File
	for _, zippedFile := range reader.File {
		if isImage(zippedFile.Name) {
			list = append(list, *zippedFile)
		}
	}

	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return reader, list, nil
}

// 指定されたバイトバッファをファイルに保存する
func save(filename string, b *bytes.Buffer) (string, error) {
	filename = createNotDuplicateFilename(filename)
	writer, err := os.Create(filename)
	if err != nil {
		return filename, err
	}
	defer writer.Close()

	_, err = writer.Write(b.Bytes())
	return filepath.Base(filename), err
}

// ファイルが画像である場合、trueを返す
func isImage(filename string) bool {
	var exts = []string{".png", ".jpg", ".jpeg", ".webp"}
	ext := filepath.Ext(filename)
	ext = strings.ToLower(ext)

	for _, v := range exts {
		if v == ext {
			return true
		}
	}
	return false
}

// 指定されたファイルが既に存在する場合、重複しないファイル名を生成して返す
func createNotDuplicateFilename(filename string) string {
	exp, _ := regexp.Compile(`\d{14}\.`)
	filename = exp.ReplaceAllString(filename, `.`)

	if _, err := os.Stat(filename); err == nil {
		newFilename := filepath.Base(filename)
		newFilename = newFilename[:len(newFilename)-len(filepath.Ext(newFilename))]
		newFilename = newFilename + fmt.Sprintf("[%s]", time.Now().Format("20060102150405"))
		filename = newFilename + filepath.Ext(filename)
		filename = filepath.Join(distpath, filename)
	}

	return filename
}
