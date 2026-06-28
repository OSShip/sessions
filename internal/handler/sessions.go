package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/OSShip/sessions/internal/model"
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
	if err != nil || mentorID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id := uuid.New().String()
	roomName, jitsiURL := h.JitsiRoom(req.ListingID, id)
	sess, err := h.Store.CreateSession(r.Context(), id, req.ListingID, req.ScheduledAt, roomName, jitsiURL)
	if err != nil {
		slog.ErrorContext(r.Context(), "create session failed", "listing_id", req.ListingID, "err", err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []model.Session{}
	}
	WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		ScheduledAt *time.Time `json:"scheduled_at"`
		Status      string     `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateSession(r.Context(), id, req.ScheduledAt, req.Status); err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
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
	if err != nil || !allowed {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	sess, err := h.Store.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	_ = h.Store.RecordJoin(r.Context(), id, userID)
	slog.InfoContext(r.Context(), "session joined", "session_id", id, "user_id", userID)
	WriteJSON(w, http.StatusOK, map[string]interface{}{"jitsi_url": sess.JitsiURL, "room": sess.JitsiRoomName})
}
