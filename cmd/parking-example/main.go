package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	parking "parkingops"
)

var demoSecret = []byte("parking-demo-secret-32-bytes-v1")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return runDemo()
	}
	switch args[0] {
	case "demo":
		return runDemo()
	case "entry":
		return runEntry(args[1:])
	case "exit":
		return runExit(args[1:])
	default:
		return fmt.Errorf("unknown command %q; use entry, exit, or demo", args[0])
	}
}

func runEntry(args []string) error {
	flags := flag.NewFlagSet("entry", flag.ContinueOnError)
	plate := flags.String("plate", "", "vehicle plate")
	entryTime := flags.String("time", "", "entry time in RFC3339")
	zone := flags.String("zone", "", "parking zone code")
	if err := flags.Parse(args); err != nil {
		return err
	}
	parsedTime, err := time.Parse(time.RFC3339, *entryTime)
	if err != nil {
		return fmt.Errorf("parse entry time: %w", err)
	}
	issuer, err := parking.NewIssuer(demoSecret)
	if err != nil {
		return err
	}
	credential, err := issuer.Issue(*plate, parsedTime, *zone)
	if err != nil {
		return err
	}
	return writeJSON(credential)
}

func runExit(args []string) error {
	flags := flag.NewFlagSet("exit", flag.ContinueOnError)
	token := flags.String("credential", "", "offline credential token")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *token == "" {
		return errors.New("credential is required")
	}
	issuer, err := parking.NewIssuer(demoSecret)
	if err != nil {
		return err
	}
	validator, err := parking.NewExitValidator(issuer, parking.NewMemoryAuditRepository())
	if err != nil {
		return err
	}
	return writeJSON(validator.Validate(*token))
}

func runDemo() error {
	issuer, err := parking.NewIssuer(demoSecret)
	if err != nil {
		return err
	}
	repository := parking.NewMemoryAuditRepository()
	validator, err := parking.NewExitValidator(issuer, repository)
	if err != nil {
		return err
	}
	entryTime := time.Date(2026, time.August, 16, 8, 30, 0, 0, time.UTC)
	credential, err := issuer.Issue("A-12345", entryTime, "P2")
	if err != nil {
		return err
	}
	return writeJSON(struct {
		Credential parking.Credential      `json:"credential"`
		PreCharge  parking.PreChargeStatus `json:"pre_charge"`
		Audit      []parking.AuditRecord   `json:"audit"`
	}{
		Credential: credential,
		PreCharge:  validator.Validate(credential.Token),
		Audit:      repository.All(),
	})
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
