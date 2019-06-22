// Package mkcert wraps the mkcert CLI (https://github.com/FiloSottile/mkcert)
// to provide a programmatic means of generating certificates for local
// services. mkcert output is parsed to find the certificate file locations and
// whether the CA is trusted.
//
// The CA used and trust stores considered are controlled using the CAROOT and
// TRUST_STORES envvars mkcert wants. See mkcert -help for more.
package mkcert

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
)

var (
	// ErrNoDomains is returned by Gen to indicate no domains were requested.
	// mkcert will not be invoked.
	ErrNoDomains = errors.New("mkcert: no domains specified")
)

// Cert points to the certificates generated by mkcert, with additional CA and
// trust info.
type Cert struct {
	// CARoot is the mkcert directory containing its root CA.
	CARoot string
	// Trusted indicates that the root CA is installed in all of the system
	// trust stores. If not, the user will need to run 'mkcert -install' to
	// ensure their browser trusts the certificate we have generated.
	Trusted bool
	// Domains the certificate covers.
	Domains []string
	// File is the filepath of the certificate file.
	File string
	// KeyFile is the filepath of the private key.
	KeyFile string
}

// Exec invokes mkcert to acquire a certificate. A certificate for localhost
// can be requested using:
//
//     mkcert.Exec(Domains("localhost", "::1", "127.0.0.1"))
func Exec(opts ...Opt) (Cert, error) {
	var p params
	for _, o := range opts {
		o(&p)
	}
	if len(p.domains) == 0 {
		return Cert{}, ErrNoDomains
	}

	// Ask mkcert to generate the certificates.
	var args []string
	if p.certFile != "" {
		args = append(args, "-cert-file", p.certFile)
	}
	if p.keyFile != "" {
		args = append(args, "-key-file", p.keyFile)
	}
	cmd := exec.Command("mkcert", append(args, p.domains...)...)
	cmd.Dir = p.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Cert{}, fmt.Errorf("mkcert: %s", err)
	}

	certFile, keyFile := parseFiles(out)
	cert := Cert{
		CARoot:  parseCA(out),
		Trusted: parseTrusted(out),
		Domains: p.domains,
		File:    certFile,
		KeyFile: keyFile,
	}
	if cmd.Dir != "" {
		if !filepath.IsAbs(cert.File) {
			cert.File = filepath.Join(cmd.Dir, cert.File)
		}
		if !filepath.IsAbs(cert.KeyFile) {
			cert.KeyFile = filepath.Join(cmd.Dir, cert.KeyFile)
		}
	}
	if !cert.Trusted && p.requireTrust {
		err = fmt.Errorf("mkcert: CA at %s not trusted, run mkcert -install", cert.CARoot)
	}
	return cert, err
}

type params struct {
	dir          string
	certFile     string
	keyFile      string
	domains      []string
	requireTrust bool
}

type Opt func(*params)

// Domains is the list of domains to generate the certificate for.
func Domains(domains ...string) Opt {
	return func(p *params) { p.domains = domains }
}

// RequireTrusted indicates whether Exec errors if the CA is not trusted.
func RequireTrusted(req bool) Opt {
	return func(p *params) { p.requireTrust = req }
}

// Directory specifies the working directory of mkcert, and is the path relative
// to which CertFile and KeyFile are relative to, if specified. When blank,
// defaults to the current directory.
func Directory(path string) Opt {
	return func(p *params) { p.dir = path }
}

// CertFile overrides the location of the generated certificate.
func CertFile(path string) Opt {
	return func(p *params) { p.certFile = path }
}

// KeyFile overrides the location of the generated private key.
func KeyFile(path string) Opt {
	return func(p *params) { p.keyFile = path }
}

func parseCA(out []byte) string {
	match := caRe.FindSubmatch(out)
	if len(match) < 2 {
		return ""
	}
	return string(match[1])
}

var caRe = regexp.MustCompile(`local CA at "(.+?)" [💥✨]\n`)

func parseTrusted(out []byte) bool {
	return !bytes.Contains(out, []byte("not installed"))
}

func parseFiles(out []byte) (cert, key string) {
	match := fileRe.FindSubmatch(out)
	if len(match) < 3 {
		return "", ""
	}
	return string(match[1]), string(match[2])
}

var fileRe = regexp.MustCompile(`(?m)The certificate is at "(.+?)" and the key at "(.+?)"`)
