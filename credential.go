package parking

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrEmptySecret       = errors.New("credential secret is empty")
	ErrInvalidPlate      = errors.New("plate is required")
	ErrInvalidEntryTime  = errors.New("entry time is required")
	ErrInvalidZoneCode   = errors.New("zone code is required")
	ErrMalformedToken    = errors.New("credential token is malformed")
	ErrInvalidSignature  = errors.New("credential signature is invalid")
	ErrInvalidCredential = errors.New("credential payload is invalid")
)

type Claims struct {
	Plate     string `json:"plate"`
	EntryTime string `json:"entry_time"`
	ZoneCode  string `json:"zone_code"`
}

type Credential struct {
	Token  string `json:"token"`
	Claims Claims `json:"claims"`
}

type Issuer struct {
	secret []byte
}

func NewIssuer(secret []byte) (*Issuer, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}
	return &Issuer{secret: append([]byte(nil), secret...)}, nil
}

func (i *Issuer) Issue(plate string, entryTime time.Time, zoneCode string) (Credential, error) {
	plate = strings.ToUpper(strings.TrimSpace(plate))
	zoneCode = strings.ToUpper(strings.TrimSpace(zoneCode))
	if plate == "" {
		return Credential{}, ErrInvalidPlate
	}
	if entryTime.IsZero() {
		return Credential{}, ErrInvalidEntryTime
	}
	if zoneCode == "" {
		return Credential{}, ErrInvalidZoneCode
	}

	claims := Claims{
		Plate:     plate,
		EntryTime: entryTime.UTC().Truncate(time.Second).Format(time.RFC3339),
		ZoneCode:  zoneCode,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return Credential{}, fmt.Errorf("encode credential: %w", err)
	}

	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signaturePart := base64.RawURLEncoding.EncodeToString(i.signature(payload))
	return Credential{Token: payloadPart + "." + signaturePart, Claims: claims}, nil
}

func (i *Issuer) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Claims{}, ErrMalformedToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrMalformedToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrMalformedToken
	}
	if !hmac.Equal(signature, i.signature(payload)) {
		return Claims{}, ErrInvalidSignature
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidCredential
	}
	if claims.Plate == "" || claims.EntryTime == "" || claims.ZoneCode == "" {
		return Claims{}, ErrInvalidCredential
	}
	if _, err := time.Parse(time.RFC3339, claims.EntryTime); err != nil {
		return Claims{}, ErrInvalidCredential
	}
	return claims, nil
}

func (i *Issuer) signature(payload []byte) []byte {
	mac := hmac.New(sha256.New, i.secret)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}
