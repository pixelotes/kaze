package media

import (
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Library is the in-memory index built by Scan. Lookup-only operations
// take a read lock; the scan path holds the write lock.
type Library struct {
	mu sync.RWMutex

	// Display order of libraries as configured.
	musicFolders []MusicFolder

	artists map[string]*Artist
	albums  map[string]*Album
	tracks  map[string]*Track

	// Lookup helpers for search / browsing.
	artistNames map[string]string // lowercased name -> ID (first writer wins)

	// scanning is non-zero while a rescan is in progress.
	scanning atomic.Bool
}

type MusicFolder struct {
	ID   int
	Name string
	Path string
}

type Stats struct {
	Artists int
	Albums  int
	Tracks  int
}

func NewLibrary() *Library {
	return &Library{
		artists:     map[string]*Artist{},
		albums:      map[string]*Album{},
		tracks:      map[string]*Track{},
		artistNames: map[string]string{},
	}
}

// Rescan clears the current index and re-walks every previously
// configured library root. Safe to call concurrently with reads — the
// swap happens under the write lock. The boolean Scanning flag flips
// for the duration so scanStatus requests reflect reality.
func (l *Library) Rescan() {
	if !l.scanning.CompareAndSwap(false, true) {
		// A scan is already in progress; coalesce.
		return
	}
	defer l.scanning.Store(false)

	l.mu.Lock()
	folders := append([]MusicFolder(nil), l.musicFolders...)
	l.artists = map[string]*Artist{}
	l.albums = map[string]*Album{}
	l.tracks = map[string]*Track{}
	l.artistNames = map[string]string{}
	l.musicFolders = nil
	l.mu.Unlock()

	for _, f := range folders {
		if err := l.Scan(f.Name, f.Path); err != nil {
			log.Printf("rescan: %s: %v", f.Name, err)
		}
	}
	s := l.Stats()
	log.Printf("rescan complete: %d artists / %d albums / %d tracks", s.Artists, s.Albums, s.Tracks)
}

// IsScanning returns true if a rescan is currently in flight.
func (l *Library) IsScanning() bool {
	return l.scanning.Load()
}

func (l *Library) Stats() Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return Stats{Artists: len(l.artists), Albums: len(l.albums), Tracks: len(l.tracks)}
}

func (l *Library) MusicFolders() []MusicFolder {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]MusicFolder, len(l.musicFolders))
	copy(out, l.musicFolders)
	return out
}

func (l *Library) Artist(id string) (*Artist, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	a, ok := l.artists[id]
	return a, ok
}

func (l *Library) Album(id string) (*Album, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	a, ok := l.albums[id]
	return a, ok
}

func (l *Library) Track(id string) (*Track, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.tracks[id]
	return t, ok
}

// Artists returns all artists in stable name order.
func (l *Library) Artists() []*Artist {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]*Artist, 0, len(l.artists))
	for _, a := range l.artists {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		return naturalLess(out[i].Name, out[j].Name)
	})
	return out
}

// Albums returns all albums in stable name order.
func (l *Library) Albums() []*Album {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]*Album, 0, len(l.albums))
	for _, a := range l.albums {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Artist != out[j].Artist {
			return naturalLess(out[i].Artist, out[j].Artist)
		}
		if out[i].Year != out[j].Year && out[i].Year > 0 && out[j].Year > 0 {
			return out[i].Year < out[j].Year
		}
		return naturalLess(out[i].Name, out[j].Name)
	})
	return out
}

// TracksOfAlbum returns ordered tracks for an album.
func (l *Library) TracksOfAlbum(albumID string) []*Track {
	l.mu.RLock()
	defer l.mu.RUnlock()
	album, ok := l.albums[albumID]
	if !ok {
		return nil
	}
	out := make([]*Track, 0, len(album.TrackIDs))
	for _, tid := range album.TrackIDs {
		if t, ok := l.tracks[tid]; ok {
			out = append(out, t)
		}
	}
	return out
}

// Search returns up to limit artists/albums/songs matching q (substring,
// case-insensitive). Used by search3.
func (l *Library) Search(q string, artistsLimit, albumsLimit, songsLimit int) (artists []*Artist, albums []*Album, tracks []*Track) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		return nil, nil, nil
	}

	for _, a := range l.artists {
		if strings.Contains(strings.ToLower(a.Name), needle) {
			artists = append(artists, a)
		}
	}
	sort.Slice(artists, func(i, j int) bool { return naturalLess(artists[i].Name, artists[j].Name) })
	if len(artists) > artistsLimit && artistsLimit > 0 {
		artists = artists[:artistsLimit]
	}

	for _, al := range l.albums {
		if strings.Contains(strings.ToLower(al.Name), needle) ||
			strings.Contains(strings.ToLower(al.Artist), needle) {
			albums = append(albums, al)
		}
	}
	sort.Slice(albums, func(i, j int) bool { return naturalLess(albums[i].Name, albums[j].Name) })
	if len(albums) > albumsLimit && albumsLimit > 0 {
		albums = albums[:albumsLimit]
	}

	for _, t := range l.tracks {
		if strings.Contains(strings.ToLower(t.Title), needle) {
			tracks = append(tracks, t)
		}
	}
	sort.Slice(tracks, func(i, j int) bool { return naturalLess(tracks[i].Title, tracks[j].Title) })
	if len(tracks) > songsLimit && songsLimit > 0 {
		tracks = tracks[:songsLimit]
	}

	return
}

// naturalLess compares strings with digit runs treated as numbers, so
// "Track 2" < "Track 10".
func naturalLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	ia, ib := 0, 0
	for ia < len(la) && ib < len(lb) {
		ca, cb := la[ia], lb[ib]
		if ca >= '0' && ca <= '9' && cb >= '0' && cb <= '9' {
			na, nb := 0, 0
			for ia < len(la) && la[ia] >= '0' && la[ia] <= '9' {
				na = na*10 + int(la[ia]-'0')
				ia++
			}
			for ib < len(lb) && lb[ib] >= '0' && lb[ib] <= '9' {
				nb = nb*10 + int(lb[ib]-'0')
				ib++
			}
			if na != nb {
				return na < nb
			}
		} else {
			if ca != cb {
				return ca < cb
			}
			ia++
			ib++
		}
	}
	return len(la) < len(lb)
}
