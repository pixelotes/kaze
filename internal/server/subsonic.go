package server

import (
	"net/http"
	"sort"
	"strings"
	"unicode"

	"kaze/internal/media"
)

// indexBucket returns the bucket name for an artist on the index pages
// ("A".."Z", or "#" for anything else).
func indexBucket(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "#"
	}
	r := []rune(name)[0]
	r = unicode.ToUpper(r)
	if r >= 'A' && r <= 'Z' {
		return string(r)
	}
	return "#"
}

// --- ping / getLicense ---

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.License = &License{Valid: true}
	})
}

// --- getMusicFolders ---

func (s *Server) handleGetMusicFolders(w http.ResponseWriter, r *http.Request) {
	folders := s.lib.MusicFolders()
	writeSubsonic(w, r, func(resp *Response) {
		mf := &MusicFolders{}
		for _, f := range folders {
			mf.MusicFolder = append(mf.MusicFolder, MusicFolder{ID: f.ID, Name: f.Name})
		}
		resp.MusicFolders = mf
	})
}

// --- getArtists (ID3) ---

func (s *Server) handleGetArtists(w http.ResponseWriter, r *http.Request) {
	all := s.lib.Artists()
	buckets := map[string][]ArtistID3{}
	for _, a := range all {
		entry := ArtistID3{
			ID:         a.ID,
			Name:       a.Name,
			AlbumCount: len(a.AlbumIDs),
		}
		if a.CoverArt != "" {
			entry.CoverArt = a.ID
		}
		key := indexBucket(a.Name)
		buckets[key] = append(buckets[key], entry)
	}

	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		// "#" goes last for nicer UX.
		if keys[i] == "#" {
			return false
		}
		if keys[j] == "#" {
			return true
		}
		return keys[i] < keys[j]
	})

	out := &ArtistsID3{IgnoredArticles: "The El La Los Las Le Les"}
	for _, k := range keys {
		out.Index = append(out.Index, IndexID3{Name: k, Artist: buckets[k]})
	}

	writeSubsonic(w, r, func(resp *Response) {
		resp.Artists = out
	})
}

// --- getIndexes (legacy, but Sonixd / other clients still ask) ---

func (s *Server) handleGetIndexes(w http.ResponseWriter, r *http.Request) {
	all := s.lib.Artists()
	buckets := map[string][]ArtistIndexed{}
	for _, a := range all {
		entry := ArtistIndexed{
			ID:         a.ID,
			Name:       a.Name,
			AlbumCount: len(a.AlbumIDs),
		}
		if a.CoverArt != "" {
			entry.CoverArt = a.ID
		}
		key := indexBucket(a.Name)
		buckets[key] = append(buckets[key], entry)
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == "#" {
			return false
		}
		if keys[j] == "#" {
			return true
		}
		return keys[i] < keys[j]
	})

	out := &Indexes{
		LastModified:    0,
		IgnoredArticles: "The El La Los Las Le Les",
	}
	for _, k := range keys {
		out.Index = append(out.Index, IndexNode{Name: k, Artist: buckets[k]})
	}

	writeSubsonic(w, r, func(resp *Response) {
		resp.Indexes = out
	})
}

// --- getArtist ---

func (s *Server) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeSubsonicError(w, r, ErrCodeMissingParam, "missing id")
		return
	}
	artist, ok := s.lib.Artist(id)
	if !ok {
		writeSubsonicError(w, r, ErrCodeNotFound, "artist not found")
		return
	}
	resp := ArtistWithAlbumsID3{
		ArtistID3: ArtistID3{
			ID:         artist.ID,
			Name:       artist.Name,
			AlbumCount: len(artist.AlbumIDs),
		},
	}
	if artist.CoverArt != "" {
		resp.CoverArt = artist.ID
	}
	for _, aid := range artist.AlbumIDs {
		al, ok := s.lib.Album(aid)
		if !ok {
			continue
		}
		resp.Album = append(resp.Album, buildAlbumID3(s.lib, al))
	}
	writeSubsonic(w, r, func(r *Response) { r.Artist = &resp })
}

// --- getAlbum ---

func (s *Server) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeSubsonicError(w, r, ErrCodeMissingParam, "missing id")
		return
	}
	album, ok := s.lib.Album(id)
	if !ok {
		writeSubsonicError(w, r, ErrCodeNotFound, "album not found")
		return
	}
	out := AlbumWithSongsID3{AlbumID3: buildAlbumID3(s.lib, album)}
	for _, tid := range album.TrackIDs {
		t, ok := s.lib.Track(tid)
		if !ok {
			continue
		}
		out.Song = append(out.Song, buildSong(album, t))
	}
	writeSubsonic(w, r, func(r *Response) { r.Album = &out })
}

// --- getAlbumList2 ---

func (s *Server) handleGetAlbumList2(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	size := qInt(r, "size", 10)
	offset := qInt(r, "offset", 0)
	if size <= 0 {
		size = 10
	}

	all := s.lib.Albums()

	switch typ {
	case "alphabeticalByName":
		sort.Slice(all, func(i, j int) bool { return strings.ToLower(all[i].Name) < strings.ToLower(all[j].Name) })
	case "alphabeticalByArtist":
		sort.Slice(all, func(i, j int) bool {
			if all[i].Artist != all[j].Artist {
				return strings.ToLower(all[i].Artist) < strings.ToLower(all[j].Artist)
			}
			return strings.ToLower(all[i].Name) < strings.ToLower(all[j].Name)
		})
	case "byYear":
		from := qInt(r, "fromYear", 0)
		to := qInt(r, "toYear", 0)
		filtered := all[:0]
		for _, a := range all {
			if from > 0 && a.Year < from {
				continue
			}
			if to > 0 && a.Year > to {
				continue
			}
			filtered = append(filtered, a)
		}
		all = filtered
		sort.Slice(all, func(i, j int) bool { return all[i].Year < all[j].Year })
	case "newest", "recent":
		// Without persisted timestamps, "newest" devolves to alphabetic.
		// Most clients will accept this gracefully.
		sort.Slice(all, func(i, j int) bool { return all[i].Name > all[j].Name })
	case "random":
		// Deterministic shuffle via name hash — good enough without rand state.
		sort.Slice(all, func(i, j int) bool { return naturalHash(all[i].ID) < naturalHash(all[j].ID) })
	default:
		sort.Slice(all, func(i, j int) bool {
			if all[i].Artist != all[j].Artist {
				return strings.ToLower(all[i].Artist) < strings.ToLower(all[j].Artist)
			}
			return strings.ToLower(all[i].Name) < strings.ToLower(all[j].Name)
		})
	}

	if offset > len(all) {
		offset = len(all)
	}
	end := offset + size
	if end > len(all) {
		end = len(all)
	}
	page := all[offset:end]

	out := &AlbumList2{}
	for _, a := range page {
		out.Album = append(out.Album, buildAlbumID3(s.lib, a))
	}
	writeSubsonic(w, r, func(r *Response) { r.AlbumList2 = out })
}

// --- search3 ---

func (s *Server) handleSearch3(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("query")
	artistsLimit := qInt(r, "artistCount", 20)
	albumsLimit := qInt(r, "albumCount", 20)
	songsLimit := qInt(r, "songCount", 20)

	artists, albums, tracks := s.lib.Search(q, artistsLimit, albumsLimit, songsLimit)

	out := &SearchResult3{}
	for _, a := range artists {
		entry := ArtistID3{
			ID:         a.ID,
			Name:       a.Name,
			AlbumCount: len(a.AlbumIDs),
		}
		if a.CoverArt != "" {
			entry.CoverArt = a.ID
		}
		out.Artist = append(out.Artist, entry)
	}
	for _, al := range albums {
		out.Album = append(out.Album, buildAlbumID3(s.lib, al))
	}
	for _, t := range tracks {
		al, _ := s.lib.Album(t.AlbumID)
		out.Song = append(out.Song, buildSong(al, t))
	}
	writeSubsonic(w, r, func(r *Response) { r.SearchResult3 = out })
}

// --- getMusicDirectory (legacy browse) ---

func (s *Server) handleGetMusicDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeSubsonicError(w, r, ErrCodeMissingParam, "missing id")
		return
	}
	switch media.EntityKind(id) {
	case "artist":
		artist, ok := s.lib.Artist(id)
		if !ok {
			writeSubsonicError(w, r, ErrCodeNotFound, "artist not found")
			return
		}
		dir := &Directory{ID: artist.ID, Name: artist.Name}
		for _, aid := range artist.AlbumIDs {
			al, ok := s.lib.Album(aid)
			if !ok {
				continue
			}
			child := Child{
				ID:     al.ID,
				Parent: artist.ID,
				IsDir:  true,
				Title:  al.Name,
				Album:  al.Name,
				Artist: al.Artist,
				Year:   al.Year,
				Genre:  al.Genre,
			}
			if al.CoverArt != "" {
				child.CoverArt = al.ID
			}
			dir.Child = append(dir.Child, child)
		}
		writeSubsonic(w, r, func(r *Response) { r.Directory = dir })
	case "album":
		al, ok := s.lib.Album(id)
		if !ok {
			writeSubsonicError(w, r, ErrCodeNotFound, "album not found")
			return
		}
		dir := &Directory{ID: al.ID, Parent: al.ArtistID, Name: al.Name}
		for _, tid := range al.TrackIDs {
			t, ok := s.lib.Track(tid)
			if !ok {
				continue
			}
			dir.Child = append(dir.Child, buildSong(al, t))
		}
		writeSubsonic(w, r, func(r *Response) { r.Directory = dir })
	default:
		writeSubsonicError(w, r, ErrCodeNotFound, "unknown id")
	}
}

// --- getCoverArt ---

func (s *Server) handleGetCoverArt(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeSubsonicError(w, r, ErrCodeMissingParam, "missing id")
		return
	}

	// Allow song-IDs by walking up to their album.
	if media.EntityKind(id) == "track" {
		if t, ok := s.lib.Track(id); ok {
			id = t.AlbumID
		}
	}

	var coverPath string
	switch media.EntityKind(id) {
	case "album":
		if al, ok := s.lib.Album(id); ok {
			coverPath = al.CoverArt
		}
	case "artist":
		if a, ok := s.lib.Artist(id); ok {
			coverPath = a.CoverArt
		}
	}

	if coverPath == "" {
		// Subsonic spec wants a binary response, so we 404 instead of a JSON error.
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Content-Type", media.CoverContentType(coverPath))
	http.ServeFile(w, r, coverPath)
}

// --- stream / download ---

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeSubsonicError(w, r, ErrCodeMissingParam, "missing id")
		return
	}
	t, ok := s.lib.Track(id)
	if !ok {
		writeSubsonicError(w, r, ErrCodeNotFound, "song not found")
		return
	}
	w.Header().Set("Content-Type", t.ContentType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeFile(w, r, t.Path)
}

// --- Builders ---

func buildAlbumID3(lib *media.Library, al *media.Album) AlbumID3 {
	out := AlbumID3{
		ID:        al.ID,
		Name:      al.Name,
		Artist:    al.Artist,
		ArtistID:  al.ArtistID,
		SongCount: len(al.TrackIDs),
		Year:      al.Year,
		Genre:     al.Genre,
	}
	if al.CoverArt != "" {
		out.CoverArt = al.ID
	}
	for _, tid := range al.TrackIDs {
		if t, ok := lib.Track(tid); ok {
			out.Duration += t.Duration
		}
	}
	return out
}

func buildSong(al *media.Album, t *media.Track) Child {
	c := Child{
		ID:          t.ID,
		IsDir:       false,
		Title:       t.Title,
		Album:       t.Album,
		Artist:      t.Artist,
		Track:       t.TrackNo,
		Year:        t.Year,
		Genre:       t.Genre,
		Size:        t.Size,
		ContentType: t.ContentType,
		Suffix:      t.Suffix,
		Duration:    t.Duration,
		BitRate:     t.BitRate,
		IsVideo:     false,
		DiscNumber:  t.DiscNo,
		AlbumID:     t.AlbumID,
		ArtistID:    t.ArtistID,
		Type:        "music",
	}
	if al != nil {
		c.Parent = al.ID
		if al.CoverArt != "" {
			c.CoverArt = al.ID
		}
	}
	return c
}

// naturalHash gives a stable-ish ordering key for "random" listing without
// needing to seed an RNG per request.
func naturalHash(s string) uint64 {
	var h uint64 = 1469598103934665603
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
