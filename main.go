package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var urlStore = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

func generateShortKey() string {
	urlStore.Lock()
	defer urlStore.Unlock()
	shortKey := fmt.Sprintf("%x", len(urlStore.m))
	return shortKey
}

// Validate a URL
func isValidURL(testURL string) bool {
	_, err := url.ParseRequestURI(testURL)
	return err == nil
}

func ShortenURL(c *gin.Context) {
	var requestBody struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate the URL
	if !isValidURL(requestBody.URL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	// Generate a short key
	shortKey := generateShortKey()

	urlStore.Lock()
	urlStore.m[shortKey] = requestBody.URL
	urlStore.Unlock()

	c.JSON(http.StatusOK, gin.H{"short_url": fmt.Sprintf("http://localhost:8080/%s", shortKey)})
}

func GetOriginalURL(c *gin.Context) {
	shortKey := c.Param("shortKey")

	urlStore.RLock()
	originalURL, exists := urlStore.m[shortKey]
	urlStore.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	// Redirect to the original URL
	c.Redirect(http.StatusMovedPermanently, originalURL)
}

func main() {
	r := gin.Default()

	r.POST("/shorten", ShortenURL)

	r.GET("/:shortKey", GetOriginalURL)

	// Run the server on port 8080
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func TestShortenURL(t *testing.T) {
	router := gin.Default()
	router.POST("/shorten", ShortenURL)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "short_url")
}

func TestRedirectURL(t *testing.T) {
	router := gin.Default()
	router.GET("/:shortKey", GetOriginalURL)

	shortKey := "0"
	urlStore.Lock()
	urlStore.m[shortKey] = "https://example.com"
	urlStore.Unlock()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/"+shortKey, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))
}
