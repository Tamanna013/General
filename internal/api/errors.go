package api

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const (
	CodeObjectNotFound             = "OBJECT_NOT_FOUND"
	CodeVersionNotFound            = "VERSION_NOT_FOUND"
	CodeInvalidRequest             = "INVALID_REQUEST"
	CodeChunkIntegrityFailure      = "CHUNK_INTEGRITY_FAILURE"
	CodeCannotDeleteCurrentVersion = "CANNOT_DELETE_CURRENT_VERSION"
	CodeInvalidRollbackTarget      = "INVALID_ROLLBACK_TARGET"
	CodeInternalError              = "INTERNAL_ERROR"
)

func writeJSONError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	resp := ErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	
	json.NewEncoder(w).Encode(resp)
}
