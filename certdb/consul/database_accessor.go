package consul

import (
    "encoding/json"
    "fmt"
    "log"
    "github.com/hashicorp/consul/api"
    "github.com/ucosty/cfssl/certdb"
    "io/ioutil"
    "time"
)

type ConsulAccessor struct {
    host string
    prefix string
    config map[string]string
    consul *api.Client
}

const (
    ocspPrefix            = `ocsp`
    certificatePrefix     = `certificate`
    ocspIdTemplate        = `%s/%s/%s/` + ocspPrefix
    certificateIdTemplate = `%s/%s/%s/` + certificatePrefix
)

func (d *ConsulAccessor) certificateId(serial, aki string) string {
    return fmt.Sprintf(certificateIdTemplate, d.config["prefix"], serial, aki)
}

func NewAccessor(config string) *ConsulAccessor {
    // Load the database configuration
    accessor := new(ConsulAccessor)
    err := accessor.LoadConfiguration(config)
    if err != nil {
        return nil
    }


    settings := api.DefaultConfig()
    settings.Address = accessor.config["uri"]

    accessor.consul, err = api.NewClient(settings)
    fmt.Println("Connecting to consul at " + settings.Address)
    if err != nil {
        return nil
    }

    return accessor
}

func (d *ConsulAccessor) LoadConfiguration(config string) error {
    body, err := ioutil.ReadFile(config)
    json.Unmarshal(body, &d.config)

    if _, ok := d.config["uri"]; !ok {
        fmt.Println("Could not find configuration option 'uri' in " + config)
        return nil
    }

    if _, ok := d.config["prefix"]; !ok {
        fmt.Println("Could not find configuration option 'prefix' in " + config)
        return nil
    }

    return err
}

func (d *ConsulAccessor) InsertCertificate(cr certdb.CertificateRecord) error {
    log.Printf("Inserting certificate into " + d.certificateId(cr.Serial, cr.AKI))
    certificate, err := json.Marshal(cr)
    if err != nil {
        return err
    }

    d.consul.KV().Put(&api.KVPair{Key: d.certificateId(cr.Serial, cr.AKI), Value: []byte(certificate)}, nil)
    d.GetCertificate(cr.Serial, cr.AKI)
    return err
}

func (d *ConsulAccessor) GetCertificate(serial, aki string) (crs []certdb.CertificateRecord, err error) {
    kv, _, err := d.consul.KV().Get(d.certificateId(serial, aki), nil)
    if err != nil {
        return nil, nil
    }
    certificate := new(certdb.CertificateRecord)
    err = json.Unmarshal(kv.Value, &certificate)
    if err != nil {
        return nil, nil
    }

    crs = append(crs, *certificate)
    return crs, err
}

func (d *ConsulAccessor) GetUnexpiredCertificates() ([]certdb.CertificateRecord, error) {
    return nil, nil
}

func (d *ConsulAccessor) RevokeCertificate(serial, aki string, reasonCode int) error {
    return nil
}

func (d *ConsulAccessor) InsertOCSP(rr certdb.OCSPRecord) error {
    return nil
}

func (d *ConsulAccessor) GetOCSP(serial, aki string) ([]certdb.OCSPRecord, error) {
    return nil, nil
}

func (d *ConsulAccessor) GetUnexpiredOCSPs() ([]certdb.OCSPRecord, error) {
    return nil, nil
}

func (d *ConsulAccessor) UpdateOCSP(serial, aki, body string, expiry time.Time) error {
    return nil
}

func (d *ConsulAccessor) UpsertOCSP(serial, aki, body string, expiry time.Time) error {
    return nil
}

