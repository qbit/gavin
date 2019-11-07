package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/webdav"
	"golang.org/x/sys/unix"
)

var listen string
var davDir string
var staticDir string

func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&listen, "http", ":8080", "Listen on")
	flag.StringVar(&davDir, "dir", dir, "WebDAV directory to serve.")
	flag.StringVar(&staticDir, "static", dir, "Directory to serve static resources from.")
	flag.Parse()

	unix.Unveil(staticDir, "r")
	unix.Unveil(davDir, "rwc")
	err = unix.UnveilBlock()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	wdav := &webdav.Handler{
		Prefix:     "/dav/",
		LockSystem: webdav.NewMemLS(),
		FileSystem: webdav.Dir(davDir),
		Logger: func(r *http.Request, err error) {
			log.Printf("%s : %s - %s", r.Method, r.URL.Path, err)
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	mux.HandleFunc("/dav/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !(ok == true && u == "qbit" && p == "hai") {
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
	log.Panic(s.Serve(lis))
}
