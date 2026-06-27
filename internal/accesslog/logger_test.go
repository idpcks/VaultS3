package accesslog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccessLoggerWritesJSONLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	l, err := NewAccessLogger(path)
	if err != nil {
		t.Fatalf("NewAccessLogger: %v", err)
	}
	l.Log(AccessEntry{Method: "PUT", Bucket: "b", Key: "k", Status: 200, Bytes: 42, ClientIP: "1.2.3.4"})
	l.Log(AccessEntry{Method: "GET", Bucket: "b", Status: 403})
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	var e AccessEntry
	if err := json.Unmarshal([]byte(lines[0]), &e); err != nil {
		t.Fatalf("unmarshal line 0: %v", err)
	}
	if e.Method != "PUT" || e.Bucket != "b" || e.Key != "k" || e.Status != 200 || e.Bytes != 42 || e.ClientIP != "1.2.3.4" {
		t.Fatalf("entry mismatch: %+v", e)
	}
}

func TestAccessLoggerAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")

	l1, err := NewAccessLogger(path)
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	l1.Log(AccessEntry{Method: "GET"})
	l1.Close()

	// Re-opening must append, not truncate.
	l2, err := NewAccessLogger(path)
	if err != nil {
		t.Fatalf("open 2: %v", err)
	}
	l2.Log(AccessEntry{Method: "PUT"})
	l2.Close()

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("append failed: expected 2 lines, got %d", len(lines))
	}
}
