package server

import (
	"fmt"
	"log"
	"net/http"

	"kaze/internal/config"
	"kaze/internal/media"

	"github.com/gorilla/mux"
)

type Server struct {
	cfg    *config.Config
	lib    *media.Library
	router *mux.Router
}

func New(cfg *config.Config, lib *media.Library) *Server {
	s := &Server{cfg: cfg, lib: lib, router: mux.NewRouter()}
	s.routes()
	return s
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.App.Port)
	log.Printf("kaze listening on %s", addr)
	return http.ListenAndServe(addr, s.logRequests(s.router))
}

func (s *Server) routes() {
	// Static + landing.
	s.router.PathPrefix("/js/").Handler(http.StripPrefix("/js/", http.FileServer(http.Dir("web/js"))))
	s.router.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("web/css"))))
	s.router.PathPrefix("/vendor/").Handler(http.StripPrefix("/vendor/", http.FileServer(http.Dir("web/vendor"))))
	s.router.HandleFunc("/", s.serveIndex).Methods("GET")

	// Web UI auth + browse API (JWT-authenticated).
	s.router.HandleFunc("/api/v1/login", s.handleLogin).Methods("POST")
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.jwtMiddleware)
	api.HandleFunc("/browse", s.handleBrowse).Methods("GET")
	api.HandleFunc("/image", s.handleImage).Methods("GET")
	api.HandleFunc("/stream", s.handleStreamWeb).Methods("GET")
	api.HandleFunc("/download", s.handleDownload).Methods("GET")

	// Subsonic REST: both .view and bare forms are accepted by spec.
	rest := s.router.PathPrefix("/rest").Subrouter()
	registerSubsonic(rest, s)
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

// registerSubsonic wires each Subsonic endpoint under both /rest/foo and
// /rest/foo.view (the historical extension). Auth wraps every endpoint
// except ping, which clients sometimes call before auth to probe.
func registerSubsonic(r *mux.Router, s *Server) {
	endpoints := []struct {
		name    string
		handler http.HandlerFunc
		noAuth  bool
	}{
		{"ping", s.handlePing, false},
		{"getLicense", s.handleGetLicense, false},
		{"getMusicFolders", s.handleGetMusicFolders, false},
		{"getIndexes", s.handleGetIndexes, false},
		{"getArtists", s.handleGetArtists, false},
		{"getArtist", s.handleGetArtist, false},
		{"getAlbum", s.handleGetAlbum, false},
		{"getAlbumList2", s.handleGetAlbumList2, false},
		{"search3", s.handleSearch3, false},
		{"getMusicDirectory", s.handleGetMusicDirectory, false},
		{"getCoverArt", s.handleGetCoverArt, false},
		{"stream", s.handleStream, false},
		{"download", s.handleStream, false},

		// Stubs / read-only conveniences.
		{"getPlaylists", s.handleGetPlaylists, false},
		{"getPlaylist", s.handleGetPlaylist, false},
		{"createPlaylist", s.handleCreatePlaylist, false},
		{"updatePlaylist", s.handleUpdatePlaylist, false},
		{"deletePlaylist", s.handleDeletePlaylist, false},
		{"getUser", s.handleGetUser, false},
		{"scrobble", s.handleScrobble, false},
		{"star", s.handleStar, false},
		{"unstar", s.handleUnstar, false},
		{"setRating", s.handleSetRating, false},
		{"getStarred", s.handleGetStarred, false},
		{"getStarred2", s.handleGetStarred2, false},
		{"getGenres", s.handleGetGenres, false},
		{"getRandomSongs", s.handleGetRandomSongs, false},
		{"getSongsByGenre", s.handleGetSongsByGenre, false},
		{"getNowPlaying", s.handleGetNowPlaying, false},
		{"getTopSongs", s.handleGetTopSongs, false},
		{"getSimilarSongs", s.handleGetSimilarSongs, false},
		{"getSimilarSongs2", s.handleGetSimilarSongs2, false},
		{"getArtistInfo", s.handleGetArtistInfo, false},
		{"getArtistInfo2", s.handleGetArtistInfo2, false},
		{"getAlbumInfo", s.handleGetAlbumInfo, false},
		{"getAlbumInfo2", s.handleGetAlbumInfo2, false},
		{"getAvatar", s.handleGetAvatar, false},
		{"startScan", s.handleStartScan, false},
		{"scanLibrary", s.handleStartScan, false}, // alias used by Submariner and others
		{"getScanStatus", s.handleGetScanStatus, false},
		{"getBookmarks", s.handleGetBookmarks, false},
		{"createBookmark", s.handleCreateBookmark, false},
		{"deleteBookmark", s.handleDeleteBookmark, false},
		{"getPodcasts", s.handleGetPodcasts, false},
		{"getNewestPodcasts", s.handleGetNewestPodcasts, false},
		{"getInternetRadioStations", s.handleGetInternetRadioStations, false},
		{"getShares", s.handleGetShares, false},
		{"getVideos", s.handleGetVideos, false},
		{"getLyrics", s.handleGetLyrics, false},
		{"getLyricsBySongId", s.handleGetLyrics, false},
		{"getOpenSubsonicExtensions", s.handleGetOpenSubsonicExtensions, false},
	}
	for _, e := range endpoints {
		h := http.Handler(e.handler)
		if !e.noAuth {
			h = s.subsonicAuthMiddleware(h)
		}
		r.Handle("/"+e.name, h)
		r.Handle("/"+e.name+".view", h)
	}
}

// logRequests is a tiny access log so it's obvious what clients ask for.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s%s", r.Method, r.URL.Path, queryForLog(r))
		next.ServeHTTP(w, r)
	})
}

func queryForLog(r *http.Request) string {
	q := r.URL.Query()
	// Strip sensitive bits from access log.
	q.Del("p")
	q.Del("t")
	q.Del("s")
	q.Del("token")
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}
