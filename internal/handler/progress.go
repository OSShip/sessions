package handler

import (
	"encoding/json"
	"net/http"

	"github.com/OSShip/sessions/internal/model"
	"github.com/OSShip/utils/observability"
)

func (h *Handler) AddProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		EnrollmentID string `json:"enrollment_id"`
		Note         string `json:"note"`
		PRURL        string `json:"pr_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	allowed, err := h.Store.CanAccessEnrollment(r.Context(), req.EnrollmentID, userID)
	if err != nil {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify enrollment access", err, "enrollment_id", req.EnrollmentID)
		return
	}
	if !allowed {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify enrollment access", nil, "enrollment_id", req.EnrollmentID, "reason", "not_participant")
		return
	}
	id, err := h.Store.AddProgress(r.Context(), req.EnrollmentID, req.Note, req.PRURL)
	if err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "add progress", err, "enrollment_id", req.EnrollmentID)
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) ListProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	enrollmentID := r.URL.Query().Get("enrollment_id")
	if enrollmentID == "" {
		http.Error(w, `{"error":"enrollment_id required"}`, http.StatusBadRequest)
		return
	}
	allowed, err := h.Store.CanAccessEnrollment(r.Context(), enrollmentID, userID)
	if err != nil {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify enrollment access", err, "enrollment_id", enrollmentID)
		return
	}
	if !allowed {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify enrollment access", nil, "enrollment_id", enrollmentID, "reason", "not_participant")
		return
	}
	list, err := h.Store.ListProgress(r.Context(), enrollmentID)
	if err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "list progress", err, "enrollment_id", enrollmentID)
		return
	}
	if list == nil {
		list = []model.ProgressEntry{}
	}
	WriteJSON(w, http.StatusOK, list)
}
