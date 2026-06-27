package scanner

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kodiqa-Solutions/VaultS3/internal/metadata"
	"github.com/Kodiqa-Solutions/VaultS3/internal/storage"
)

func newScanRig(t *testing.T, webhookURL string, failClosed bool) (*Scanner, storage.Engine) {
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
	eng.CreateBucketDir("b")
	store.CreateBucket("b")

	s := NewScanner(store, eng, webhookURL, 1, 5, "quarantine", failClosed, 100*1024*1024, 16)
	return s, eng
}

func TestScannerCleanObjectStays(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer srv.Close()

	s, eng := newScanRig(t, srv.URL, false)
	eng.PutObject("b", "good.txt", strings.NewReader("safe content"), 12)

	s.processJob(ScanJob{Bucket: "b", Key: "good.txt", Size: 12})

	res := s.RecentResults(10)
	if len(res) != 1 || res[0].Status != "clean" {
		t.Fatalf("results=%+v, want one clean", res)
	}
	if !eng.ObjectExists("b", "good.txt") {
		t.Fatal("clean object should remain in place")
	}
}

func TestScannerInfectedObjectQuarantined(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotAcceptable) // 406 == infected
		w.Write([]byte("EICAR-Test-Signature found"))
	}))
	defer srv.Close()

	s, eng := newScanRig(t, srv.URL, false)
	eng.PutObject("b", "evil.txt", strings.NewReader("virus"), 5)

	s.processJob(ScanJob{Bucket: "b", Key: "evil.txt", Size: 5})

	res := s.RecentResults(10)
	if len(res) != 1 || res[0].Status != "infected" {
		t.Fatalf("results=%+v, want one infected", res)
	}
	if eng.ObjectExists("b", "evil.txt") {
		t.Fatal("infected object should be removed from its bucket")
	}
	if !eng.ObjectExists("quarantine", "b/evil.txt") {
		t.Fatal("infected object should be moved to the quarantine bucket")
	}
}

func TestScannerFailClosedQuarantinesOnWebhookError(t *testing.T) {
	// Unreachable webhook (connection refused).
	s, eng := newScanRig(t, "http://127.0.0.1:1/scan", true)
	eng.PutObject("b", "x.txt", strings.NewReader("data"), 4)

	s.processJob(ScanJob{Bucket: "b", Key: "x.txt", Size: 4})

	res := s.RecentResults(10)
	if len(res) != 1 || res[0].Status != "error" {
		t.Fatalf("results=%+v, want one error", res)
	}
	if eng.ObjectExists("b", "x.txt") {
		t.Fatal("fail-closed mode must quarantine on webhook error")
	}
	if !eng.ObjectExists("quarantine", "b/x.txt") {
		t.Fatal("object should be in quarantine after fail-closed error")
	}
}

func TestScannerQueueDepth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s, _ := newScanRig(t, srv.URL, false)
	if s.QueueDepth() != 0 {
		t.Fatalf("queue should start empty, got %d", s.QueueDepth())
	}
	s.Scan("b", "k", 10)
	if s.QueueDepth() != 1 {
		t.Fatalf("queue depth=%d, want 1 after enqueue", s.QueueDepth())
	}

	// Objects larger than maxScanSize are skipped, not enqueued.
	big, _ := newScanRig(t, srv.URL, false)
	big.maxScanSize = 5
	big.Scan("b", "huge", 1000)
	if big.QueueDepth() != 0 {
		t.Fatalf("oversized object should be skipped, queue=%d", big.QueueDepth())
	}
}
