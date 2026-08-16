package parking

import "errors"

var (
	ErrNilIssuer          = errors.New("issuer is required")
	ErrNilAuditRepository = errors.New("audit repository is required")
)

type ChargeState string

const (
	ChargeReady   ChargeState = "ready_for_charge"
	ChargeBlocked ChargeState = "blocked"
)

type PreChargeStatus struct {
	CredentialValid bool             `json:"credential_valid"`
	Status          ValidationStatus `json:"status"`
	ChargeState     ChargeState      `json:"charge_state"`
	Plate           string           `json:"plate,omitempty"`
	EntryTime       string           `json:"entry_time,omitempty"`
	ZoneCode        string           `json:"zone_code,omitempty"`
	Reason          string           `json:"reason,omitempty"`
}

type ExitValidator struct {
	issuer *Issuer
	audit  AuditRepository
}

func NewExitValidator(issuer *Issuer, audit AuditRepository) (*ExitValidator, error) {
	if issuer == nil {
		return nil, ErrNilIssuer
	}
	if audit == nil {
		return nil, ErrNilAuditRepository
	}
	return &ExitValidator{issuer: issuer, audit: audit}, nil
}

func (v *ExitValidator) Validate(token string) PreChargeStatus {
	claims, err := v.issuer.Verify(token)
	if err != nil {
		result := PreChargeStatus{
			CredentialValid: false,
			Status:          ValidationInvalid,
			ChargeState:     ChargeBlocked,
			Reason:          err.Error(),
		}
		v.audit.Record(AuditRecord{Status: result.Status, Reason: result.Reason})
		return result
	}

	result := PreChargeStatus{
		CredentialValid: true,
		Status:          ValidationValid,
		ChargeState:     ChargeReady,
		Plate:           claims.Plate,
		EntryTime:       claims.EntryTime,
		ZoneCode:        claims.ZoneCode,
	}
	v.audit.Record(AuditRecord{
		Plate:     claims.Plate,
		EntryTime: claims.EntryTime,
		ZoneCode:  claims.ZoneCode,
		Status:    result.Status,
	})
	return result
}
