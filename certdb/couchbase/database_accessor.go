package couchbase

import (
    "fmt"
    "time"
    "encoding/json"
    "github.com/couchbase/gocb"
    "io/ioutil"
    "github.com/cloudflare/cfssl/certdb"
)

type CouchbaseAccessor struct {
    bucketName string
    couchbase *gocb.Cluster
    bucket *gocb.Bucket
    config map[string]string
}

type CertificateWrapper struct {
    DocumentType string
    Record       certdb.CertificateRecord
}

type OCSPWrapper struct {
    DocumentType string
    Record       certdb.OCSPRecord
}

const (
    certificatePrefix = `cert-`
    OCSPPrefix = `ocsp-`
    getUnexpiredOCSPN1QL = `SELECT * FROM %s WHERE DocumentType='ocsp' AND STR_TO_MILLIS(Record.Expiry) > NOW_MILLIS();`
    getUnexpiredCertificatesN1QL = `SELECT * FROM %s WHERE DocumentType='certificate' AND STR_TO_MILLIS(Record.Expiry) > NOW_MILLIS();`
)

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
        fmt.Println("Could not find confiugration option 'uri' in " + config)
        return nil
    }

    if _, ok := d.config["bucket"]; !ok {
        fmt.Println("Could not find confiugration option 'bucket' in " + config)
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

func (d *CouchbaseAccessor) InsertCertificate(cr certdb.CertificateRecord) error {
    _, err := d.bucket.Insert(certificatePrefix + cr.Serial, &CertificateWrapper{DocumentType: "certificate", Record: cr}, 0)
    return err
}

func (d *CouchbaseAccessor) GetCertificate(serial, aki string) (crs []certdb.CertificateRecord, err error) {
    var certificate CertificateWrapper
    _, err = d.bucket.Get(certificatePrefix + serial, &certificate)
    if certificate.Record.AKI != aki {
        return nil, nil
    }
    crs = append(crs, certificate.Record)
    return crs, err
}

func (d *CouchbaseAccessor) GetUnexpiredCertificates() (crs []certdb.CertificateRecord, err error) {
    query := gocb.NewN1qlQuery(fmt.Sprintf(getUnexpiredCertificatesN1QL, d.bucketName))
    records, err := d.bucket.ExecuteN1qlQuery(query, nil)

    var row interface{}
    for records.Next(&row) {
        certificate_data, _ := row.(map[string]interface{})
        var certificate CertificateWrapper
        certificateJSON, _ := json.Marshal(certificate_data[d.bucketName])
        json.Unmarshal(certificateJSON, &certificate)
        crs = append(crs, certificate.Record)
    }
    return crs, nil
}

func (d *CouchbaseAccessor) RevokeCertificate(serial, aki string, reasonCode int) error {
    certificate := new(CertificateWrapper)
    cas, _ := d.bucket.Get(certificatePrefix + serial, &certificate)
    if certificate.Record.AKI != aki {
        return nil
    }
    certificate.Record.Status = "revoked"
    certificate.Record.RevokedAt = time.Now().UTC()
    certificate.Record.Reason = reasonCode

    _, err := d.bucket.Replace(certificatePrefix + serial, certificate, cas, 0)
    return err
}

func (d *CouchbaseAccessor) InsertOCSP(rr certdb.OCSPRecord) error {
    _, err := d.bucket.Insert(OCSPPrefix + rr.Serial, &OCSPWrapper{DocumentType: "ocsp", Record: rr}, 0)
    return err
}

func (d *CouchbaseAccessor) GetOCSP(serial, aki string) (rrs []certdb.OCSPRecord, err error) {
    var ocsp OCSPWrapper
    d.bucket.Get(OCSPPrefix + serial, &ocsp)
    if ocsp.Record.AKI != aki {
        return nil, nil
    }
    rrs = append(rrs, ocsp.Record)
    return rrs, nil
}

func (d *CouchbaseAccessor) GetUnexpiredOCSPs() (rrs []certdb.OCSPRecord, err error) {
    query := gocb.NewN1qlQuery(fmt.Sprintf(getUnexpiredOCSPN1QL, d.bucketName))
    records, err := d.bucket.ExecuteN1qlQuery(query, nil)

    var row interface{}
    for records.Next(&row) {
        ocsp_data, _ := row.(map[string]interface{})
        var ocsp OCSPWrapper
        ocspJSON, _ := json.Marshal(ocsp_data[d.bucketName])
        json.Unmarshal(ocspJSON, &ocsp)
        rrs = append(rrs, ocsp.Record)
    }
    return rrs, nil
}

func (d *CouchbaseAccessor) UpdateOCSP(serial, aki, body string, expiry time.Time) error {
    var ocsp OCSPWrapper
    cas, _ := d.bucket.Get(OCSPPrefix + serial, &ocsp)

    if ocsp.Record.AKI != aki {
        return nil
    }

    ocsp.Record.Body = body
    ocsp.Record.Expiry = expiry

    _, err := d.bucket.Replace(OCSPPrefix + serial, &ocsp, cas, 0)
    return err
}

func (d *CouchbaseAccessor) UpsertOCSP(serial, aki, body string, expiry time.Time) error {
    var ocsp OCSPWrapper
    d.bucket.Get(OCSPPrefix + serial, &ocsp)

    if ocsp.DocumentType != "" && ocsp.Record.AKI != aki {
        return nil
    }

    ocsp.DocumentType = "ocsp"
    ocsp.Record.Body = body
    ocsp.Record.Expiry = expiry

    _, err := d.bucket.Upsert(OCSPPrefix + serial, &ocsp, 0)
    return err
}
