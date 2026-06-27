package dashboard

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDashboardHandler exercises the SPA routing rules against the embedded
// filesystem: root and extension-less routes fall back to index.html (200),
// while a missing asset with a file extension returns 404.
func TestDashboardHandler(t *testing.T) {
	h := Handler()

	t.Run("root serves html", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/dashboard/", nil))
		if rec.Code != 200 {
			t.Fatalf("root status=%d, want 200", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("root content-type=%q, want text/html", ct)
		}
	})

	t.Run("spa route falls back to index", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/dashboard/buckets/my-bucket", nil))
		if rec.Code != 200 {
			t.Fatalf("spa route status=%d, want 200 (index.html fallback)", rec.Code)
		}
	})

	t.Run("missing asset returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/dashboard/nonexistent-asset-xyz.js", nil))
		if rec.Code != 404 {
			t.Fatalf("missing asset status=%d, want 404", rec.Code)
		}
	})
}
