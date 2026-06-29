package jitsi

import "fmt"

type JitsiInfo struct {
	ApiKey             string
	AppID              string
	PrivateKeyFilepath string
}

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

func SignJWTJitsi(userName, email string, is_moderator bool, jitsi_info JitsiInfo) (string, error) {

	builder, err := NewJaaSJwtBuilder(
		WithDefaults(),
		WithAPIKey(jitsi_info.ApiKey),
		WithUserName(userName),
		WithUserEmail(email),
		WithModerator(is_moderator),
		WithAppID(jitsi_info.AppID),
	)
	if err != nil {
		return "", err
	}

	pk, err := readRsaPrivateKey(jitsi_info.PrivateKeyFilepath)

	if err != nil {
		return "", err
	}

	return builder.SignWith(pk)
}

func RoomURL(baseURL, roomName string) string {
	return fmt.Sprintf("%s/%s", baseURL, roomName)
}
