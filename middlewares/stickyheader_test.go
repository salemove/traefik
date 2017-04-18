package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStickyHeaderWhenNoStickiness(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	stickyHeader := NewStickyHeader(handler)
	responseWriter := httptest.NewRecorder()

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	stickyHeader.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode, "should be successful request")
}

func TestStickyHeaderSetWhenResponseHasStickyCookie(t *testing.T) {
	backend := "http://1.2.3.4"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := http.Cookie{Name: "_TRAEFIK_BACKEND", Value: backend}
		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	})
	stickyHeader := NewStickyHeader(handler)
	responseWriter := httptest.NewRecorder()

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	stickyHeader.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode, "should be successful request")
	assert.Equal(t, backend, response.Header.Get("X-Traefik-Backend"), "should have backend header")
	assert.Equal(t, "X-Traefik-Backend", response.Header.Get("Access-Control-Expose-Headers"), "should have backend header")
}

func TestStickyHeaderSetWhenRequestWithBackendHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("_TRAEFIK_BACKEND")
		assert.Equal(t, "http://1.2.3.4", cookie.Value, "should have a request cookie")
		w.WriteHeader(http.StatusOK)
	})
	stickyHeader := NewStickyHeader(handler)
	responseWriter := httptest.NewRecorder()

	request, _ := http.NewRequest("GET", "http://example.com?X-Traefik-Backend=http://1.2.3.4", nil)
	stickyHeader.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode, "should be successful request")
}
