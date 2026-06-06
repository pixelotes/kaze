package media

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
)

// trackTags is the subset of metadata we care about. AlbumArtist is the
// preferred grouping key when present (it's what makes compilations and
// "Various Artists" work without putting every track under a different
// "artist").
type trackTags struct {
	Title       string
	Artist      string
	AlbumArtist string
	Album       string
	Year        int
	Genre       string
	TrackNo     int
	DiscNo      int
}

// readTags returns best-effort metadata from an audio file. Any field
// that can't be determined is left at its zero value; the caller is
// expected to fall back to filesystem-derived values.
func readTags(path string) trackTags {
	out := trackTags{}

	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return out
	}

	out.Title = strings.TrimSpace(m.Title())
	out.Artist = strings.TrimSpace(m.Artist())
	out.AlbumArtist = strings.TrimSpace(m.AlbumArtist())
	out.Album = strings.TrimSpace(m.Album())
	out.Year = m.Year()
	out.Genre = strings.TrimSpace(m.Genre())
	if tn, _ := m.Track(); tn > 0 {
		out.TrackNo = tn
	}
	if dn, _ := m.Disc(); dn > 0 {
		out.DiscNo = dn
	}
	return out
}

// filenameTrackNo extracts a leading track number from a filename like
// "03 - Title.flac" or "03. Title.mp3". Returns 0 if none is present.
var leadingNumberRe = regexp.MustCompile(`^\s*(\d{1,3})\b`)

func filenameTrackNo(name string) int {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if m := leadingNumberRe.FindStringSubmatch(base); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// titleFromFilename strips leading "03 - " / "03. " / "03 " patterns
// from a filename and returns a presentable title.
var leadingNumberPrefixRe = regexp.MustCompile(`^\s*\d{1,3}\s*[-.\s]\s*`)

func titleFromFilename(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return strings.TrimSpace(leadingNumberPrefixRe.ReplaceAllString(base, ""))
}
