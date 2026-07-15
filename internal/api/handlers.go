package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	
	"project-atlas/internal/download"
	"project-atlas/internal/metadata"
	"project-atlas/internal/upload"
	"project-atlas/internal/version"
)

type Handlers struct {
	repo       metadata.Repository
	uploadSvc  *upload.Service
	downSvc    *download.Service
	versionSvc *version.Service
	maxUpload  int64
}

func NewHandlers(repo metadata.Repository, uploadSvc *upload.Service, downSvc *download.Service, versionSvc *version.Service, maxUpload int64) *Handlers {
	return &Handlers{
		repo:       repo,
		uploadSvc:  uploadSvc,
		downSvc:    downSvc,
		versionSvc: versionSvc,
		maxUpload:  maxUpload,
	}
}

// Helper for sending JSON
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// GET /api/v1/objects/{id}
func (h *Handlers) GetObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}

	obj, err := h.repo.GetObject(r.Context(), id)
	if err != nil {
		if errors.Is(err, metadata.ErrObjectNotFound) {
			writeJSONError(w, http.StatusNotFound, CodeObjectNotFound, "Object not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	// We can return the current version info or object info. The prompt for Phase 1 asks for specific shape.
	// We'll return the object structure, or if current version exists, its shape.
	// Actually, let's just return a unified structure if needed, or the Version struct.
	// If the object exists but has no versions (e.g. still uploading), we just return object info.
	if obj.CurrentVersionID != nil {
		v, err := h.repo.GetVersion(r.Context(), *obj.CurrentVersionID)
		if err == nil {
			writeJSON(w, http.StatusOK, v)
			return
		}
	}
	
	writeJSON(w, http.StatusOK, obj)
}

// POST /api/v1/objects
func (h *Handlers) UploadObject(w http.ResponseWriter, r *http.Request) {
	// Parse multipart
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload)
	if err := r.ParseMultipartForm(h.maxUpload); err != nil {
		writeJSONError(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest, "Upload too large or invalid multipart")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Missing 'file' field")
		return
	}
	defer file.Close()

	if header.Size == 0 {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Empty file upload")
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		name = header.Filename
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	v, err := h.uploadSvc.UploadNewObject(r.Context(), name, contentType, header.Size, file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, v)
}

// POST /api/v1/objects/{id}/versions
func (h *Handlers) UploadVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload)
	if err := r.ParseMultipartForm(h.maxUpload); err != nil {
		writeJSONError(w, http.StatusRequestEntityTooLarge, CodeInvalidRequest, "Upload too large or invalid multipart")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Missing 'file' field")
		return
	}
	defer file.Close()

	if header.Size == 0 {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Empty file upload")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	v, err := h.uploadSvc.UploadVersion(r.Context(), id, contentType, header.Size, file)
	if err != nil {
		// Could check for ObjectNotFound etc
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, v)
}

// GET /api/v1/objects/{id}/download
func (h *Handlers) DownloadObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}

	obj, err := h.repo.GetObject(r.Context(), id)
	if err != nil {
		if errors.Is(err, metadata.ErrObjectNotFound) {
			writeJSONError(w, http.StatusNotFound, CodeObjectNotFound, "Object not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	if obj.CurrentVersionID == nil {
		writeJSONError(w, http.StatusNotFound, CodeObjectNotFound, "Object has no committed version")
		return
	}

	v, err := h.repo.GetVersion(r.Context(), *obj.CurrentVersionID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	w.Header().Set("Content-Type", v.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+obj.Name+"\"")

	if err := h.downSvc.DownloadVersion(r.Context(), v.ID, w); err != nil {
		// NOTE: if headers are already sent, writing JSON error doesn't work well
		// We just abort connection or log it. But we can't un-send headers.
		// Handled by returning, since download stream handles errors.
	}
}

// GET /api/v1/objects/{id}/versions/{versionId}/download
func (h *Handlers) DownloadVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	vidStr := chi.URLParam(r, "versionId")
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}
	vid, err := uuid.Parse(vidStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid version ID format")
		return
	}

	obj, err := h.repo.GetObject(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, CodeObjectNotFound, "Object not found")
		return
	}

	v, err := h.repo.GetVersion(r.Context(), vid)
	if err != nil {
		if errors.Is(err, metadata.ErrVersionNotFound) {
			writeJSONError(w, http.StatusNotFound, CodeVersionNotFound, "Version not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	w.Header().Set("Content-Type", v.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+obj.Name+"\"")

	if err := h.downSvc.DownloadVersion(r.Context(), v.ID, w); err != nil {
		// Log and abort.
	}
}

// GET /api/v1/objects/{id}/versions
func (h *Handlers) ListVersions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}

	obj, err := h.repo.GetObject(r.Context(), id)
	if err != nil {
		if errors.Is(err, metadata.ErrObjectNotFound) {
			writeJSONError(w, http.StatusNotFound, CodeObjectNotFound, "Object not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	versions, err := h.versionSvc.ListVersions(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	response := map[string]interface{}{
		"object_id":          obj.ID,
		"current_version_id": obj.CurrentVersionID,
		"versions":           versions,
	}
	
	writeJSON(w, http.StatusOK, response)
}

// GET /api/v1/objects/{id}/versions/{versionId}
func (h *Handlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	vidStr := chi.URLParam(r, "versionId")
	vid, err := uuid.Parse(vidStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid version ID format")
		return
	}

	v, err := h.versionSvc.GetVersion(r.Context(), vid)
	if err != nil {
		if errors.Is(err, metadata.ErrVersionNotFound) {
			writeJSONError(w, http.StatusNotFound, CodeVersionNotFound, "Version not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, v)
}

// POST /api/v1/objects/{id}/versions/{versionId}/rollback
func (h *Handlers) RollbackVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	vidStr := chi.URLParam(r, "versionId")
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}
	vid, err := uuid.Parse(vidStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid version ID format")
		return
	}

	newVer, err := h.versionSvc.Rollback(r.Context(), id, vid)
	if err != nil {
		if err.Error() == "INVALID_ROLLBACK_TARGET" {
			writeJSONError(w, http.StatusBadRequest, CodeInvalidRollbackTarget, "Invalid rollback target")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, newVer)
}

// DELETE /api/v1/objects/{id}/versions/{versionId}
func (h *Handlers) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	vidStr := chi.URLParam(r, "versionId")
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}
	vid, err := uuid.Parse(vidStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid version ID format")
		return
	}

	err = h.versionSvc.DeleteVersion(r.Context(), id, vid)
	if err != nil {
		if err.Error() == "CANNOT_DELETE_CURRENT_VERSION" {
			writeJSONError(w, http.StatusConflict, CodeCannotDeleteCurrentVersion, "Cannot delete current version")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/objects/{id}
func (h *Handlers) DeleteObject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, CodeInvalidRequest, "Invalid object ID format")
		return
	}

	err = h.versionSvc.DeleteObject(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, CodeInternalError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
