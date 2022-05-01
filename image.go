package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"

	"github.com/chai2010/webp"
)

type fileImage struct {
	Filename string
	Image    image.Image
}

// zip.Fileから画像を読み込む
func NewZipInImage(file zip.File) (fileImage, error) {
	i := fileImage{Filename: file.Name}
	var fileReader io.ReadCloser
	fileReader, err := file.Open()
	if err != nil {
		err = errors.New(fmt.Sprintf("%sの展開に失敗しました(%s)", file.Name, err.Error()))
		return i, err
	}
	defer fileReader.Close()

	i.Image, _, err = image.Decode(fileReader)
	if err != nil {
		err = errors.New(fmt.Sprintf("%sのデコードに失敗しました(%s)", file.Name, err.Error()))
	}
	return i, err
}

// 画像をwriterに書き込む
func (i fileImage) Write(writer *zip.Writer) error {
	entryName := i.Filename[:len(i.Filename)-len(filepath.Ext(i.Filename))]
	if format == "jpeg" {
		entryName = entryName + ".jpg"
	} else if format == "png" {
		entryName = entryName + ".png"
	} else {
		entryName = entryName + ".webp"
	}

	header := zip.FileHeader{Name: entryName, Method: zip.Store}
	fileWriter, err := writer.CreateHeader(&header)
	if err != nil {
		return errors.New(fmt.Sprintf("%sのエンコードに失敗しました(%s)", entryName, err.Error()))
	}

	if format == "jpeg" {
		err = jpeg.Encode(fileWriter, i.Image, nil)
	} else if format == "png" {
		err = png.Encode(fileWriter, i.Image)
	} else {
		err = webp.Encode(fileWriter, i.Image, nil)
	}

	if err != nil {
		return errors.New(fmt.Sprintf("%sのエンコードに失敗しました(%s)", entryName, err.Error()))
	}

	i.Image = nil
	return nil
}
