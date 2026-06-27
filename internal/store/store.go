package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/OSShip/sessions/internal/model"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) GetListingMentorID(ctx context.Context, listingID string) (string, error) {
	var mentorID string
	err := s.pool.QueryRow(ctx, `SELECT mentor_id FROM listings WHERE id=$1`, listingID).Scan(&mentorID)
	return mentorID, err
}

func (s *Store) CreateSession(ctx context.Context, id, listingID string, scheduledAt time.Time, roomName, jitsiURL string) (model.Session, error) {
	if id == "" {
		id = uuid.New().String()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO mentorship_sessions (id, listing_id, scheduled_at, jitsi_room_name, jitsi_url) VALUES ($1,$2,$3,$4,$5)`,
		id, listingID, scheduledAt, roomName, jitsiURL)
	if err != nil {
		return model.Session{}, err
	}
	return model.Session{
		ID:            id,
		ListingID:     listingID,
		ScheduledAt:   scheduledAt,
		JitsiRoomName: roomName,
		JitsiURL:      jitsiURL,
		Status:        "scheduled",
	}, nil
}

func (s *Store) ListByListing(ctx context.Context, listingID string) ([]model.Session, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, listing_id, scheduled_at, jitsi_room_name, jitsi_url, status FROM mentorship_sessions WHERE listing_id=$1 ORDER BY scheduled_at`, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Session
	for rows.Next() {
		var sess model.Session
		if err := rows.Scan(&sess.ID, &sess.ListingID, &sess.ScheduledAt, &sess.JitsiRoomName, &sess.JitsiURL, &sess.Status); err != nil {
			continue
		}
		list = append(list, sess)
	}
	return list, nil
}

func (s *Store) UpdateSession(ctx context.Context, id string, scheduledAt *time.Time, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE mentorship_sessions SET scheduled_at=COALESCE($1,scheduled_at), status=COALESCE(NULLIF($2,''),status), updated_at=NOW() WHERE id=$3`,
		scheduledAt, status, id)
	return err
}

func (s *Store) GetSession(ctx context.Context, id string) (model.Session, error) {
	var sess model.Session
	err := s.pool.QueryRow(ctx,
		`SELECT id, listing_id, scheduled_at, jitsi_room_name, jitsi_url, status FROM mentorship_sessions WHERE id=$1`, id).
		Scan(&sess.ID, &sess.ListingID, &sess.ScheduledAt, &sess.JitsiRoomName, &sess.JitsiURL, &sess.Status)
	return sess, err
}

func (s *Store) CanAccessSession(ctx context.Context, sessionID, userID string) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM mentorship_sessions ms
			JOIN listings l ON l.id = ms.listing_id
			WHERE ms.id = $1 AND l.mentor_id = $2
		) OR EXISTS(
			SELECT 1 FROM mentorship_sessions ms
			JOIN enrollments e ON e.listing_id = ms.listing_id
			WHERE ms.id = $1 AND e.student_id = $2 AND e.status = 'active'
		)`, sessionID, userID).Scan(&allowed)
	return allowed, err
}

func (s *Store) RecordJoin(ctx context.Context, sessionID, userID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO session_attendance (session_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, sessionID, userID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE mentorship_sessions SET status='live' WHERE id=$1 AND status='scheduled'`, sessionID)
	return err
}

func (s *Store) CanAccessEnrollment(ctx context.Context, enrollmentID, userID string) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM enrollments e WHERE e.id = $1 AND e.student_id = $2
		) OR EXISTS(
			SELECT 1 FROM enrollments e
			JOIN listings l ON l.id = e.listing_id
			WHERE e.id = $1 AND l.mentor_id = $2
		)`, enrollmentID, userID).Scan(&allowed)
	return allowed, err
}

func (s *Store) AddProgress(ctx context.Context, enrollmentID, note, prURL string) (string, error) {
	id := uuid.New().String()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO progress_entries (id, enrollment_id, note, pr_url) VALUES ($1,$2,$3,$4)`,
		id, enrollmentID, note, nullStr(prURL))
	return id, err
}

func (s *Store) ListProgress(ctx context.Context, enrollmentID string) ([]model.ProgressEntry, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, enrollment_id, COALESCE(note,''), COALESCE(pr_url,''), created_at FROM progress_entries WHERE enrollment_id=$1 ORDER BY created_at`,
		enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ProgressEntry
	for rows.Next() {
		var e model.ProgressEntry
		if err := rows.Scan(&e.ID, &e.EnrollmentID, &e.Note, &e.PRURL, &e.CreatedAt); err != nil {
			continue
		}
		list = append(list, e)
	}
	return list, nil
}

type UpcomingSession struct {
	ID         string
	ListingID  string
	MentorID   string
	MentorEmail string
}

func (s *Store) ListUpcomingSessions(ctx context.Context, within time.Duration) ([]UpcomingSession, error) {
	deadline := time.Now().UTC().Add(within)
	rows, err := s.pool.Query(ctx, `
		SELECT ms.id, ms.listing_id, l.mentor_id, COALESCE(mu.email,'')
		FROM mentorship_sessions ms
		JOIN listings l ON l.id = ms.listing_id
		JOIN users mu ON mu.id = l.mentor_id
		WHERE ms.status = 'scheduled'
		  AND ms.scheduled_at > NOW()
		  AND ms.scheduled_at <= $1`, deadline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UpcomingSession
	for rows.Next() {
		var u UpcomingSession
		if err := rows.Scan(&u.ID, &u.ListingID, &u.MentorID, &u.MentorEmail); err != nil {
			continue
		}
		list = append(list, u)
	}
	return list, nil
}

func (s *Store) ListActiveStudentEmails(ctx context.Context, listingID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.email FROM enrollments e
		JOIN users u ON u.id = e.student_id
		WHERE e.listing_id = $1 AND e.status = 'active'`, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			continue
		}
		emails = append(emails, email)
	}
	return emails, nil
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
