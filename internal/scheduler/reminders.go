package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/OSShip/sessions/internal/events"
	"github.com/OSShip/sessions/internal/store"
)

const (
	checkInterval = 5 * time.Minute
	reminderWindow = 30 * time.Minute
)

func StartReminders(ctx context.Context, st *store.Store, pub *events.Publisher) {
	go runReminders(ctx, st, pub)
}

func runReminders(ctx context.Context, st *store.Store, pub *events.Publisher) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	reminded := &sync.Map{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publishDueReminders(ctx, st, pub, reminded)
		}
	}
}

func publishDueReminders(ctx context.Context, st *store.Store, pub *events.Publisher, reminded *sync.Map) {
	sessions, err := st.ListUpcomingSessions(ctx, reminderWindow)
	if err != nil {
		log.Printf("reminder scheduler: list sessions: %v", err)
		return
	}
	for _, sess := range sessions {
		if _, loaded := reminded.LoadOrStore(sess.ID, true); loaded {
			continue
		}
		studentEmails, _ := st.ListActiveStudentEmails(ctx, sess.ListingID)
		payload := map[string]string{
			"session_id":   sess.ID,
			"listing_id":   sess.ListingID,
			"mentor_email": sess.MentorEmail,
		}
		if len(studentEmails) > 0 {
			payload["student_email"] = studentEmails[0]
		}
		if err := pub.PublishReminderDue(ctx, payload); err != nil {
			log.Printf("reminder scheduler: publish %s: %v", sess.ID, err)
			reminded.Delete(sess.ID)
		}
	}
}
