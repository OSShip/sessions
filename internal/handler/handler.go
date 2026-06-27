package handler

import (
	"encoding/json"
	"net/http"

	"github.com/OSShip/sessions/internal/events"
	"github.com/OSShip/sessions/internal/jitsi"
	"github.com/OSShip/sessions/internal/store"
)

type Handler struct {
	Store      *store.Store
	Events     *events.Publisher
	JitsiBase  string
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) JitsiRoom(listingID, sessionID string) (string, string) {
	room := jitsi.RoomName(listingID, sessionID)
	return room, jitsi.RoomURL(h.JitsiBase, room)
}
