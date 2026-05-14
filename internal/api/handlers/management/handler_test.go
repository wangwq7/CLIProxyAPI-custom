package management

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestAuthenticateManagementKey_LocalhostIPBan_BlocksCorrectKeyDuringBan(t *testing.T) {
	h := &Handler{
		cfg:            &config.Config{},
		failedAttempts: make(map[string]*attemptInfo),
		envSecret:      "test-secret",
	}

	for i := 0; i < 5; i++ {
		allowed, statusCode, errMsg := h.AuthenticateManagementKey("127.0.0.1", true, "wrong-secret")
		if allowed {
			t.Fatalf("expected auth to be denied at attempt %d", i+1)
		}
		if statusCode != http.StatusUnauthorized || errMsg != "invalid management key" {
			t.Fatalf("unexpected auth failure at attempt %d: status=%d msg=%q", i+1, statusCode, errMsg)
		}
	}

	allowed, statusCode, errMsg := h.AuthenticateManagementKey("127.0.0.1", true, "test-secret")
	if allowed {
		t.Fatalf("expected correct key to be denied while banned")
	}
	if statusCode != http.StatusForbidden {
		t.Fatalf("expected forbidden status while banned, got %d", statusCode)
	}
	if !strings.HasPrefix(errMsg, "IP banned due to too many failed attempts. Try again in") {
		t.Fatalf("unexpected banned message: %q", errMsg)
	}
}

func TestAuthenticateManagementKey_ServiceSecret(t *testing.T) {
	h := &Handler{
		cfg:                 &config.Config{},
		failedAttempts:      make(map[string]*attemptInfo),
		allowRemoteOverride: true,
		serviceSecrets:      []string{"service-secret"},
	}

	allowed, statusCode, errMsg := h.AuthenticateManagementKey("10.0.0.20", false, "service-secret")
	if !allowed {
		t.Fatalf("expected service secret to authenticate, status=%d msg=%q", statusCode, errMsg)
	}

	allowed, statusCode, errMsg = h.AuthenticateManagementKey("10.0.0.20", false, "wrong-secret")
	if allowed {
		t.Fatalf("expected wrong service secret to be denied")
	}
	if statusCode != http.StatusUnauthorized || errMsg != "invalid management key" {
		t.Fatalf("unexpected auth failure: status=%d msg=%q", statusCode, errMsg)
	}
}

func TestHasServiceSecretConfigured_FromFile(t *testing.T) {
	t.Setenv("MANAGEMENT_SERVICE_TOKEN", "")
	tokenFile := filepath.Join(t.TempDir(), "service-token")
	if err := os.WriteFile(tokenFile, []byte("file-service-secret\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	t.Setenv("MANAGEMENT_SERVICE_TOKEN_FILE", tokenFile)

	if !HasServiceSecretConfigured() {
		t.Fatalf("expected service secret file to enable service auth")
	}

	h := NewHandler(&config.Config{}, "", nil)
	if len(h.serviceSecrets) != 1 || h.serviceSecrets[0] != "file-service-secret" {
		t.Fatalf("service secrets = %#v, want file-service-secret", h.serviceSecrets)
	}
}
