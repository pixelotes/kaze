package media

import (
	"encoding/base64"
	"path/filepath"
	"strings"
)

// ID prefixes — let clients treat IDs as opaque while letting us
// route lookups without touching the filesystem.
const (
	prefixArtist = "ar-"
	prefixAlbum  = "al-"
	prefixTrack  = "tr-"
	// Library "roots" act as Subsonic music folders. We encode them
	// numerically since the spec expects integer-ish folder IDs.
)

type Track struct {
	ID          string
	Path        string
	Title       string
	TrackNo     int
	DiscNo      int
	Duration    int   // seconds
	Size        int64 // bytes
	BitRate     int   // kbps, 0 if unknown
	Suffix      string
	ContentType string

	AlbumID  string
	ArtistID string
	Album    string
	Artist   string
	Year     int
	Genre    string
}

type Album struct {
	ID       string
	Path     string
	Name     string
	Artist   string
	ArtistID string
	Year     int
	Genre    string
	CoverArt string // path to folder.jpg / cover.jpg / etc, "" if none
	Library  string
	TrackIDs []string // ordered by disc/track number
}

type Artist struct {
	ID       string
	Path     string
	Name     string
	CoverArt string
	Library  string
	AlbumIDs []string // ordered by year then name
}

// EncodePath turns an absolute filesystem path into an opaque,
// URL-safe identifier scoped by entity type.
func EncodePath(prefix, path string) string {
	return prefix + base64.RawURLEncoding.EncodeToString([]byte(path))
}

// DecodeID returns the filesystem path encoded into an entity ID.
// It accepts any of the known prefixes.
func DecodeID(id string) (string, error) {
	for _, p := range []string{prefixArtist, prefixAlbum, prefixTrack} {
		if strings.HasPrefix(id, p) {
			b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(id, p))
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	}
	// Tolerate legacy/unprefixed IDs.
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ArtistID(path string) string { return EncodePath(prefixArtist, path) }
func AlbumID(path string) string  { return EncodePath(prefixAlbum, path) }
func TrackID(path string) string  { return EncodePath(prefixTrack, path) }

// EntityKind returns "artist", "album", "track" or "" based on prefix.
func EntityKind(id string) string {
	switch {
	case strings.HasPrefix(id, prefixArtist):
		return "artist"
	case strings.HasPrefix(id, prefixAlbum):
		return "album"
	case strings.HasPrefix(id, prefixTrack):
		return "track"
	default:
		return ""
	}
}

var audioExts = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/opus",
	".m4a":  "audio/mp4",
	".m4b":  "audio/mp4",
	".aac":  "audio/aac",
	".wav":  "audio/wav",
	".wma":  "audio/x-ms-wma",
}

var imageExts = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
}

func isAudio(name string) bool {
	_, ok := audioExts[strings.ToLower(filepath.Ext(name))]
	return ok
}

// IsAudio is the exported flavor used by the web-UI browse endpoint.
func IsAudio(name string) bool { return isAudio(name) }

func contentType(name string) string {
	if ct, ok := audioExts[strings.ToLower(filepath.Ext(name))]; ok {
		return ct
	}
	return "application/octet-stream"
}

func suffix(name string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
}
