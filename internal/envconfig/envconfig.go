// Package envconfig resolves the provider base URL that launched harnesses
// are pointed at. It replaces Ollama's envconfig package: instead of reading
// OLLAMA_HOST, the URL comes from the --url flag or $HARNESS_PROVIDER_URL.
package envconfig

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	// EnvVar is the environment variable consulted for the provider URL.
	EnvVar = "HARNESS_PROVIDER_URL"

	// DefaultURL is used when neither the flag nor the environment is set.
	DefaultURL = "http://127.0.0.1:11434"
)

var (
	mu       sync.RWMutex
	override string
)

// SetBaseURL sets the process-wide provider base URL (used by the --url flag).
func SetBaseURL(u string) {
	mu.Lock()
	defer mu.Unlock()
	override = strings.TrimRight(strings.TrimSpace(u), "/")
}

// BaseURL returns the configured provider base URL:
// flag override, then $HARNESS_PROVIDER_URL, then DefaultURL.
func BaseURL() string {
	mu.RLock()
	o := override
	mu.RUnlock()
	if o != "" {
		return o
	}
	if v := strings.TrimSpace(os.Getenv(EnvVar)); v != "" {
		return strings.TrimRight(v, "/")
	}
	return DefaultURL
}

// Host returns the provider base URL as a *url.URL. It never returns an
// error; an unparseable value falls back to DefaultURL.
func Host() *url.URL {
	u, err := url.Parse(BaseURL())
	if err != nil || u.Scheme == "" || u.Host == "" {
		u, _ = url.Parse(DefaultURL)
	}
	return u
}

// ConnectableHost returns the base URL rewritten so loopback addresses use
// "localhost", mirroring Ollama's preference for a connectable address.
func ConnectableHost() *url.URL {
	u := Host()
	if u.Hostname() == "127.0.0.1" || u.Hostname() == "0.0.0.0" {
		clone := *u
		clone.Host = "localhost" + ":" + u.Port()
		return &clone
	}
	return u
}

// ContextLength returns $HARNESS_CONTEXT_LENGTH (or 0 when unset), mirroring
// Ollama's OLLAMA_CONTEXT_LENGTH knob.
func ContextLength() int {
	n, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("HARNESS_CONTEXT_LENGTH")))
	return n
}
