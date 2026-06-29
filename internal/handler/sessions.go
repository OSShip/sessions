package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/OSShip/sessions/internal/model"
	"github.com/OSShip/utils/observability"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		ListingID   string    `json:"listing_id"`
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	mentorID, err := h.Store.GetListingMentorID(r.Context(), req.ListingID)
	if err != nil {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify listing mentor", err, "listing_id", req.ListingID)
		return
	}
	if mentorID != userID {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify listing mentor", nil, "listing_id", req.ListingID, "reason", "not_listing_mentor")
		return
	}

	id := uuid.New().String()
	roomName, jitsiURL := h.JitsiRoom(req.ListingID, id)
	sess, err := h.Store.CreateSession(r.Context(), id, req.ListingID, req.ScheduledAt, roomName, jitsiURL)
	if err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "create session", err, "listing_id", req.ListingID)
		return
	}

	studentEmails, _ := h.Store.ListActiveStudentEmails(r.Context(), req.ListingID)
	payload := map[string]string{
		"session_id": sess.ID,
		"listing_id": req.ListingID,
	}
	if len(studentEmails) > 0 {
		payload["student_email"] = studentEmails[0]
	}
	if err := h.Events.PublishScheduled(r.Context(), payload); err != nil {
		slog.WarnContext(r.Context(), "session scheduled event failed", "session_id", sess.ID, "err", err)
	}
	slog.InfoContext(r.Context(), "session created", "session_id", sess.ID, "listing_id", req.ListingID, "mentor_id", userID)
	WriteJSON(w, http.StatusCreated, sess)
}

func (h *Handler) ListByListing(w http.ResponseWriter, r *http.Request) {
	listingID := chi.URLParam(r, "listingId")
	list, err := h.Store.ListByListing(r.Context(), listingID)
	if err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "list sessions by listing", err, "listing_id", listingID)
		return
	}
	if list == nil {
		list = []model.Session{}
	}
	WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req struct {
		ScheduledAt *time.Time `json:"scheduled_at"`
		Status      string     `json:"status"`
		IsActive    *bool      `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	mentorID, err := h.Store.GetSessionListingMentorID(r.Context(), id)
	if err != nil {
		observability.RespondError(w, r, http.StatusForbidden, "only the listing mentor can update this session", "verify session mentor", err, "session_id", id)
		return
	}
	if mentorID != userID {
		observability.RespondError(w, r, http.StatusForbidden, "only the listing mentor can update this session", "verify session mentor", nil, "session_id", id, "reason", "not_listing_mentor")
		return
	}

	status := req.Status
	var isActive *bool

	if req.IsActive != nil {
		isActive = req.IsActive
		if *req.IsActive {
			status = "live"
		}
	}

	if status == "completed" || status == "cancelled" {
		falseVal := false
		isActive = &falseVal
	}

	if status != "" {
		switch status {
		case "completed", "cancelled", "scheduled", "live":
		default:
			http.Error(w, `{"error":"invalid status"}`, http.StatusBadRequest)
			return
		}
	}

	if err := h.Store.UpdateSession(r.Context(), id, req.ScheduledAt, status, isActive); err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "update session", err, "session_id", id)
		return
	}
	slog.InfoContext(r.Context(), "session updated", "session_id", id, "mentor_id", userID, "status", status, "is_active", isActive)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	allowed, err := h.Store.CanAccessSession(r.Context(), id, userID)
	if err != nil {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify session access", err, "session_id", id)
		return
	}
	if !allowed {
		observability.RespondError(w, r, http.StatusForbidden, "forbidden", "verify session access", nil, "session_id", id, "reason", "not_participant")
		return
	}
	sess, err := h.Store.GetSession(r.Context(), id)
	if err != nil {
		observability.RespondError(w, r, http.StatusNotFound, "not found", "get session", err, "session_id", id)
		return
	}

	mentorID, _ := h.Store.GetSessionListingMentorID(r.Context(), id)
	if mentorID != userID && !sess.IsActive {
		http.Error(w, `{"error":"session not active yet — wait for the mentor to start"}`, http.StatusForbidden)
		return
	}

	if sess.Status == "completed" || sess.Status == "cancelled" {
		http.Error(w, `{"error":"session has ended"}`, http.StatusForbidden)
		return
	}

	if err := h.Store.RecordJoin(r.Context(), id, userID); err != nil {
		slog.WarnContext(r.Context(), "record session join failed", "session_id", id, "user_id", userID, "err", err)
	}
	slog.InfoContext(r.Context(), "session joined", "session_id", id, "user_id", userID)
	WriteJSON(w, http.StatusOK, map[string]interface{}{"jitsi_url": sess.JitsiURL, "room": sess.JitsiRoomName})
}
