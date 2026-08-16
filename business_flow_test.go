package parking_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	parking "parkingops"
)

var fixtureSecret = []byte("parking-test-secret-32-bytes-v1")

func TestEntryToExitBusinessFlow(t *testing.T) {
	issuer, err := parking.NewIssuer(fixtureSecret)
	if err != nil {
		t.Fatal(err)
	}
	repository := parking.NewMemoryAuditRepository()
	validator, err := parking.NewExitValidator(issuer, repository)
	if err != nil {
		t.Fatal(err)
	}
	entryTime := time.Date(2026, time.August, 16, 8, 30, 0, 0, time.FixedZone("CST", 8*60*60))

	credential, err := issuer.Issue(" a-12345 ", entryTime, " p2 ")
	if err != nil {
		t.Fatal(err)
	}
	status := validator.Validate(credential.Token)

	if !status.CredentialValid {
		t.Fatalf("credential valid = false, reason = %q", status.Reason)
	}
	if status.Status != parking.ValidationValid {
		t.Fatalf("status = %q, want %q", status.Status, parking.ValidationValid)
	}
	if status.ChargeState != parking.ChargeReady {
		t.Fatalf("charge state = %q, want %q", status.ChargeState, parking.ChargeReady)
	}
	if status.Plate != "A-12345" || status.EntryTime != "2026-08-16T00:30:00Z" || status.ZoneCode != "P2" {
		t.Fatalf("unexpected pre-charge status: %+v", status)
	}

	records := repository.All()
	if len(records) != 1 {
		t.Fatalf("audit records = %d, want 1", len(records))
	}
	if records[0].Status != status.Status {
		t.Fatalf("audit status = %q, want returned status %q", records[0].Status, status.Status)
	}
}

func TestTamperedCredentialIsRejectedAndAudited(t *testing.T) {
	issuer, err := parking.NewIssuer(fixtureSecret)
	if err != nil {
		t.Fatal(err)
	}
	repository := parking.NewMemoryAuditRepository()
	validator, err := parking.NewExitValidator(issuer, repository)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := issuer.Issue("B-67890", time.Date(2026, 8, 16, 1, 0, 0, 0, time.UTC), "P3")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(credential.Token, ".")
	parts[1] = "A" + parts[1][1:]
	if parts[1] == strings.Split(credential.Token, ".")[1] {
		parts[1] = "B" + parts[1][1:]
	}

	status := validator.Validate(strings.Join(parts, "."))

	if status.CredentialValid || status.Status != parking.ValidationInvalid || status.ChargeState != parking.ChargeBlocked {
		t.Fatalf("unexpected rejected status: %+v", status)
	}
	records := repository.All()
	if len(records) != 1 || records[0].Status != parking.ValidationInvalid {
		t.Fatalf("unexpected audit records: %+v", records)
	}
}

func TestCredentialIssuanceIsDeterministic(t *testing.T) {
	issuer, err := parking.NewIssuer(fixtureSecret)
	if err != nil {
		t.Fatal(err)
	}
	entryTime := time.Date(2026, 8, 16, 1, 2, 3, 999, time.UTC)
	first, err := issuer.Issue("C-24680", entryTime, "P1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := issuer.Issue("C-24680", entryTime, "P1")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("credentials differ: first = %+v, second = %+v", first, second)
	}
	claims, err := issuer.Verify(first.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims != first.Claims {
		t.Fatalf("verified claims = %+v, want %+v", claims, first.Claims)
	}
}

func TestEntryRejectsInvalidInput(t *testing.T) {
	issuer, err := parking.NewIssuer(fixtureSecret)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		plate string
		time  time.Time
		zone  string
		want  error
	}{
		{name: "plate", time: time.Unix(1, 0), zone: "P1", want: parking.ErrInvalidPlate},
		{name: "time", plate: "D-13579", zone: "P1", want: parking.ErrInvalidEntryTime},
		{name: "zone", plate: "D-13579", time: time.Unix(1, 0), want: parking.ErrInvalidZoneCode},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := issuer.Issue(test.plate, test.time, test.zone)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestMemoryAuditRepositoryReturnsSnapshot(t *testing.T) {
	repository := parking.NewMemoryAuditRepository()
	repository.Record(parking.AuditRecord{Plate: "E-11223", Status: parking.ValidationValid})
	first := repository.All()
	first[0].Status = parking.ValidationInvalid
	second := repository.All()
	if second[0].Status != parking.ValidationValid {
		t.Fatalf("stored status = %q, want %q", second[0].Status, parking.ValidationValid)
	}
}
