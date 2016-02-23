package couchbase

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/couchbase/gocb"
	"github.com/ucosty/cfssl/certdb"
	cferr "github.com/ucosty/cfssl/errors"
	"io/ioutil"
	"time"
)

type CouchbaseAccessor struct {
	bucketName string
	couchbase  *gocb.Cluster
	bucket     *gocb.Bucket
	config     map[string]string
}

type CertificateWrapper struct {
	Type   string                   `json:"type,omitempty"`
	Record certdb.CertificateRecord `json:"record,omitempty"`
}

type OCSPWrapper struct {
	Type   string            `json:"type,omitempty"`
	Record certdb.OCSPRecord `json:"record,omitempty"`
}

const (
	ocspType              = `ocsp`
	certificateType       = `certificate`
	ocspIdTemplate        = ocspType + `:%s:%s`
	certificateIdTemplate = certificateType + `:%s:%s`
	getUnexpiredN1QL      = `SELECT * FROM %s WHERE type='%s' AND STR_TO_MILLIS(record.expiry) > NOW_MILLIS();`
)

func ocspId(serial, aki string) string {
	return fmt.Sprintf(ocspIdTemplate, serial, aki)
}

func certificateId(serial, aki string) string {
	return fmt.Sprintf(certificateIdTemplate, serial, aki)
}

func (d *CouchbaseAccessor) checkBucket() error {
	if d.bucket == nil {
		return cferr.Wrap(cferr.CertStoreError, cferr.Unknown,
			errors.New("Unknown bucket object, please check SetBucket method"))
	}
	return nil
}

func NewAccessor(config string) *CouchbaseAccessor {
	// Load the database configuration
	accessor := new(CouchbaseAccessor)
	err := accessor.LoadConfiguration(config)
	if err != nil {
		return nil
	}

	cb, err := gocb.Connect(accessor.config["uri"])
	if err != nil {
		return nil
	}
	accessor.SetCouchbase(cb)
	accessor.SetBucket(accessor.config["bucket"], accessor.config["password"])
	return accessor
}

func (d *CouchbaseAccessor) LoadConfiguration(config string) error {
	body, err := ioutil.ReadFile(config)
	json.Unmarshal(body, &d.config)

	if _, ok := d.config["uri"]; !ok {
		fmt.Println("Could not find configuration option 'uri' in " + config)
		return nil
	}

	if _, ok := d.config["bucket"]; !ok {
		fmt.Println("Could not find configuration option 'bucket' in " + config)
		return nil
	}

	if _, ok := d.config["password"]; !ok {
		d.config["password"] = ""
	}

	return err
}

func (d *CouchbaseAccessor) SetCouchbase(cb *gocb.Cluster) {
	d.couchbase = cb
}

func (d *CouchbaseAccessor) SetBucket(bucket string, password string) {
	d.bucketName = bucket
	d.bucket, _ = d.couchbase.OpenBucket(d.bucketName, password)
}

// PK is serial + aki
func (d *CouchbaseAccessor) InsertCertificate(cr certdb.CertificateRecord) error {
	err := d.checkBucket()
	if err != nil {
		return err
	}

	_, err = d.bucket.Insert(certificateId(cr.Serial, cr.AKI), &CertificateWrapper{Type: certificateType, Record: cr}, 0)
	return err
}

func (d *CouchbaseAccessor) GetCertificate(serial, aki string) (crs []certdb.CertificateRecord, err error) {
	err = d.checkBucket()
	if err != nil {
		return nil, err
	}

	var certificate CertificateWrapper
	_, err = d.bucket.Get(certificateId(serial, aki), &certificate)
	crs = append(crs, certificate.Record)
	return crs, err
}

func (d *CouchbaseAccessor) GetUnexpiredCertificates() (crs []certdb.CertificateRecord, err error) {
	err = d.checkBucket()
	if err != nil {
		return nil, err
	}

	query := gocb.NewN1qlQuery(fmt.Sprintf(getUnexpiredN1QL, d.bucketName, certificateType)).Consistency(gocb.RequestPlus).AdHoc(false)
	records, err := d.bucket.ExecuteN1qlQuery(query, nil)

	if err != nil {
		return nil, err
	}

	var row interface{}
	for records.Next(&row) {
		certificate_data, _ := row.(map[string]interface{})
		var certificate CertificateWrapper
		certificateJSON, _ := json.Marshal(certificate_data[d.bucketName])
		json.Unmarshal(certificateJSON, &certificate)
		crs = append(crs, certificate.Record)
	}
	records.Close()
	return crs, nil
}

func (d *CouchbaseAccessor) RevokeCertificate(serial, aki string, reasonCode int) error {
	err := d.checkBucket()
	if err != nil {
		return err
	}

	certificateId := certificateId(serial, aki)
	certificate := new(CertificateWrapper)
	cas, _ := d.bucket.Get(certificateId, &certificate)
	certificate.Record.Status = "revoked"
	certificate.Record.RevokedAt = time.Now().UTC()
	certificate.Record.Reason = reasonCode

	_, err = d.bucket.Replace(certificateId, certificate, cas, 0)
	return err
}

func (d *CouchbaseAccessor) InsertOCSP(rr certdb.OCSPRecord) error {
	err := d.checkBucket()
	if err != nil {
		return err
	}

	_, err = d.bucket.Insert(ocspId(rr.Serial, rr.AKI), &OCSPWrapper{Type: ocspType, Record: rr}, 0)
	return err
}

func (d *CouchbaseAccessor) GetOCSP(serial, aki string) (rrs []certdb.OCSPRecord, err error) {
	err = d.checkBucket()
	if err != nil {
		return nil, err
	}

	var ocsp OCSPWrapper
	d.bucket.Get(ocspId(serial, aki), &ocsp)
	rrs = append(rrs, ocsp.Record)
	return rrs, nil
}

func (d *CouchbaseAccessor) GetUnexpiredOCSPs() (rrs []certdb.OCSPRecord, err error) {
	err = d.checkBucket()
	if err != nil {
		return nil, err
	}

	query := gocb.NewN1qlQuery(fmt.Sprintf(getUnexpiredN1QL, d.bucketName, ocspType)).Consistency(gocb.RequestPlus).AdHoc(false)
	records, err := d.bucket.ExecuteN1qlQuery(query, nil)

	if err != nil {
		return nil, err
	}

	var row interface{}
	for records.Next(&row) {
		ocsp_data, _ := row.(map[string]interface{})
		var ocsp OCSPWrapper
		ocspJSON, _ := json.Marshal(ocsp_data[d.bucketName])
		json.Unmarshal(ocspJSON, &ocsp)
		rrs = append(rrs, ocsp.Record)
	}
	records.Close()
	return rrs, nil
}

func (d *CouchbaseAccessor) UpdateOCSP(serial, aki, body string, expiry time.Time) error {
	err := d.checkBucket()
	if err != nil {
		return err
	}

	ocspId := ocspId(serial, aki)
	var ocsp OCSPWrapper
	cas, _ := d.bucket.Get(ocspId, &ocsp)

	ocsp.Record.Body = body
	ocsp.Record.Expiry = expiry

	_, err = d.bucket.Replace(ocspId, &ocsp, cas, 0)
	return err
}

func (d *CouchbaseAccessor) UpsertOCSP(serial, aki, body string, expiry time.Time) error {
	err := d.checkBucket()
	if err != nil {
		return err
	}

	ocspId := ocspId(serial, aki)
	var ocsp OCSPWrapper
	d.bucket.Get(ocspId, &ocsp)

	ocsp.Type = ocspType
	ocsp.Record.AKI = aki
	ocsp.Record.Body = body
	ocsp.Record.Expiry = expiry
	ocsp.Record.Serial = serial

	_, err = d.bucket.Upsert(ocspId, &ocsp, 0)
	return err
}
