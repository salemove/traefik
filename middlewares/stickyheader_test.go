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

func TestStickyHeaderSetWhenResponseHasStickyCookieWithPath(t *testing.T) {
	backend := "http://1.2.3.4"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := http.Cookie{Name: "_TRAEFIK_BACKEND", Value: backend, Path: "/path"}
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

func TestStickyHeaderSetsResponseCookieWhenValidCustomHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	stickyHeader := NewStickyHeader(handler)
	responseWriter := httptest.NewRecorder()

	request, _ := http.NewRequest("GET", "http://example.com?X-Traefik-Backend=http://1.2.3.4", nil)
	stickyHeader.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode, "should be successful request")

	cookie := getResponseCookieByName(response, "_TRAEFIK_BACKEND")
	assert.Equal(t, "http://1.2.3.4", cookie, "should use backend from query string")
	assert.Equal(t, "http://1.2.3.4", response.Header.Get("X-Traefik-Backend"), "should have a sticky header")
}

func TestStickyHeaderSetsResponseCookieWhenInvalidCustomHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := http.Cookie{Name: "_TRAEFIK_BACKEND", Value: "http://2.3.4.5"}
		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusOK)
	})
	stickyHeader := NewStickyHeader(handler)
	responseWriter := httptest.NewRecorder()

	request, _ := http.NewRequest("GET", "http://example.com?X-Traefik-Backend=http://1.2.3.4", nil)
	stickyHeader.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode, "should be successful request")

	cookie := getResponseCookieByName(response, "_TRAEFIK_BACKEND")
	assert.Equal(t, "http://2.3.4.5", cookie, "should have a valid backend")
	assert.Equal(t, "http://2.3.4.5", response.Header.Get("X-Traefik-Backend"), "should have a sticky header")
}

func TestStickyHeaderPrefersBackendFromCookie(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("_TRAEFIK_BACKEND")
		assert.Equal(t, "http://0.0.0.2", cookie.Value, "should have a backend from cookie")
		w.WriteHeader(http.StatusOK)
	})
	stickyHeader := NewStickyHeader(handler)
	responseWriter := httptest.NewRecorder()

	request, _ := http.NewRequest("GET", "http://example.com?X-Traefik-Backend=http://0.0.0.1", nil)
	request.AddCookie(&http.Cookie{Name: "_TRAEFIK_BACKEND", Value: "http://0.0.0.2"})
	stickyHeader.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	assert.Equal(t, http.StatusOK, response.StatusCode, "should be successful request")

	responseCookie := getResponseCookieByName(response, "_TRAEFIK_BACKEND")
	assert.Equal(t, "", responseCookie, "should have no response cookie")
	assert.Equal(t, 0, len(response.Header["X-Traefik-Backend"]), "should have no sticky header")
}

func getResponseCookieByName(response *http.Response, name string) string {
	for _, cookie := range response.Cookies() {
		if name == cookie.Name {
			return cookie.Value
		}
	}
	return ""
}
