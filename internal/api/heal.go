package api

import (
	"net/http"
)

// handleHeal handles POST /api/v1/heal — triggers an erasure-coding heal pass.
//
// Optional query params:
//   - bucket: restrict the scan to a single bucket (default: all buckets)
//   - prefix: restrict the scan to object keys under this prefix
//
// The heal runs asynchronously (a full scan can be long-running), so the
// endpoint returns 202 Accepted once the pass has been started.
func (h *APIHandler) handleHeal(w http.ResponseWriter, r *http.Request) {
	if h.ecHealer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "erasure coding is not enabled",
		})
		return
	}

	bucket := r.URL.Query().Get("bucket")
	prefix := r.URL.Query().Get("prefix")

	go h.ecHealer.Heal(bucket, prefix)

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "heal initiated",
		"bucket": bucket,
		"prefix": prefix,
	})
}
