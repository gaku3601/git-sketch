package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "git-sketch"
	app.Usage = "This app echo input arguments"
	app.Version = "0.0.1"

	app.Commands = []cli.Command{
		{
			Name:    "open",
			Aliases: []string{"o"},
			Usage:   "open sketch file dir",
			Action: func(c *cli.Context) error {
				cd, _ := os.Getwd()
				fmt.Println(cd)
				os.Chdir(c.Args().Get(0))
				dest, err := os.Create(cd + "/sketch.zip")
				if err != nil {
					panic(err)
				}

				zipWriter := zip.NewWriter(dest)
				defer zipWriter.Close()

				// zip化
				for _, s := range dirwalk("./") {
					fmt.Println(s)
					if err := addToZip(s, zipWriter); err != nil {
						panic(err)
					}
				}

				// file名の変更
				fp := replaceExt(cd+"/sketch.zip", ".zip", ".sketch")
				// sketchファイル化
				if err := os.Rename(cd+"/sketch.zip", fp); err != nil {
					panic(err)
				}
				// 開く
				exec.Command("open", fp).Run()
				return nil
			},
		},
		{
			Name:    "save",
			Aliases: []string{"s"},
			Usage:   "save sketch file",
			Action: func(c *cli.Context) error {
				fmt.Println(c.Args().Get(0))
				apath, _ := filepath.Abs(c.Args().Get(0))
				fmt.Println(apath)
				fp := replaceExt(apath, ".sketch", ".zip")
				// zipファイル化
				if err := os.Rename(apath, fp); err != nil {
					panic(err)
				}
				fmt.Println(fp)
				// unzip
				_, err := Unzip(fp, filepath.Dir(fp)+"/"+getFileNameWithoutExt(fp))
				if err != nil {
					panic(err)
				}

				fmt.Println("save")
				return nil
			},
		},
	}

	app.Run(os.Args)
}
func dirwalk(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	var paths []string
	for _, file := range files {
		if file.IsDir() {
			paths = append(paths, dirwalk(filepath.Join(dir, file.Name()))...)
			continue
		}
		paths = append(paths, filepath.Join(dir, file.Name()))
	}

	return paths
}

func addToZip(filename string, zipWriter *zip.Writer) error {
	src, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer src.Close()

	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, src)
	if err != nil {
		return err
	}

	return nil
}

func getFileNameWithoutExt(path string) string {
	// Fixed with a nice method given by mattn-san
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}

func replaceExt(filePath, from, to string) string {
	ext := filepath.Ext(filePath)
	if len(from) > 0 && ext != from {
		return filePath
	}
	return filePath[:len(filePath)-len(ext)] + to
}

func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}
