package server

import (
	"net/http"
	"sort"

	"kaze/internal/media"
)

// Stubs for endpoints that clients call during initial sync. We don't
// persist user state (playlists, stars, ratings, scrobbles), so reads
// return empty payloads and writes are no-ops that report success.

func (s *Server) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.Playlists = &Playlists{}
	})
}

func (s *Server) handleGetPlaylist(w http.ResponseWriter, r *http.Request) {
	// We have no playlists. Returning "not found" lets clients move on
	// instead of waiting on a phantom resource.
	writeSubsonicError(w, r, ErrCodeNotFound, "playlist not found")
}

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.Playlist = &Playlist{ID: "0", Name: "(unsupported)"}
	})
}

func (s *Server) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		username, _ = s.checkSubsonicAuth(r)
	}
	if username == "" {
		username = "kaze"
	}
	writeSubsonic(w, r, func(resp *Response) {
		resp.User = &User{
			Username:          username,
			ScrobblingEnabled: false,
			AdminRole:         true,
			SettingsRole:      true,
			DownloadRole:      true,
			StreamRole:        true,
			CoverArtRole:      true,
			PlaylistRole:      false,
			UploadRole:        false,
			CommentRole:       false,
			PodcastRole:       false,
			JukeboxRole:       false,
			ShareRole:         false,
		}
	})
}

func (s *Server) handleScrobble(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleStar(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleUnstar(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleSetRating(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleGetStarred(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.Starred = &Starred{}
	})
}

func (s *Server) handleGetStarred2(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.Starred2 = &Starred2{}
	})
}

func (s *Server) handleGetGenres(w http.ResponseWriter, r *http.Request) {
	counts := map[string]struct {
		songs  int
		albums map[string]bool
	}{}

	for _, al := range s.lib.Albums() {
		key := al.Genre
		if key == "" {
			continue
		}
		c := counts[key]
		if c.albums == nil {
			c.albums = map[string]bool{}
		}
		c.albums[al.ID] = true
		c.songs += len(al.TrackIDs)
		counts[key] = c
	}

	out := &Genres{}
	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		c := counts[n]
		out.Genre = append(out.Genre, Genre{
			Value: n, SongCount: c.songs, AlbumCount: len(c.albums),
		})
	}
	writeSubsonic(w, r, func(resp *Response) {
		resp.Genres = out
	})
}

func (s *Server) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	size := qInt(r, "size", 10)
	if size <= 0 {
		size = 10
	}
	all := s.lib.Albums()
	out := &Songs{}
	// Naive sample: iterate by hashed order, pick first N tracks.
	sort.Slice(all, func(i, j int) bool { return naturalHash(all[i].ID) < naturalHash(all[j].ID) })
	for _, al := range all {
		for _, tid := range al.TrackIDs {
			if t, ok := s.lib.Track(tid); ok {
				out.Song = append(out.Song, buildSong(al, t))
				if len(out.Song) >= size {
					break
				}
			}
		}
		if len(out.Song) >= size {
			break
		}
	}
	writeSubsonic(w, r, func(resp *Response) {
		resp.RandomSongs = out
	})
}

func (s *Server) handleGetSongsByGenre(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")
	size := qInt(r, "count", 10)
	if size <= 0 {
		size = 10
	}
	out := &Songs{}
	if genre != "" {
		for _, al := range s.lib.Albums() {
			if al.Genre != genre {
				continue
			}
			for _, tid := range al.TrackIDs {
				if t, ok := s.lib.Track(tid); ok {
					out.Song = append(out.Song, buildSong(al, t))
					if len(out.Song) >= size {
						break
					}
				}
			}
			if len(out.Song) >= size {
				break
			}
		}
	}
	writeSubsonic(w, r, func(resp *Response) {
		resp.SongsByGenre = out
	})
}

func (s *Server) handleGetNowPlaying(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.NowPlaying = &NowPlaying{}
	})
}

func (s *Server) handleGetTopSongs(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.TopSongs = &Songs{}
	})
}

func (s *Server) handleGetSimilarSongs(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.SimilarSongs = &Songs{}
	})
}

func (s *Server) handleGetSimilarSongs2(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.SimilarSongs2 = &Songs{}
	})
}

func (s *Server) handleGetArtistInfo(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.ArtistInfo = &ArtistInfo{}
	})
}

func (s *Server) handleGetArtistInfo2(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.ArtistInfo2 = &ArtistInfo{}
	})
}

func (s *Server) handleGetAlbumInfo(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.AlbumInfo = &AlbumInfo{}
	})
}

func (s *Server) handleGetAlbumInfo2(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		resp.AlbumInfo = &AlbumInfo{}
	})
}

// handleGetAvatar serves a 404 — we don't have user avatars.
func (s *Server) handleGetAvatar(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// --- Library scan endpoints (real, not stubs) ---

func (s *Server) handleStartScan(w http.ResponseWriter, r *http.Request) {
	go s.lib.Rescan()
	stats := s.lib.Stats()
	writeSubsonic(w, r, func(resp *Response) {
		resp.ScanStatus = &ScanStatus{
			Scanning: true,
			Count:    int64(stats.Tracks),
		}
	})
}

func (s *Server) handleGetScanStatus(w http.ResponseWriter, r *http.Request) {
	stats := s.lib.Stats()
	writeSubsonic(w, r, func(resp *Response) {
		resp.ScanStatus = &ScanStatus{
			Scanning: s.lib.IsScanning(),
			Count:    int64(stats.Tracks),
		}
	})
}

// --- Empty stubs for the remaining catalog endpoints clients may probe ---

func (s *Server) handleGetBookmarks(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) { resp.Bookmarks = &Bookmarks{} })
}

func (s *Server) handleCreateBookmark(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, nil)
}

func (s *Server) handleGetPodcasts(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) { resp.Podcasts = &Podcasts{} })
}

func (s *Server) handleGetNewestPodcasts(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) { resp.NewestPodcasts = &NewestPodcasts{} })
}

func (s *Server) handleGetInternetRadioStations(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) { resp.InternetRadioStations = &InternetRadioStations{} })
}

func (s *Server) handleGetShares(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) { resp.Shares = &Shares{} })
}

func (s *Server) handleGetVideos(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) { resp.Videos = &Videos{} })
}

func (s *Server) handleGetLyrics(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) { resp.Lyrics = &Lyrics{} })
}

func (s *Server) handleGetOpenSubsonicExtensions(w http.ResponseWriter, r *http.Request) {
	writeSubsonic(w, r, func(resp *Response) {
		// Empty list = we advertise OpenSubsonic via the envelope but
		// don't claim any extension implementations.
		resp.OpenSubsonicExtensions = []OpenSubsonicExtension{}
	})
}

// Silence unused-import lint if media isn't referenced; we still need it
// transitively via shared helpers in subsonic.go.
var _ = media.ArtistID