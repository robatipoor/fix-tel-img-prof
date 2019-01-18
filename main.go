package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

var port string
var addr string

func init() {
	port = os.Getenv("PORT")
	addr = ""
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("public")))
	mux.HandleFunc("/upload", upload)
	address := fmt.Sprintf("%s:%s", addr, port)
	log.Println("Start Sever ... ", address)
	log.Fatalln(http.ListenAndServe(address, mux))
}

// UploadFile uploads a file to the server
func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		log.Println("GET Method not valide !")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	file, fileHandler, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	mimeType := fileHandler.Header.Get("Content-Type")
	b, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		return
	}
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/png":
		b, err = fixSizeImage(b)
		if err != nil {
			log.Println(err)
			return
		}
	default:
		log.Println("invlide type file")
		return
	}
	w.Header().Set("Content-Type", "image/jpg")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// resize image to fix telegram profile image
func fixSizeImage(b []byte) ([]byte, error) {
	read := bytes.NewReader(b)
	img, f, err := image.Decode(read)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	rec := img.Bounds()
	width := rec.Dx()
	height := rec.Dy()
	log.Printf("Resize Image %d X %d %s", width, height, f)
	if height > width {
		height = width
	} else if width > height {
		width = height
	}
	// Resize the image
	resizeImg := imaging.Fit(img, width, height, imaging.Lanczos)
	// Create a new black background image
	bgImage := imaging.New(width, height, color.Black)
	// paste the resized images into background image.
	img = imaging.PasteCenter(bgImage, resizeImg)
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	var format imaging.Format
	switch f {
	case "jpeg", "jpg":
		format = imaging.JPEG
	case "png":
		format = imaging.PNG
	}
	err = imaging.Encode(writer, img, format)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return buf.Bytes(), nil
}

// read image file form fs
func readFile(filename string) ([]byte, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return b, nil
}

// write image file to fs
func writeFile(filename string, b []byte) (int, error) {
	f, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	defer f.Close()
	n, err := f.Write(b)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return n, nil
}

// resize all image in root directory
func reSizeAllImageDir(root string) error {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		log.Println(err)
		return err
	}
	for _, path := range files[1:] {
		b, err := readFile(path)
		if err != nil {
			log.Println(err)
			return err
		}
		b, err = fixSizeImage(b)
		if err != nil {
			log.Println(err)
			return err
		}
		ext := filepath.Ext(path)
		output := strings.TrimSuffix(path, ext)
		output = fmt.Sprintf("%s%s%s", output, "-resize", ext)
		_, err = writeFile(output, b)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
