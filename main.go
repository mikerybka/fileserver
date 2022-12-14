package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	"golang.org/x/crypto/bcrypt"
)

func main() {
	flag.Parse()
	publicDir := flag.Arg(0)
	privateDir := flag.Arg(1)
	certDir := flag.Arg(2)
	logsDir := flag.Arg(3)
	authDir := flag.Arg(4)
	email := flag.Arg(5)
	h := &handler{
		publicDir:  publicDir,
		privateDir: privateDir,
		logsDir:    logsDir,
		authDir:    authDir,
	}
	manager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(certDir),
		HostPolicy: func(_ context.Context, host string) error {
			hosts, err := listHosts(publicDir)
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
	publicDir  string
	privateDir string
	logsDir    string
	authDir    string
}

func (h *handler) auth() *auth {
	return &auth{dir: h.authDir}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logRequest(r, h.logsDir)
	a := h.auth()

	if strings.HasPrefix(r.URL.Path, "/auth") {
		http.StripPrefix("/auth", a).ServeHTTP(w, r)
		return
	}

	userID := a.getUserID(r)
	if userID != "public" {
		dir := filepath.Join(h.privateDir, userID, r.Host)
		_, err := os.Stat(filepath.Join(dir, r.URL.Path))
		if err == nil {
			http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
			return
		}
	}
	dir := filepath.Join(h.publicDir, r.Host)
	http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
}

func (a *auth) getUserID(r *http.Request) string {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return "public"
	}
	sessionToken := cookie.Value
	sessionFile := filepath.Join(a.dir, "sessions", sessionToken)
	userID, err := os.ReadFile(sessionFile)
	if err != nil {
		return "public"
	}
	return string(userID)
}

func logRequest(r *http.Request, dir string) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	l := struct {
		FromIP  string              `json:"from_ip"`
		Method  string              `json:"method"`
		Host    string              `json:"host"`
		Path    string              `json:"path"`
		Query   map[string][]string `json:"query"`
		Headers map[string][]string `json:"headers"`
		Body    []byte              `json:"body"`
	}{
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

type auth struct {
	dir string
}

func (a *auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/signup":
		a.signup(w, r)
	case "/login":
		a.login(w, r)
	case "/logout":
		a.logout(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (a *auth) signup(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<form method="POST">
				<input type="text" name="user" placeholder="user">
				<input type="password" name="pass" placeholder="pass">
				<input type="submit" value="Sign up">
			</form>
		`))
		return
	}
	if r.Method == "POST" {
		user := r.PostFormValue("user")
		pass := r.PostFormValue("pass")
		userFile := filepath.Join(a.dir, "users", user)
		_, err := os.ReadFile(userFile)
		if err == nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("user already exists"))
			return
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("failed to hash password"))
			return
		}
		err = os.WriteFile(userFile, passwordHash, os.ModePerm)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("failed to write user file: " + err.Error()))
			return
		}
	}
}

func (a *auth) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<form method="POST">
				<input type="text" name="user" placeholder="user">
				<input type="password" name="pass" placeholder="pass">
				<input type="submit" value="Login">
			</form>
		`))
		return
	}
	if r.Method == "POST" {
		user := r.PostFormValue("user")
		pass := r.PostFormValue("pass")
		userFile := filepath.Join(a.dir, "users", user)
		passwordHash, err := os.ReadFile(userFile)
		if err != nil {
			if os.IsNotExist(err) {
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		err = bcrypt.CompareHashAndPassword(passwordHash, []byte(pass))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sessionToken := generateSessionToken()
		sessionFile := filepath.Join(a.dir, "sessions", sessionToken)
		err = os.WriteFile(sessionFile, []byte(user), os.ModePerm)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    sessionToken,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		w.Write([]byte("Success!"))
	}
}

func generateSessionToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (a *auth) logout(w http.ResponseWriter, r *http.Request) {
}
