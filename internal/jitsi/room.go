package jitsi

import "fmt"

func RoomName(listingID, sessionID string) string {
	listingPrefix := listingID
	if len(listingPrefix) > 8 {
		listingPrefix = listingPrefix[:8]
	}
	sessionPrefix := sessionID
	if len(sessionPrefix) > 8 {
		sessionPrefix = sessionPrefix[:8]
	}
	return fmt.Sprintf("osship-listing-%s-session-%s", listingPrefix, sessionPrefix)
}

func RoomURL(baseURL, roomName string) string {
	return fmt.Sprintf("%s/%s", baseURL, roomName)
}
