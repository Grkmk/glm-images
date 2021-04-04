package handlers

import (
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/grkmk/glm-images/files"
	"github.com/hashicorp/go-hclog"
)

// Files is a handler for reading and writing files
type Files struct {
	log   hclog.Logger
	store files.Storage
}

// NewFiles creates a new File handler
func NewFiles(s files.Storage, l hclog.Logger) *Files {
	return &Files{store: s, log: l}
}

// UploadRest implements the http.Handler interface
func (f *Files) UploadRest(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	fn := vars["filename"]

	f.log.Info("Handle POST", "id", id, "filename", fn)

	// no need to check for invalid id or filename as the mux router will not send requests
	// here unless they have the correct parameters

	f.saveFile(id, fn, rw, r.Body)
}

func (f *Files) UploadMultipart(responseWriter http.ResponseWriter, request *http.Request) {
	err := request.ParseMultipartForm(128 * 1024)
	if err != nil {
		http.Error(responseWriter, "Expected multipart form data", http.StatusBadRequest)
		return
	}

	_, idErr := strconv.Atoi(request.FormValue("id"))
	if idErr != nil {
		http.Error(responseWriter, "Expected integer id", http.StatusBadRequest)
		return
	}

	file, multipartHeader, err := request.FormFile("file")
	if err != nil {
		http.Error(responseWriter, "Expected file", http.StatusBadRequest)
		return
	}

	f.saveFile(request.FormValue("id"), multipartHeader.Filename, responseWriter, file)
}

func (f *Files) invalidURI(uri string, rw http.ResponseWriter) {
	f.log.Error("Invalid path", "path", uri)
	http.Error(rw, "Invalid file path should be in the format: /[id]/[filepath]", http.StatusBadRequest)
}

// saveFile saves the contents of the request to a file
func (f *Files) saveFile(id, path string, rw http.ResponseWriter, r io.ReadCloser) {
	f.log.Info("Save file for product", "id", id, "path", path)

	fp := filepath.Join(id, path)
	err := f.store.Save(fp, r)
	if err != nil {
		f.log.Error("Unable to save file", "error", err)
		http.Error(rw, "Unable to save file", http.StatusInternalServerError)
	}
}
