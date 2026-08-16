package parking

import "sync"

type ValidationStatus string

const (
	ValidationPending ValidationStatus = "pending_validation"
	ValidationValid   ValidationStatus = "valid"
	ValidationInvalid ValidationStatus = "invalid"
)

type AuditRecord struct {
	Plate     string           `json:"plate,omitempty"`
	EntryTime string           `json:"entry_time,omitempty"`
	ZoneCode  string           `json:"zone_code,omitempty"`
	Status    ValidationStatus `json:"status"`
	Reason    string           `json:"reason,omitempty"`
}

type AuditRepository interface {
	Record(AuditRecord)
	All() []AuditRecord
}

type MemoryAuditRepository struct {
	mu      sync.RWMutex
	records []AuditRecord
}

func NewMemoryAuditRepository() *MemoryAuditRepository {
	return &MemoryAuditRepository{}
}

func (r *MemoryAuditRepository) Record(record AuditRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
}

func (r *MemoryAuditRepository) All() []AuditRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]AuditRecord(nil), r.records...)
}
