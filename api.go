package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type NoteResponse struct {
	Note
	Path    string `json:"Path"`
	Content string `json:"Content"`
}

type NotePost struct {
	Content string `json:"Content"`
}

func apiHeartbeat(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Heartbeat received."))
}

func apiList(w http.ResponseWriter, r *http.Request) {
	jsonResponse, err := json.Marshal(noteCache)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func apiNote(dir string, w http.ResponseWriter, r *http.Request, readOnly bool) {
	filename := strings.Split(r.URL.Path, "/")[3]

	if r.Method == "POST" {
		if readOnly {
			http.Error(w, "NOTESIUM_DIR is set to read-only mode", http.StatusForbidden)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var notePost NotePost
		if err := json.Unmarshal(body, &notePost); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if notePost.Content == "" {
			http.Error(w, "Content field is required", http.StatusBadRequest)
			return
		}

		if _, ok := noteCache[filename]; !ok {
			http.Error(w, "Note not found", http.StatusNotFound)
			return
		}

		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(path, []byte(notePost.Content), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		noteCache = nil
		populateCache(dir)
	}

	note, ok := noteCache[filename]
	if !ok {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	path := filepath.Join(dir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	noteResponse := NoteResponse{
		Note:    *note,
		Path:    path,
		Content: string(content),
	}

	jsonResponse, err := json.Marshal(noteResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

//////////////////////////////// experimental ////////////////////////////////

type bufferedResponseWriter struct {
	buf bytes.Buffer
	w   http.ResponseWriter
}

func (rw *bufferedResponseWriter) Write(p []byte) (n int, err error) {
	return rw.buf.Write(p)
}

func (rw *bufferedResponseWriter) Flush() {
	rw.w.Write(rw.buf.Bytes())
	if f, ok := rw.w.(http.Flusher); ok {
		f.Flush()
	}
}

func apiRaw(dir string, w http.ResponseWriter, r *http.Request) {
	pathSegments := strings.Split(r.URL.Path, "/")
	if len(pathSegments) < 4 || pathSegments[3] == "" {
		http.Error(w, "no command specified", http.StatusNotFound)
		return
	}
	command := pathSegments[3]

	args := []string{command}
	queryParameters := r.URL.Query()
	for key, values := range queryParameters {
		for _, value := range values {
			if value == "true" {
				arg := fmt.Sprintf("--%s", key)
				args = append(args, arg)
			} else if value != "" && value != "false" {
				arg := fmt.Sprintf("--%s=%s", key, value)
				args = append(args, arg)
			}
		}
	}

	cmd, err := parseOptions(args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writer := &bufferedResponseWriter{w: w}
	defer writer.Flush()

	switch cmd.Name {
	case "list":
		notesiumList(dir, cmd.Options.(listOptions), writer)
	case "links":
		notesiumLinks(dir, cmd.Options.(linksOptions), writer)
	case "lines":
		notesiumLines(dir, cmd.Options.(linesOptions), writer)
	default:
		http.Error(w, fmt.Sprintf("unrecognized command: %s", command), http.StatusBadRequest)
		return
	}
}
