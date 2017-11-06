package middlewares

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	headerName = "X-Traefik-Backend"
	queryName  = "X-Traefik-Backend"
	cookieName = "_TRAEFIK_BACKEND"
)

// StickyHeader is a middleware that adds X-Traefik-Backend header when sticky
// cookies are used. Also uses X-Traefik-Backend from a query string when a
// cookie is not present but sticky cookies are being used.
type StickyHeader struct {
	next http.Handler
}

// NewStickyHeader is called at start
func NewStickyHeader(next http.Handler) *StickyHeader {
	return &StickyHeader{next}
}

type backendHeaderWriter struct {
	http.ResponseWriter
	backendFromQueryString string
}

func (w *backendHeaderWriter) WriteHeader(status int) {
	if backendLocation := w.getResponseCookieByName(cookieName); backendLocation != "" {
		// Temporary: Deleting /socket.io/ cookie. We're using sticky cookies with / path only.
		tempCookie := &http.Cookie{Name: cookieName, Value: "", Path: "/socket.io", MaxAge: 0, Expires: time.Now().Add(-100 * time.Hour)}
		http.SetCookie(w.ResponseWriter, tempCookie)

		// Found backend location cookie. Adding it to headers.
		w.ResponseWriter.Header().Set(headerName, backendLocation)
	} else if w.backendFromQueryString != "" {
		// Temporary: Deleting /socket.io/ cookie. We're using sticky cookies with / path only.
		tempCookie := &http.Cookie{Name: cookieName, Value: "", Path: "/socket.io", MaxAge: 0, Expires: time.Now().Add(-100 * time.Hour)}
		http.SetCookie(w.ResponseWriter, tempCookie)

		// Backend location from the query string was valid. Add it to Set-Cookie
		// header to ensure cookies and headers are in sync.
		cookie := &http.Cookie{Name: cookieName, Value: w.backendFromQueryString, Path: "/"}
		http.SetCookie(w.ResponseWriter, cookie)
		w.ResponseWriter.Header().Set(headerName, w.backendFromQueryString)
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *backendHeaderWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *backendHeaderWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (sh *StickyHeader) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	backendFromQueryString := ""

	if _, err := req.Cookie(cookieName); err == http.ErrNoCookie {
		// Cookie is not set. Checking query string for the backend.
		queryValues := req.URL.Query()
		if backendLocation := queryValues.Get(queryName); backendLocation != "" {
			// Found the backend from the query string. Storing for later use.
			backendFromQueryString = backendLocation

			// Setting the backend as a cookie for the sticky module to work.
			cookie := &http.Cookie{Name: cookieName, Value: backendLocation}
			req.AddCookie(cookie)
		}
	}

	writer := &backendHeaderWriter{w, backendFromQueryString}
	writer.addOrAppendHeader("Access-Control-Expose-Headers", headerName)
	sh.next.ServeHTTP(writer, req)
}

// Extracted from https://golang.org/src/net/http/cookie.go #readSetCookies for
// extracing cookie value.
func (w *backendHeaderWriter) getResponseCookieByName(name string) string {
	headers := w.ResponseWriter.Header()
	setCookies := headers["Set-Cookie"]

	for _, cookie := range setCookies {
		parts := strings.Split(strings.TrimSpace(cookie), ";")
		if len(parts) == 1 && parts[0] == "" {
			continue
		}

		parts[0] = strings.TrimSpace(parts[0])

		j := strings.Index(parts[0], "=")
		if j < 0 {
			continue
		}

		name, value := parts[0][:j], parts[0][j+1:]

		if name == cookieName {
			return value
		}
	}

	return ""
}

func (w *backendHeaderWriter) addOrAppendHeader(name string, value string) {
	if currentValue := w.ResponseWriter.Header().Get(name); currentValue != "" {
		newValue := strings.Join([]string{currentValue, value}, ", ")
		w.ResponseWriter.Header().Set(name, newValue)
	} else {
		w.ResponseWriter.Header().Set(name, value)
	}
}
