package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	service *Service
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Register(w http.ResponseWriter, request *http.Request) {
	var input credentialsRequest
	if !decodeJSON(w, request, &input) {
		return
	}
	user, err := handler.service.Register(request.Context(), input.Email, input.Username, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			writeError(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "Email is already registered")
		case errors.Is(err, ErrValidation):
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid registration data")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok", "user": user})
}

func (handler *Handler) Login(w http.ResponseWriter, request *http.Request) {
	var input credentialsRequest
	if !decodeJSON(w, request, &input) {
		return
	}
	user, token, err := handler.service.Login(request.Context(), input.Email, input.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "token": token, "user": user})
}

func decodeJSON(w http.ResponseWriter, request *http.Request, destination any) bool {
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"status": "error", "code": code, "message": message})
}
