package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/icio/mkcert"
)

func main() {
	// Flags.
	bind := flag.String("b", "localhost:12345", "bind host:addr")
	flag.Parse()

	// Create a temporary directory for the certificate files.
	dir, err := ioutil.TempDir("", "mkcert")
	if err != nil {
		log.Fatal(err)
	}

	// Get our certificate.
	cert, err := mkcert.Exec(
		// Domains tells mkcert what certificate to generate.
		mkcert.Domains("localhost"),
		// RequireTrusted(true) tells Exec to return an error if the CA isn't
		// in the trust stores.
		mkcert.RequireTrusted(true),
		mkcert.Directory(dir),
		// CertFile and KeyFile override the default behaviour of generating
		// the keys in the local directory.
		// mkcert.CertFile(filepath.Join(dir, "cert.pem")),
		// mkcert.KeyFile(filepath.Join(dir, "key.pem")),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Using certificate: %#v", cert)
	log.Printf("✨ https://%s/ ✨", *bind)

	// Launch the server.
	h := http.FileServer(http.Dir("."))
	log.Fatal(http.ListenAndServeTLS(*bind, cert.File, cert.KeyFile, h))
}
