package httptls

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/grafana/sobek"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/js/promises"
)

type (
	// RootModule is the global module instance that will create Client
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		clock

		vu modules.VU
	}

	clock interface {
		Now() time.Time
	}
)

// Ensure the interfaces are implemented correctly
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{
		clock: clockNowFunc(time.Now),
		vu:    vu,
	}
}

// Exports implements the modules.Instance interface and returns
// the exports of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{Named: map[string]interface{}{
		"isExpired": mi.IsExpired,
		"chain":     mi.Chain,
	}}
}

func (mi *ModuleInstance) IsExpired(target sobek.Value) *sobek.Promise {
	p, resolve, reject := promises.New(mi.vu)
	targetAddr := target.String()
	if targetAddr == "" {
		reject(errors.New("target is required"))
		return p
	} // TODO: parse and validate url
	go func() {
		expired, err := mi.isCertExpired(targetAddr)
		if err != nil {
			reject(err)
			return
		}
		resolve(expired)
	}()
	return p
}

func (mi *ModuleInstance) isCertExpired(target string) (bool, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(
		dialer,
		"tcp",
		target+":443",
		&tls.Config{
			InsecureSkipVerify: true,
		})
	if err != nil {
		return false, err
	}
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) < 1 {
		return false, fmt.Errorf("chain of peer certificates for %s is empty", target)
	}

	now := mi.clock.Now()
	for _, c := range certs {
		if now.After(c.NotAfter) {
			return true, nil
		}
	}
	return false, nil
}

func (mi *ModuleInstance) Chain(target string) *sobek.Promise {
	p, resolve, reject := promises.New(mi.vu)
	go func() {
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		conn, err := tls.DialWithDialer(
			dialer,
			"tcp",
			target+":443",
			&tls.Config{
				InsecureSkipVerify: true,
			})
		if err != nil {
			reject(err)
			return
		}
		peerCerts := conn.ConnectionState().PeerCertificates
		if len(peerCerts) < 1 {
			reject(fmt.Errorf("chain of peer certificates for %s is empty", target))
			return
		}
		jscerts := make([]*sobek.Object, 0, len(peerCerts))
		rt := mi.vu.Runtime()
		for _, c := range peerCerts {
			jsc := JSCert{
				Subject: c.Subject.String(),
				Expires: c.NotAfter.UnixMilli(),
				Isca:    c.IsCA,
			}
			jscerts = append(jscerts, rt.ToValue(jsc).ToObject(rt))
		}
		resolve(jscerts)
	}()
	return p
}

type JSCert struct {
	Subject string
	Expires int64
	Isca    bool
}

type clockNowFunc func() time.Time

func (clockNowFunc) Now() time.Time {
	return time.Now()
}
