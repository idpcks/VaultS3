package inventory

import (
	"bytes"
	"encoding/csv"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kodiqa-Solutions/VaultS3/internal/metadata"
	"github.com/Kodiqa-Solutions/VaultS3/internal/storage"
)

func TestGenerateBucketReport(t *testing.T) {
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

	objs := []struct{ key, data string }{
		{"a.txt", "hello"},
		{"dir/b.txt", "world!"},
	}
	for _, o := range objs {
		if _, _, err := eng.PutObject("b", o.key, strings.NewReader(o.data), int64(len(o.data))); err != nil {
			t.Fatalf("put %s: %v", o.key, err)
		}
		store.PutObjectMeta(metadata.ObjectMeta{
			Bucket:       "b",
			Key:          o.key,
			Size:         int64(len(o.data)),
			ETag:         "etag-" + o.key,
			ContentType:  "text/plain",
			LastModified: time.Now().UnixNano(),
		})
	}

	r := NewReporter(store, eng, Config{DestBucket: "reports"})
	if err := r.generateBucketReport("b"); err != nil {
		t.Fatalf("generateBucketReport: %v", err)
	}

	// The report lands in the destination bucket under inventory/<bucket>/.
	reports, _, err := eng.ListObjects("reports", "inventory/b/", "", 100)
	if err != nil || len(reports) != 1 {
		t.Fatalf("expected 1 report object, got %d (err=%v)", len(reports), err)
	}

	rc, _, err := eng.GetObject("reports", reports[0].Key)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()

	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatalf("parse CSV: %v", err)
	}
	if len(records) != 3 { // header + 2 objects
		t.Fatalf("CSV has %d rows, want 3", len(records))
	}
	if records[0][0] != "Bucket" || records[0][1] != "Key" || records[0][2] != "Size" {
		t.Fatalf("unexpected header: %v", records[0])
	}
	// Objects are listed sorted by key: a.txt first.
	if records[1][1] != "a.txt" || records[1][2] != "5" {
		t.Fatalf("row 1 = %v, want a.txt size 5", records[1])
	}
	if records[2][1] != "dir/b.txt" || records[2][2] != "6" {
		t.Fatalf("row 2 = %v, want dir/b.txt size 6", records[2])
	}
}
