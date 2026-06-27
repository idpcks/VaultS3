package metrics

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kodiqa-Solutions/VaultS3/internal/metadata"
	"github.com/Kodiqa-Solutions/VaultS3/internal/storage"
)

func newCollector(t *testing.T) *Collector {
	t.Helper()
	base := t.TempDir()
	eng, err := storage.NewFileSystem(filepath.Join(base, "data"))
	if err != nil {
		t.Fatalf("fs: %v", err)
	}
	store, err := metadata.NewStore(filepath.Join(base, "meta.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return NewCollector(store, eng)
}

func TestCollectorCounters(t *testing.T) {
	c := newCollector(t)
	c.RecordRequest("GET")
	c.RecordRequest("GET")
	c.RecordRequest("PUT")
	c.RecordError()
	c.RecordBytesIn(100)
	c.RecordBytesOut(250)

	if got := c.TotalRequests(); got != 3 {
		t.Fatalf("TotalRequests=%d want 3", got)
	}
	if got := c.TotalErrors(); got != 1 {
		t.Fatalf("TotalErrors=%d want 1", got)
	}
	if got := c.TotalBytesIn(); got != 100 {
		t.Fatalf("TotalBytesIn=%d want 100", got)
	}
	if got := c.TotalBytesOut(); got != 250 {
		t.Fatalf("TotalBytesOut=%d want 250", got)
	}

	byMethod := c.RequestsByMethod()
	if byMethod["GET"] != 2 || byMethod["PUT"] != 1 {
		t.Fatalf("RequestsByMethod=%v want GET:2 PUT:1", byMethod)
	}
}

func TestCollectorUnknownMethodBucketsToOther(t *testing.T) {
	c := newCollector(t)
	c.RecordRequest("PATCH")
	if c.RequestsByMethod()["OTHER"] != 1 {
		t.Fatalf("unknown method should map to OTHER: %v", c.RequestsByMethod())
	}
}

func TestCollectorServeHTTP(t *testing.T) {
	c := newCollector(t)
	c.RecordRequest("GET")
	c.RecordError()
	c.RecordLatency(50 * time.Millisecond)
	c.RecordBucketRequest("b1", "PUT")

	rec := httptest.NewRecorder()
	c.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))

	if rec.Code != 200 {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`vaults3_requests_total{method="GET"} 1`,
		"vaults3_request_errors_total 1",
		"vaults3_request_duration_seconds_count 1",
		`vaults3_bucket_requests_total{bucket="b1",method="PUT"} 1`,
		"vaults3_uptime_seconds",
		"vaults3_go_goroutines",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("/metrics output missing %q\n---\n%s", want, body)
		}
	}
}
