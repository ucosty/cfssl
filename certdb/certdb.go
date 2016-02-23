package certdb

import (
	"time"
)

// CertificateRecord encodes a certificate and its metadata
// that will be recorded in a database.
type CertificateRecord struct {
	Serial    string    `sql:"serial_number" json:"serial,omitempty"`
	AKI       string    `sql:"authority_key_identifier" json:"authority_key_identifier,omitempty"`
	CALabel   string    `sql:"ca_label" json:"ca_label,omitempty"`
	Status    string    `sql:"status" json:"status,omitempty"`
	Reason    int       `sql:"reason" json:"reason,omitempty"`
	Expiry    time.Time `sql:"expiry" json:"expiry,omitempty"`
	RevokedAt time.Time `sql:"revoked_at" json:"revoked_at,omitempty"`
	PEM       string    `sql:"pem" json:"pem,omitempty"`
}

// OCSPRecord encodes a OCSP response body and its metadata
// that will be recorded in a database.
type OCSPRecord struct {
	Serial string    `sql:"serial_number" json:"serial,omitempty"`
	AKI    string    `sql:"authority_key_identifier" json:"authority_key_identifier,omitempty"`
	Body   string    `sql:"body" json:"body,omitempty"`
	Expiry time.Time `sql:"expiry" json:"expiry,omitempty"`
}

// Accessor abstracts the CRUD of certdb objects from a DB.
type Accessor interface {
	InsertCertificate(cr CertificateRecord) error
	GetCertificate(serial, aki string) ([]CertificateRecord, error)
	GetUnexpiredCertificates() ([]CertificateRecord, error)
	RevokeCertificate(serial, aki string, reasonCode int) error
	InsertOCSP(rr OCSPRecord) error
	GetOCSP(serial, aki string) ([]OCSPRecord, error)
	GetUnexpiredOCSPs() ([]OCSPRecord, error)
	UpdateOCSP(serial, aki, body string, expiry time.Time) error
	UpsertOCSP(serial, aki, body string, expiry time.Time) error
}
