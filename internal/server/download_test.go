package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"kaze/internal/config"
	"kaze/internal/media"
)

func TestHandleDownload(t *testing.T) {
	lib := t.TempDir()
	outside := t.TempDir()

	album := filepath.Join(lib, "Artist", "Album")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	track := filepath.Join(album, "01 - Song, \"live\".flac")
	if err := os.WriteFile(track, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Libraries: []config.Library{{FriendlyName: "Music", Path: lib}}}
	s := New(cfg, media.NewLibrary())

	get := func(path string, header http.Header) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/download?path="+url.QueryEscape(path), nil)
		for k, v := range header {
			req.Header[k] = v
		}
		rec := httptest.NewRecorder()
		s.handleDownload(rec, req)
		return rec
	}

	t.Run("file in a library is served as an attachment", func(t *testing.T) {
		rec := get(track, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Body.String(); got != "0123456789" {
			t.Errorf("body = %q", got)
		}
		cd := rec.Header().Get("Content-Disposition")
		if cd != `attachment; filename="01 - Song, \"live\".flac"` {
			t.Errorf("Content-Disposition = %q", cd)
		}
	})

	t.Run("range requests still work", func(t *testing.T) {
		rec := get(track, http.Header{"Range": {"bytes=2-4"}})
		if rec.Code != http.StatusPartialContent || rec.Body.String() != "234" {
			t.Errorf("status = %d, body = %q, want 206 \"234\"", rec.Code, rec.Body.String())
		}
	})

	for name, path := range map[string]string{
		"file outside every library": secret,
		"escape with ..":             filepath.Join(lib, "..", filepath.Base(outside), "secret.txt"),
	} {
		t.Run(name+" is forbidden", func(t *testing.T) {
			if rec := get(path, nil); rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", rec.Code)
			}
		})
	}

	t.Run("directory is not listed", func(t *testing.T) {
		if rec := get(album, nil); rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("missing file is 404", func(t *testing.T) {
		if rec := get(filepath.Join(album, "missing.flac"), nil); rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}
