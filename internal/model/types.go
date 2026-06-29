package model

import "time"

type Session struct {
	ID            string    `json:"id"`
	ListingID     string    `json:"listing_id"`
	ScheduledAt   time.Time `json:"scheduled_at"`
	JitsiRoomName string    `json:"jitsi_room_name"`
	JitsiURL      string    `json:"jitsi_url"`
	Status        string    `json:"status"`
	IsActive      bool      `json:"is_active"`
}

type ProgressEntry struct {
	ID           string    `json:"id"`
	EnrollmentID string    `json:"enrollment_id"`
	Note         string    `json:"note,omitempty"`
	PRURL        string    `json:"pr_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
