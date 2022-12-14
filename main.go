package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func main() {
	flag.Parse()
	dir := flag.Arg(0)
	certDir := flag.Arg(1)
	logsDir := flag.Arg(2)
	email := flag.Arg(3)
	h := &handler{
		dir:     dir,
		logsDir: logsDir,
	}
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
	l := manager.Listener()
	err := http.Serve(l, h)
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

type handler struct {
	dir     string
	logsDir string
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := logRequest(r, h.logsDir); err != nil {
		panic(err)
	}
	host := r.Host
	dir := filepath.Join(h.dir, host)
	if r.Method == http.MethodGet {
		http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
	} else {
		writeSuccess(w, r)
	}
}

func writeSuccess(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"ok": true}`)
	} else {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Success!")
	}
}

type RequestLog struct {
	FromIP  string              `json:"from_ip"`
	Method  string              `json:"method"`
	Host    string              `json:"host"`
	Path    string              `json:"path"`
	Query   map[string][]string `json:"query"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

func logRequest(r *http.Request, dir string) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	l := RequestLog{
		FromIP:  r.RemoteAddr,
		Method:  r.Method,
		Host:    r.Host,
		Path:    r.URL.Path,
		Query:   r.URL.Query(),
		Headers: r.Header,
		Body:    body,
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	timestamp := time.Now().UnixNano()
	var logFile string
	for {
		logFile = filepath.Join(dir, strconv.Itoa(int(timestamp)))
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			break
		}
		timestamp++
	}
	return os.WriteFile(logFile, b, os.ModePerm)
}
