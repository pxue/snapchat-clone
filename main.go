package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	uuid "github.com/satori/go.uuid"
)

var (
	// string id -> file path
	imageCache = map[string]*Image{
		"123456": &Image{
			ID:       "123456",
			Filename: "test.jpeg",
			Path:     "./cache/123456-test.jpeg",
		},
	}
)

type Image struct {
	ID        string
	Filename  string
	Path      string
	OpenedAt  *time.Time
	CreatedAt *time.Time
}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("file")
		if err != nil {
			// TODO: proper status
			fmt.Fprintf(w, "something went wrong: formfile: %v", err)
			return
		}
		defer file.Close()

		ID := uuid.NewV4().String()
		filename := fmt.Sprintf("%s-%s", ID, header.Filename)

		// TODO: max size?

		path := fmt.Sprintf("./cache/%s", filename)
		fw, err := os.Create(path)
		if err != nil {
			// TODO: proper status
			fmt.Fprintf(w, "something went wrong: create: %v", err)
			return
		}

		if _, err := io.Copy(fw, file); err != nil {
			// TODO: proper status
			fmt.Fprintf(w, "something went wrong: copy: %v", err)
			return
		}

		// cache the image
		now := time.Now()
		imageCache[ID] = &Image{
			ID:        ID,
			Filename:  filename,
			Path:      path,
			CreatedAt: &now,
		}

		fmt.Fprintf(w, "http://localhost:3000/get/%s", ID)
	})

	r.Get("/get/{uniqueId}", func(w http.ResponseWriter, r *http.Request) {
		ID := chi.URLParam(r, "uniqueId")
		cached, ok := imageCache[ID]
		if !ok {
			// not found
			fmt.Fprintf(w, "%s not found", ID)
			return
		}

		// TODO: ... mutexes

		// check if expired.
		now := time.Now()
		if cached.OpenedAt == nil {
			// then add opened at to cache
			cached.OpenedAt = &now
		} else if time.Since(*cached.OpenedAt) > 10*time.Second {
			// error, does not exist?
			// not found
			fmt.Fprintf(w, "%s not found", ID)
			return
		}

		file, err := os.Open(cached.Path)
		if err != nil {
			// error, does not exist?
			// not found
			fmt.Fprintf(w, "%s not found", ID)
			return
		}

		// TODO: detect image mimetype
		w.Header().Set("Content-Type", "image/jpeg")
		if _, err := io.Copy(w, file); err != nil {
			// error, does not exist?
			// not found
			fmt.Fprintf(w, "%s not found", ID)
			return
		}
	})

	log.Println("Server launched :3000")
	http.ListenAndServe(":3000", r)
}
