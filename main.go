package main

import (
	"crypto/tls"
	"embed"
	_ "embed"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/webdav"
	"suah.dev/protect"
)

//go:embed organice
var content embed.FS

var (
	acmeDomain string
	test       bool
	acmeListen string
	cacheDir   string
	davDir     string
	davPath    string
	listen     string
	passPath   string
	users      map[string]string
)

func init() {
	users = make(map[string]string)
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: come up with better names for things.
	// TODO: should these go in a config file?
	flag.StringVar(&acmeDomain, "domain", "", "Domain to to use for ACME requests.")
	flag.StringVar(&acmeListen, "alisten", ":80", "Listen for acme requests on")
	flag.StringVar(&cacheDir, "cache", fmt.Sprintf("%s/.cache", dir), "Directory in which to store ACME certificates.")
	flag.StringVar(&davDir, "davdir", dir, "Directory to serve over WebDAV.")
	flag.StringVar(&listen, "http", "127.0.0.1:8080", "Listen on")
	flag.StringVar(&passPath, "htpass", fmt.Sprintf("%s/.htpasswd", dir), "Path to .htpasswd file..")
	flag.StringVar(&davPath, "davpath", "/dav/", "Directory containing files to serve over WebDAV.")
	flag.BoolVar(&test, "test", false, "Enable testing mode (uses staging LetsEncrypt).")
	flag.Parse()

	// These are OpenBSD specific protections used to prevent unnecessary file access.
	_ = protect.Pledge("stdio wpath rpath cpath inet dns unveil")
	_ = protect.Unveil(passPath, "r")
	_ = protect.Unveil(davDir, "rwc")
	_ = protect.Unveil(cacheDir, "rwc")
	_ = protect.Unveil("/etc/ssl/cert.pem", "r")
	_ = protect.Unveil("/etc/resolv.conf", "r")
	err = protect.UnveilBlock()
	if err != nil {
		log.Fatal(err)
	}

	p, err := os.Open(passPath)
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	ht := csv.NewReader(p)
	ht.Comma = ':'
	ht.Comment = '#'
	ht.TrimLeadingSpace = true

	entries, err := ht.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for _, parts := range entries {
		users[parts[0]] = parts[1]
	}
}

func authenticate(user string, pass string) bool {
	htpass, exists := users[user]

	if !exists {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(htpass), []byte(pass))
	return err == nil
}

func httpLog(r *http.Request) {
	n := time.Now()
	fmt.Printf("%s (%s) [%s] \"%s %s\" %03d\n",
		r.RemoteAddr,
		n.Format(time.RFC822Z),
		r.Method,
		r.URL.Path,
		r.Proto,
		r.ContentLength,
	)
}

func logger(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpLog(r)
		f(w, r)
	}
}

func main() {
	wdav := &webdav.Handler{
		Prefix:     davPath,
		LockSystem: webdav.NewMemLS(),
		FileSystem: webdav.Dir(davDir),
		Logger: func(r *http.Request, err error) {
			httpLog(r)
		},
	}

	fileServer := http.FileServer(http.FS(content))

	mux := http.NewServeMux()
	mux.HandleFunc("/", logger(func(w http.ResponseWriter, r *http.Request) {
		// embed.FS contains the top level directory 'organice'
		// This modifies the request path to match.
		r.URL.Path = fmt.Sprintf("/organice%s", r.URL.Path)

		httpLog(r)
		fileServer.ServeHTTP(w, r)
	}))

	mux.HandleFunc(davPath, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !(ok && authenticate(user, pass)) {
			w.Header().Set("WWW-Authenticate", `Basic realm="davfs"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		wdav.ServeHTTP(w, r)
	})

	s := http.Server{
		Handler: mux,
	}

	if acmeDomain != "" {
		tlsConfig := acmeHandler(acmeDomain, acmeListen, cacheDir)
		tlsLis, err := tls.Listen("tcp", listen, tlsConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Listening for HTTPS on '%s'", listen)
		log.Panic(s.Serve(tlsLis))
	} else {
		lis, err := net.Listen("tcp", listen)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("Listening for HTTP on '%s'", listen)
		log.Panic(s.Serve(lis))
	}
}

func acmeHandler(domain, listen, cache string) *tls.Config {
	log.Printf("storing certifiates for %q in %q\n", domain, cache)

	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(cache),
		HostPolicy: autocert.HostWhitelist(domain),
	}

	if test {
		m.Client = &acme.Client{
			DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
		}
	}

	// TLS parameters graciously taken from https://github.com/jrick/domain
	tc := m.TLSConfig()
	tc.ServerName = domain
	tc.NextProtos = []string{"http/1.1", acme.ALPNProto}
	tc.MinVersion = tls.VersionTLS12
	tc.CurvePreferences = []tls.CurveID{tls.X25519, tls.CurveP256}
	tc.PreferServerCipherSuites = true
	tc.CipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}

	lis, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("ACME client listening on %s", lis.Addr())

	mHandler := m.HTTPHandler(nil)

	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpLog(r)
			mHandler.ServeHTTP(w, r)
		}),
	}

	go func() {
		log.Panic(s.Serve(lis))
	}()

	return tc
}
