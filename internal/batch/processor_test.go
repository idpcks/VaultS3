package batch

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kodiqa-Solutions/VaultS3/internal/metadata"
	"github.com/Kodiqa-Solutions/VaultS3/internal/storage"
)

func newProc(t *testing.T) (*Processor, storage.Engine) {
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
	return NewProcessor(store, eng), eng
}

func put(t *testing.T, eng storage.Engine, bucket, key, data string) {
	t.Helper()
	if _, _, err := eng.PutObject(bucket, key, bytes.NewReader([]byte(data)), int64(len(data))); err != nil {
		t.Fatalf("put %s/%s: %v", bucket, key, err)
	}
}

// waitJob polls until the job finishes. It reads status under the processor lock
// to avoid racing the execute goroutine; once a job is completed/failed the
// goroutine has exited, so the returned *Job is safe to read field-by-field.
func waitJob(t *testing.T, p *Processor, id string) *Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		p.mu.RLock()
		j := p.jobs[id]
		var st string
		if j != nil {
			st = j.Status
		}
		p.mu.RUnlock()
		if j != nil && (st == "completed" || st == "failed") {
			return j
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not finish in time")
	return nil
}

func TestBulkDeleteAll(t *testing.T) {
	p, eng := newProc(t)
	eng.CreateBucketDir("b")
	put(t, eng, "b", "a.txt", "x")
	put(t, eng, "b", "b.txt", "y")
	put(t, eng, "b", "nested/c.txt", "z")

	id := p.Submit(&Job{ID: "j1", Type: JobBulkDelete, Bucket: "b"})
	job := waitJob(t, p, id)

	if job.Status != "completed" {
		t.Fatalf("status=%s err=%s", job.Status, job.Error)
	}
	if job.Total != 3 || job.Progress != 3 {
		t.Fatalf("total=%d progress=%d, want 3/3", job.Total, job.Progress)
	}
	for _, k := range []string{"a.txt", "b.txt", "nested/c.txt"} {
		if eng.ObjectExists("b", k) {
			t.Fatalf("%s should have been deleted", k)
		}
	}
}

func TestBulkDeletePrefixScope(t *testing.T) {
	p, eng := newProc(t)
	eng.CreateBucketDir("b")
	put(t, eng, "b", "logs/1.txt", "x")
	put(t, eng, "b", "data/2.txt", "y")

	id := p.Submit(&Job{ID: "j2", Type: JobBulkDelete, Bucket: "b", Prefix: "logs/"})
	waitJob(t, p, id)

	if eng.ObjectExists("b", "logs/1.txt") {
		t.Fatal("logs/1.txt should be deleted")
	}
	if !eng.ObjectExists("b", "data/2.txt") {
		t.Fatal("data/2.txt outside the prefix should survive")
	}
}

func TestBulkCopy(t *testing.T) {
	p, eng := newProc(t)
	eng.CreateBucketDir("src")
	eng.CreateBucketDir("dst")
	put(t, eng, "src", "f.txt", "hello")

	id := p.Submit(&Job{ID: "j3", Type: JobBulkCopy, Bucket: "src", DstBucket: "dst"})
	job := waitJob(t, p, id)

	if job.Status != "completed" {
		t.Fatalf("status=%s err=%s", job.Status, job.Error)
	}
	if !eng.ObjectExists("dst", "f.txt") {
		t.Fatal("object not copied to destination")
	}
	if !eng.ObjectExists("src", "f.txt") {
		t.Fatal("copy must not remove the source")
	}
}

func TestUnknownJobTypeFails(t *testing.T) {
	p, _ := newProc(t)
	id := p.Submit(&Job{ID: "j4", Type: "bogus", Bucket: "b"})
	job := waitJob(t, p, id)
	if job.Status != "failed" {
		t.Fatalf("expected failed for unknown job type, got %s", job.Status)
	}
}

func TestListJobs(t *testing.T) {
	p, eng := newProc(t)
	eng.CreateBucketDir("b")
	id := p.Submit(&Job{ID: "j5", Type: JobBulkDelete, Bucket: "b"})
	waitJob(t, p, id)
	if len(p.ListJobs()) != 1 {
		t.Fatalf("ListJobs returned %d, want 1", len(p.ListJobs()))
	}
}
