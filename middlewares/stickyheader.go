package middlewares

import (
	"bufio"
	"net"
	"net/http"
	"strings"
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
}

func (w *backendHeaderWriter) WriteHeader(status int) {
	if backendLocation := w.getResponseCookieByName(cookieName); backendLocation != "" {
		// Found backend location cookie. Adding it to headers.
		w.ResponseWriter.Header().Set(headerName, backendLocation)
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
	if _, err := req.Cookie(cookieName); err == http.ErrNoCookie {
		// Cookie is not set. Checking query string for the backend.
		queryValues := req.URL.Query()
		if backendLocation := queryValues.Get(queryName); backendLocation != "" {
			// Got the backend from the header. Setting it as a cookie for sticky module to work.
			cookie := &http.Cookie{Name: cookieName, Value: backendLocation}
			req.AddCookie(cookie)
		}
	}

	writer := &backendHeaderWriter{w}
	writer.addOrAppendHeader("Access-Control-Expose-Headers", headerName)
	sh.next.ServeHTTP(writer, req)
}

func (w *backendHeaderWriter) getResponseCookieByName(name string) string {
	headers := w.ResponseWriter.Header()
	setCookies := headers["Set-Cookie"]

	for _, cookie := range setCookies {
		elements := strings.Split(cookie, "=")
		name, value := elements[0], elements[1]

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
