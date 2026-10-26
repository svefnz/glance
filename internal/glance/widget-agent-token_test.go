package glance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Both agent-backed widgets must send the configured token, otherwise turning
// on the agent's mandatory authentication blanks them silently.
func TestAgentRequestsCarryToken(t *testing.T) {
	const token = "s3cret"

	var gotAuth []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	if _, err := fetchAgentContainerUsage(context.Background(), server.URL, token); err != nil {
		t.Fatalf("fetchAgentContainerUsage: %v", err)
	}

	if _, err := fetchContainersFromAgent(server.URL, token, nil); err != nil {
		t.Fatalf("fetchContainersFromAgent: %v", err)
	}

	if len(gotAuth) != 2 {
		t.Fatalf("got %d requests, want 2", len(gotAuth))
	}

	for i, auth := range gotAuth {
		if want := "Bearer " + token; auth != want {
			t.Errorf("request %d Authorization = %q, want %q", i, auth, want)
		}
	}
}

// An empty token must leave the header off, so loopback deployments that need
// no credential keep working.
func TestAgentRequestsOmitEmptyToken(t *testing.T) {
	var gotAuth []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	if _, err := fetchAgentContainerUsage(context.Background(), server.URL, ""); err != nil {
		t.Fatalf("fetchAgentContainerUsage: %v", err)
	}

	for i, auth := range gotAuth {
		if auth != "" {
			t.Errorf("request %d Authorization = %q, want empty", i, auth)
		}
	}
}
