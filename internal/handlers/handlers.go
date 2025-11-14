package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/service"

	"github.com/gorilla/mux"
)

type Handler struct {
	service *service.Service
	logger  *logger.Logger
}

func New(service *service.Service, log *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  log,
	}
}

func (h *Handler) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/team/add", h.CreateTeam).Methods("POST")
	r.HandleFunc("/team/update", h.UpdateTeam).Methods("POST")
	r.HandleFunc("/team/get", h.GetTeam).Methods("GET")

	r.HandleFunc("/users/setIsActive", h.SetUserActive).Methods("POST")
	r.HandleFunc("/users/getReview", h.GetUserReviews).Methods("GET")

	r.HandleFunc("/pullRequest/create", h.CreatePR).Methods("POST")
	r.HandleFunc("/pullRequest/merge", h.MergePR).Methods("POST")
	r.HandleFunc("/pullRequest/reassign", h.ReassignReviewer).Methods("POST")

	r.HandleFunc("/stats", h.GetStats).Methods("GET")

	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	var errResp models.ErrorResponse
	errResp.Error.Code = code
	errResp.Error.Message = message
	respondJSON(w, status, errResp)
}

func parseErrorCode(err error) string {
	errMsg := err.Error()
	for _, code := range []string{
		models.ErrCodeTeamExists,
		models.ErrCodePRExists,
		models.ErrCodePRMerged,
		models.ErrCodeNotAssigned,
		models.ErrCodeNoCandidate,
		models.ErrCodeNotFound,
	} {
		if strings.HasPrefix(errMsg, code) {
			return code
		}
	}
	return "INTERNAL_ERROR"
}

func getHTTPStatusForError(code string) int {
	switch code {
	case models.ErrCodeTeamExists, models.ErrCodePRExists:
		return http.StatusBadRequest
	case models.ErrCodeNotFound:
		return http.StatusNotFound
	case models.ErrCodePRMerged, models.ErrCodeNotAssigned, models.ErrCodeNoCandidate:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode create team request: %v", err)
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	team, err := h.service.CreateTeam(r.Context(), req)
	if err != nil {
		code := parseErrorCode(err)
		status := getHTTPStatusForError(code)
		h.logger.Error("Failed to create team: %v", err)
		respondError(w, status, code, err.Error())
		return
	}

	h.logger.Info("Team created successfully: %s", req.TeamName)
	respondJSON(w, http.StatusCreated, map[string]interface{}{"team": team})
}

func (h *Handler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode update team request: %v", err)
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	team, err := h.service.UpdateTeam(r.Context(), req)
	if err != nil {
		code := parseErrorCode(err)
		status := getHTTPStatusForError(code)
		h.logger.Error("Failed to update team: %v", err)
		respondError(w, status, code, err.Error())
		return
	}

	h.logger.Info("Team updated successfully: %s", req.TeamName)
	respondJSON(w, http.StatusOK, map[string]interface{}{"team": team})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	team, err := h.service.GetTeam(r.Context(), teamName)
	if err != nil {
		code := parseErrorCode(err)
		status := getHTTPStatusForError(code)
		respondError(w, status, code, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, team)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req models.SetIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	user, err := h.service.SetUserActive(r.Context(), req)
	if err != nil {
		code := parseErrorCode(err)
		status := getHTTPStatusForError(code)
		respondError(w, status, code, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	prs, err := h.service.GetUserReviews(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	})
}

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	pr, err := h.service.CreatePR(r.Context(), req)
	if err != nil {
		code := parseErrorCode(err)
		status := getHTTPStatusForError(code)

		// PR_EXISTS should return 409 Conflict
		if code == models.ErrCodePRExists {
			status = http.StatusConflict
		}

		respondError(w, status, code, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{"pr": pr})
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req models.MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	pr, err := h.service.MergePR(r.Context(), req)
	if err != nil {
		code := parseErrorCode(err)
		status := getHTTPStatusForError(code)
		respondError(w, status, code, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"pr": pr})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req models.ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	pr, replacedBy, err := h.service.ReassignReviewer(r.Context(), req)
	if err != nil {
		code := parseErrorCode(err)
		status := getHTTPStatusForError(code)
		respondError(w, status, code, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr":          pr,
		"replaced_by": replacedBy,
	})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		h.logger.Error("Failed to get stats: %v", err)
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, models.StatsResponse{Users: stats})
}
