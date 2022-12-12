package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	flag.Parse()
	dir := flag.Arg(0)
	certDir := flag.Arg(1)
	email := flag.Arg(2)
	handler := &Handler{Dir: dir}
	manager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(certDir),
		HostPolicy: func(_ context.Context, host string) error {
			hosts, err := listHosts(dir)
			if err != nil {
				return err
			}
			for _, h := range hosts {
				if h == host {
					return nil
				}
			}
			return fmt.Errorf("host %q not allowed", host)
		},
		Email: email,
	}
	listener := manager.Listener()
	err := http.Serve(listener, handler)
	panic(err)
}

func listHosts(dir string) ([]string, error) {
	hosts := []string{}
	files, err := os.ReadDir(dir)
	if err != nil {
		return hosts, err
	}
	for _, file := range files {
		if file.IsDir() {
			hosts = append(hosts, file.Name())
		}
	}
	return hosts, nil
}

type Handler struct {
	Dir string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	dir := filepath.Join(h.Dir, host)
	http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
}
