package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/webdav"
	"golang.org/x/sys/unix"
)

var (
	davDir    string
	listen    string
	passPath  string
	prefix    string
	staticDir string
	users     map[string]string
)

func init() {
	users = make(map[string]string)
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&davDir, "dir", dir, "WebDAV directory to serve.")
	flag.StringVar(&listen, "http", ":8080", "Listen on")
	flag.StringVar(&passPath, "htpass", fmt.Sprintf("%s/.htpasswd", dir), "Path to .htpasswd file..")
	flag.StringVar(&prefix, "prefix", "/dav/", "Prefix to serve dav things from.")
	flag.StringVar(&staticDir, "static", dir, "Directory to serve static resources from.")
	flag.Parse()

	unix.Unveil(staticDir, "r")
	unix.Unveil(passPath, "r")
	unix.Unveil(davDir, "rwc")
	err = unix.UnveilBlock()
	if err != nil {
		log.Fatal(err)
	}

	p, err := os.Open(passPath)
	defer p.Close()
	if err != nil {
		log.Fatal(err)
	}

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

func validate(user string, pass string) bool {
	htpass, exists := users[user]

	if !exists {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(htpass), []byte(pass))
	if err == nil {
		return true
	}
	return false
}

func main() {
	wdav := &webdav.Handler{
		Prefix:     prefix,
		LockSystem: webdav.NewMemLS(),
		FileSystem: webdav.Dir(davDir),
		Logger: func(r *http.Request, err error) {
			n := time.Now()
			fmt.Printf("%s [%s] \"%s %s %s\" %03d\n",
				r.RemoteAddr,
				n.Format(time.RFC822Z),
				r.Method,
				r.URL.Path,
				r.Proto,
				r.ContentLength,
			)
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	mux.HandleFunc(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !(ok == true && validate(user, pass)) {
			w.Header().Set("WWW-Authenticate", `Basic realm="davfs"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		wdav.ServeHTTP(w, r)
	}))

	lis, err := net.Listen("tcp", listen)
	if err != nil {
		log.Panic(err)
	}

	s := http.Server{Handler: mux}

	log.Printf("Listening on '%s'", listen)
	log.Panic(s.Serve(lis))
}
