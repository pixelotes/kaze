package media

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scan walks libRoot recursively and indexes every audio file it finds.
// Artist / album grouping is driven by tags (albumartist > artist > "Unknown
// Artist"; album tag > parent directory name). The directory layout under
// libRoot is irrelevant for the Subsonic-facing entity model — the web UI's
// /api/v1/browse endpoint is what walks the literal filesystem.
//
// Cover art is resolved by looking in the directory of the first track of
// each album and walking up to libRoot, picking the first cover-style image
// found. This handles both "cover.jpg next to the tracks" and "cover.jpg at
// the album's parent dir for multi-disc layouts".
func (l *Library) Scan(friendlyName, libRoot string) error {
	abs, err := filepath.Abs(libRoot)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("library %q: %w", friendlyName, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("library %q is not a directory: %s", friendlyName, abs)
	}

	l.mu.Lock()
	folderID := len(l.musicFolders) + 1
	l.musicFolders = append(l.musicFolders, MusicFolder{
		ID: folderID, Name: friendlyName, Path: abs,
	})
	l.mu.Unlock()

	// Phase 1 — discover every audio file and read its tags.
	type scanned struct {
		path string
		size int64
		name string
		tags trackTags
	}
	var files []scanned
	walkErr := filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("scan: walk %s: %v", path, err)
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") && path != abs {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !isAudio(name) {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			log.Printf("scan: stat %s: %v", path, err)
			return nil
		}
		files = append(files, scanned{
			path: path,
			size: fi.Size(),
			name: name,
			tags: readTags(path),
		})
		return nil
	})
	if walkErr != nil {
		return walkErr
	}

	// Phase 2 — group by (artist, album). Keys are lowercased so that minor
	// case differences across tags don't fragment albums.
	type albumKey struct{ artist, album string }
	type albumBuild struct {
		artistDisplay string
		albumDisplay  string
		year          int
		genre         string
		firstTrackDir string
		trackPaths    []string
	}
	type artistBuild struct {
		display    string
		albumOrder []albumKey
		seenAlbums map[albumKey]bool
	}

	artists := map[string]*artistBuild{}
	albums := map[albumKey]*albumBuild{}
	tracks := map[string]*Track{} // by path

	for _, f := range files {
		artistDisplay := strings.TrimSpace(f.tags.AlbumArtist)
		if artistDisplay == "" {
			artistDisplay = strings.TrimSpace(f.tags.Artist)
		}
		if artistDisplay == "" {
			artistDisplay = "Unknown Artist"
		}

		albumDisplay := strings.TrimSpace(f.tags.Album)
		if albumDisplay == "" {
			albumDisplay = filepath.Base(filepath.Dir(f.path))
		}

		ak := strings.ToLower(artistDisplay)
		bk := albumKey{artist: ak, album: strings.ToLower(albumDisplay)}

		title := strings.TrimSpace(f.tags.Title)
		if title == "" {
			title = titleFromFilename(f.name)
		}
		trackNo := f.tags.TrackNo
		if trackNo == 0 {
			trackNo = filenameTrackNo(f.name)
		}
		discNo := f.tags.DiscNo
		if discNo == 0 {
			discNo = 1
		}

		artistID := ArtistID(ak)
		albumID := AlbumID(ak + "\x00" + strings.ToLower(albumDisplay))
		trackID := TrackID(f.path)

		tracks[f.path] = &Track{
			ID:          trackID,
			Path:        f.path,
			Title:       title,
			TrackNo:     trackNo,
			DiscNo:      discNo,
			Size:        f.size,
			Suffix:      suffix(f.name),
			ContentType: contentType(f.name),
			AlbumID:     albumID,
			ArtistID:    artistID,
			Album:       albumDisplay,
			Artist:      artistDisplay,
			Year:        f.tags.Year,
			Genre:       f.tags.Genre,
		}

		ab, ok := albums[bk]
		if !ok {
			ab = &albumBuild{
				artistDisplay: artistDisplay,
				albumDisplay:  albumDisplay,
				firstTrackDir: filepath.Dir(f.path),
			}
			albums[bk] = ab
		}
		ab.trackPaths = append(ab.trackPaths, f.path)
		if f.tags.Year > 0 && ab.year == 0 {
			ab.year = f.tags.Year
		}
		if f.tags.Genre != "" && ab.genre == "" {
			ab.genre = f.tags.Genre
		}

		ar, ok := artists[ak]
		if !ok {
			ar = &artistBuild{
				display:    artistDisplay,
				seenAlbums: map[albumKey]bool{},
			}
			artists[ak] = ar
		}
		if !ar.seenAlbums[bk] {
			ar.seenAlbums[bk] = true
			ar.albumOrder = append(ar.albumOrder, bk)
		}
	}

	// Phase 3 — sort tracks within each album.
	for _, ab := range albums {
		sort.Slice(ab.trackPaths, func(i, j int) bool {
			ti, tj := tracks[ab.trackPaths[i]], tracks[ab.trackPaths[j]]
			if ti.DiscNo != tj.DiscNo {
				return ti.DiscNo < tj.DiscNo
			}
			if ti.TrackNo != tj.TrackNo {
				return ti.TrackNo < tj.TrackNo
			}
			return naturalLess(ti.Title, tj.Title)
		})
	}

	// Phase 4 — commit to library.
	l.mu.Lock()
	defer l.mu.Unlock()

	// Albums.
	for bk, ab := range albums {
		albumID := AlbumID(bk.artist + "\x00" + bk.album)
		album := &Album{
			ID:       albumID,
			Path:     ab.firstTrackDir,
			Name:     ab.albumDisplay,
			Artist:   ab.artistDisplay,
			ArtistID: ArtistID(bk.artist),
			Year:     ab.year,
			Genre:    ab.genre,
			Library:  friendlyName,
			CoverArt: findCoverWalkingUp(ab.firstTrackDir, abs),
		}
		album.TrackIDs = make([]string, len(ab.trackPaths))
		for i, p := range ab.trackPaths {
			album.TrackIDs[i] = TrackID(p)
		}
		l.albums[albumID] = album
	}

	// Artists.
	for ak, ar := range artists {
		artistID := ArtistID(ak)

		// Order albums for this artist by year then name.
		sort.Slice(ar.albumOrder, func(i, j int) bool {
			ai, aj := albums[ar.albumOrder[i]], albums[ar.albumOrder[j]]
			if ai.year > 0 && aj.year > 0 && ai.year != aj.year {
				return ai.year < aj.year
			}
			return naturalLess(ai.albumDisplay, aj.albumDisplay)
		})
		albumIDs := make([]string, 0, len(ar.albumOrder))
		for _, bk := range ar.albumOrder {
			albumIDs = append(albumIDs, AlbumID(bk.artist+"\x00"+bk.album))
		}

		// Cover: try walking up from any album dir. Usually yields nothing
		// for artists in flat layouts; not a problem, clients don't require it.
		var cover string
		if len(ar.albumOrder) > 0 {
			cover = findCoverWalkingUp(albums[ar.albumOrder[0]].firstTrackDir, abs)
		}

		if existing, ok := l.artists[artistID]; ok {
			seen := map[string]bool{}
			for _, id := range existing.AlbumIDs {
				seen[id] = true
			}
			for _, id := range albumIDs {
				if !seen[id] {
					existing.AlbumIDs = append(existing.AlbumIDs, id)
				}
			}
			continue
		}
		l.artists[artistID] = &Artist{
			ID:       artistID,
			Name:     ar.display,
			CoverArt: cover,
			Library:  friendlyName,
			AlbumIDs: albumIDs,
		}
		if _, ok := l.artistNames[ak]; !ok {
			l.artistNames[ak] = artistID
		}
	}

	// Tracks.
	for _, t := range tracks {
		l.tracks[t.ID] = t
	}

	return nil
}

// findCoverWalkingUp searches for a cover image starting at startDir and
// walking up the directory tree, stopping at libraryRoot (inclusive). Returns
// "" if nothing found.
func findCoverWalkingUp(startDir, libraryRoot string) string {
	libraryRoot = filepath.Clean(libraryRoot)
	dir := filepath.Clean(startDir)
	for {
		if cover := FindCover(dir); cover != "" {
			return cover
		}
		if dir == libraryRoot {
			return ""
		}
		parent := filepath.Dir(dir)
		// Defence in depth: never escape libraryRoot.
		if parent == dir || !strings.HasPrefix(parent, libraryRoot) {
			return ""
		}
		dir = parent
	}
}
