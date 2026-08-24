package habits

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"microhabits/internal/auth"
)

type Handler struct {
	service     *Service
	authService *auth.Service
}

type habitInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type habitListResponse struct {
	Status string  `json:"status"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
	Total  int     `json:"total"`
	Items  []Habit `json:"items"`
}

type habitSingleResponse struct {
	Status string `json:"status"`
	Habit  Habit  `json:"habit"`
}

type habitMessageResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type completionSingleResponse struct {
	Status     string     `json:"status"`
	Completion Completion `json:"completion"`
}

type completionListResponse struct {
	Status  string       `json:"status"`
	HabitID int64        `json:"habit_id"`
	Page    int          `json:"page"`
	Limit   int          `json:"limit"`
	Total   int          `json:"total"`
	Order   string       `json:"order"`
	Items   []Completion `json:"items"`
}

type authErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}

func (handler *Handler) ListHabits(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	page, limit := parsePagination(request, 1, 20)
	items, total, err := handler.service.ListHabits(request.Context(), user.ID, page, limit, request.URL.Query().Get("sort"), request.URL.Query().Get("order"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}
	writeJSON(w, http.StatusOK, habitListResponse{Status: "ok", Page: page, Limit: limit, Total: total, Items: items})
}

func (handler *Handler) CreateHabit(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	var input habitInput
	if !decodeJSON(w, request, &input) {
		return
	}
	habit, err := handler.service.CreateHabit(request.Context(), user.ID, input.Name, input.Description)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Habit name is required")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, habitSingleResponse{Status: "ok", Habit: habit})
}

func (handler *Handler) GetHabit(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	habitID, err := parseIDParam(request, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_HABIT_ID", "Invalid habit id")
		return
	}
	habit, err := handler.service.GetHabit(request.Context(), user.ID, habitID)
	if err != nil {
		switch {
		case errors.Is(err, ErrHabitNotFound):
			writeError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, habitSingleResponse{Status: "ok", Habit: habit})
}

func (handler *Handler) UpdateHabit(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	habitID, err := parseIDParam(request, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_HABIT_ID", "Invalid habit id")
		return
	}
	var input habitInput
	if !decodeJSON(w, request, &input) {
		return
	}
	habit, err := handler.service.UpdateHabit(request.Context(), user.ID, habitID, input.Name, input.Description)
	if err != nil {
		switch {
		case errors.Is(err, ErrValidation):
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Habit name is required")
		case errors.Is(err, ErrHabitNotFound):
			writeError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, habitSingleResponse{Status: "ok", Habit: habit})
}

func (handler *Handler) DeleteHabit(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	habitID, err := parseIDParam(request, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_HABIT_ID", "Invalid habit id")
		return
	}
	if err := handler.service.DeleteHabit(request.Context(), user.ID, habitID); err != nil {
		switch {
		case errors.Is(err, ErrHabitNotFound):
			writeError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, habitMessageResponse{Status: "ok", Message: "Habit deleted"})
}

func (handler *Handler) ListCompletions(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	habitID, err := parseIDParam(request, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_HABIT_ID", "Invalid habit id")
		return
	}
	page, limit := parsePagination(request, 1, 30)
	items, total, err := handler.service.ListCompletions(request.Context(), user.ID, habitID, page, limit, request.URL.Query().Get("order"))
	if err != nil {
		switch {
		case errors.Is(err, ErrHabitNotFound):
			writeError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, completionListResponse{Status: "ok", HabitID: habitID, Page: page, Limit: limit, Total: total, Order: normalizeOrder(request.URL.Query().Get("order")), Items: items})
}

func (handler *Handler) CreateCompletion(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	habitID, err := parseIDParam(request, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_HABIT_ID", "Invalid habit id")
		return
	}
	completion, err := handler.service.CreateCompletion(request.Context(), user.ID, habitID)
	if err != nil {
		switch {
		case errors.Is(err, ErrHabitNotFound):
			writeError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, completionSingleResponse{Status: "ok", Completion: completion})
}

func (handler *Handler) DeleteCompletion(w http.ResponseWriter, request *http.Request) {
	user, ok := handler.requireAuth(w, request)
	if !ok {
		return
	}
	habitID, err := parseIDParam(request, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_HABIT_ID", "Invalid habit id")
		return
	}
	completionID, err := parseIDParam(request, "completionId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_COMPLETION_ID", "Invalid completion id")
		return
	}
	if err := handler.service.DeleteCompletion(request.Context(), user.ID, habitID, completionID); err != nil {
		switch {
		case errors.Is(err, ErrHabitNotFound):
			writeError(w, http.StatusNotFound, "HABIT_NOT_FOUND", "Habit not found")
		case errors.Is(err, ErrCompletionNotFound):
			writeError(w, http.StatusNotFound, "COMPLETION_NOT_FOUND", "Completion not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, habitMessageResponse{Status: "ok", Message: "Completion deleted"})
}

func (handler *Handler) requireAuth(w http.ResponseWriter, request *http.Request) (auth.User, bool) {
	return handler.authService.AuthenticateRequestForHandler(w, request)
}

func parseIDParam(request *http.Request, name string) (int64, error) {
	value := request.PathValue(name)
	if value == "" {
		return 0, sql.ErrNoRows
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func parsePagination(request *http.Request, fallbackPage, fallbackLimit int) (int, int) {
	page, err := strconv.Atoi(request.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = fallbackPage
	}
	limit, err := strconv.Atoi(request.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = fallbackLimit
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
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
	writeJSON(w, status, authErrorResponse{Status: "error", Code: code, Message: message})
}

func normalizeListOrder(input string) string {
	return strings.TrimSpace(strings.ToLower(input))
}
