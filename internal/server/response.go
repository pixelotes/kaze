package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

// Subsonic protocol level we advertise. Clients usually accept anything
// that's <= their own minimum requirement; 1.16.1 is a safe modern target.
const subsonicAPIVersion = "1.16.1"
const serverName = "kaze"
const serverVersion = "0.1.0"

// Error codes per Subsonic spec.
const (
	ErrCodeGeneric         = 0
	ErrCodeMissingParam    = 10
	ErrCodeClientOutdated  = 20
	ErrCodeServerOutdated  = 30
	ErrCodeWrongAuth       = 40
	ErrCodeTokenNotAllowed = 41
	ErrCodeUnauthorized    = 50
	ErrCodeTrialOver       = 60
	ErrCodeNotFound        = 70
)

// Response is the Subsonic envelope. Exactly one payload pointer is
// expected to be non-nil per request; the rest are pruned via omitempty.
type Response struct {
	XMLName xml.Name `xml:"http://subsonic.org/restapi subsonic-response" json:"-"`

	Status        string `xml:"status,attr" json:"status"`
	Version       string `xml:"version,attr" json:"version"`
	Type          string `xml:"type,attr" json:"type"`
	ServerVersion string `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool   `xml:"openSubsonic,attr" json:"openSubsonic"`

	Error *ErrorPayload `xml:"error,omitempty" json:"error,omitempty"`

	MusicFolders  *MusicFolders         `xml:"musicFolders,omitempty" json:"musicFolders,omitempty"`
	Indexes       *Indexes              `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Artists       *ArtistsID3           `xml:"artists,omitempty" json:"artists,omitempty"`
	Artist        *ArtistWithAlbumsID3  `xml:"artist,omitempty" json:"artist,omitempty"`
	Album         *AlbumWithSongsID3    `xml:"album,omitempty" json:"album,omitempty"`
	AlbumList2    *AlbumList2           `xml:"albumList2,omitempty" json:"albumList2,omitempty"`
	SearchResult3 *SearchResult3        `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
	Directory     *Directory            `xml:"directory,omitempty" json:"directory,omitempty"`
	License       *License              `xml:"license,omitempty" json:"license,omitempty"`

	// Read-only stubs that many clients call during initial sync.
	Playlists    *Playlists    `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist     *Playlist     `xml:"playlist,omitempty" json:"playlist,omitempty"`
	User         *User         `xml:"user,omitempty" json:"user,omitempty"`
	Starred      *Starred      `xml:"starred,omitempty" json:"starred,omitempty"`
	Starred2     *Starred2     `xml:"starred2,omitempty" json:"starred2,omitempty"`
	Genres       *Genres       `xml:"genres,omitempty" json:"genres,omitempty"`
	RandomSongs  *Songs        `xml:"randomSongs,omitempty" json:"randomSongs,omitempty"`
	SongsByGenre *Songs        `xml:"songsByGenre,omitempty" json:"songsByGenre,omitempty"`
	NowPlaying   *NowPlaying   `xml:"nowPlaying,omitempty" json:"nowPlaying,omitempty"`
	TopSongs     *Songs        `xml:"topSongs,omitempty" json:"topSongs,omitempty"`
	SimilarSongs *Songs        `xml:"similarSongs,omitempty" json:"similarSongs,omitempty"`
	SimilarSongs2 *Songs       `xml:"similarSongs2,omitempty" json:"similarSongs2,omitempty"`
	ArtistInfo   *ArtistInfo   `xml:"artistInfo,omitempty" json:"artistInfo,omitempty"`
	ArtistInfo2  *ArtistInfo   `xml:"artistInfo2,omitempty" json:"artistInfo2,omitempty"`
	AlbumInfo    *AlbumInfo    `xml:"albumInfo,omitempty" json:"albumInfo,omitempty"`
	ScanStatus   *ScanStatus   `xml:"scanStatus,omitempty" json:"scanStatus,omitempty"`
	Bookmarks    *Bookmarks    `xml:"bookmarks,omitempty" json:"bookmarks,omitempty"`
	Podcasts     *Podcasts     `xml:"podcasts,omitempty" json:"podcasts,omitempty"`
	NewestPodcasts *NewestPodcasts `xml:"newestPodcasts,omitempty" json:"newestPodcasts,omitempty"`
	InternetRadioStations *InternetRadioStations `xml:"internetRadioStations,omitempty" json:"internetRadioStations,omitempty"`
	Shares       *Shares       `xml:"shares,omitempty" json:"shares,omitempty"`
	Videos       *Videos       `xml:"videos,omitempty" json:"videos,omitempty"`
	Lyrics       *Lyrics       `xml:"lyrics,omitempty" json:"lyrics,omitempty"`
	OpenSubsonicExtensions []OpenSubsonicExtension `xml:"openSubsonicExtensions,omitempty" json:"openSubsonicExtensions,omitempty"`
}

type ErrorPayload struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

type License struct {
	Valid bool   `xml:"valid,attr" json:"valid"`
	Email string `xml:"email,attr,omitempty" json:"email,omitempty"`
}

type MusicFolders struct {
	MusicFolder []MusicFolder `xml:"musicFolder" json:"musicFolder"`
}

type MusicFolder struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

// --- Legacy "indexes" model (alphabetical artist listing) ---

type Indexes struct {
	LastModified    int64       `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string      `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Index           []IndexNode `xml:"index" json:"index"`
}

type IndexNode struct {
	Name   string          `xml:"name,attr" json:"name"`
	Artist []ArtistIndexed `xml:"artist" json:"artist"`
}

type ArtistIndexed struct {
	ID         string `xml:"id,attr" json:"id"`
	Name       string `xml:"name,attr" json:"name"`
	CoverArt   string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount int    `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
}

// --- ID3 artists model ---

type ArtistsID3 struct {
	IgnoredArticles string      `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Index           []IndexID3  `xml:"index" json:"index"`
}

type IndexID3 struct {
	Name   string      `xml:"name,attr" json:"name"`
	Artist []ArtistID3 `xml:"artist" json:"artist"`
}

type ArtistID3 struct {
	ID         string `xml:"id,attr" json:"id"`
	Name       string `xml:"name,attr" json:"name"`
	CoverArt   string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
}

type ArtistWithAlbumsID3 struct {
	ArtistID3
	Album []AlbumID3 `xml:"album" json:"album"`
}

// --- ID3 albums / songs ---

type AlbumID3 struct {
	ID         string `xml:"id,attr" json:"id"`
	Name       string `xml:"name,attr" json:"name"`
	Artist     string `xml:"artist,attr" json:"artist"`
	ArtistID   string `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	CoverArt   string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	Duration   int    `xml:"duration,attr" json:"duration"`
	Year       int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre      string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	Created    string `xml:"created,attr,omitempty" json:"created,omitempty"`
}

type AlbumWithSongsID3 struct {
	AlbumID3
	Song []Child `xml:"song" json:"song"`
}

type AlbumList2 struct {
	Album []AlbumID3 `xml:"album" json:"album"`
}

// Child is the song/file element used by getAlbum, getMusicDirectory,
// search3, etc. We populate the fields that make sense for audio files.
type Child struct {
	ID          string `xml:"id,attr" json:"id"`
	Parent      string `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	IsDir       bool   `xml:"isDir,attr" json:"isDir"`
	Title       string `xml:"title,attr" json:"title"`
	Album       string `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist      string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Track       int    `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year        int    `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre       string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt    string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size        int64  `xml:"size,attr,omitempty" json:"size,omitempty"`
	ContentType string `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	Suffix      string `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	Duration    int    `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	BitRate     int    `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Path        string `xml:"path,attr,omitempty" json:"path,omitempty"`
	IsVideo     bool   `xml:"isVideo,attr" json:"isVideo"`
	DiscNumber  int    `xml:"discNumber,attr,omitempty" json:"discNumber,omitempty"`
	Created     string `xml:"created,attr,omitempty" json:"created,omitempty"`
	AlbumID     string `xml:"albumId,attr,omitempty" json:"albumId,omitempty"`
	ArtistID    string `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Type        string `xml:"type,attr,omitempty" json:"type,omitempty"`
}

type SearchResult3 struct {
	Artist []ArtistID3 `xml:"artist,omitempty" json:"artist,omitempty"`
	Album  []AlbumID3  `xml:"album,omitempty" json:"album,omitempty"`
	Song   []Child     `xml:"song,omitempty" json:"song,omitempty"`
}

type Directory struct {
	ID     string  `xml:"id,attr" json:"id"`
	Parent string  `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Name   string  `xml:"name,attr" json:"name"`
	Child  []Child `xml:"child" json:"child"`
}

// --- Stub payloads for endpoints we expose but don't fully back ---

type Playlists struct {
	Playlist []Playlist `xml:"playlist,omitempty" json:"playlist,omitempty"`
}

type Playlist struct {
	ID        string  `xml:"id,attr" json:"id"`
	Name      string  `xml:"name,attr" json:"name"`
	SongCount int     `xml:"songCount,attr" json:"songCount"`
	Duration  int     `xml:"duration,attr" json:"duration"`
	Public    bool    `xml:"public,attr" json:"public"`
	Owner     string  `xml:"owner,attr,omitempty" json:"owner,omitempty"`
	Created   string  `xml:"created,attr,omitempty" json:"created,omitempty"`
	Changed   string  `xml:"changed,attr,omitempty" json:"changed,omitempty"`
	Entry     []Child `xml:"entry,omitempty" json:"entry,omitempty"`
}

type User struct {
	Username            string `xml:"username,attr" json:"username"`
	Email               string `xml:"email,attr,omitempty" json:"email,omitempty"`
	ScrobblingEnabled   bool   `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	AdminRole           bool   `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole        bool   `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole        bool   `xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole          bool   `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole        bool   `xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole        bool   `xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole         bool   `xml:"commentRole,attr" json:"commentRole"`
	PodcastRole         bool   `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole          bool   `xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole         bool   `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole           bool   `xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool   `xml:"videoConversionRole,attr" json:"videoConversionRole"`
}

type Starred struct {
	Artist []ArtistIndexed `xml:"artist,omitempty" json:"artist,omitempty"`
	Album  []Child         `xml:"album,omitempty" json:"album,omitempty"`
	Song   []Child         `xml:"song,omitempty" json:"song,omitempty"`
}

type Starred2 struct {
	Artist []ArtistID3 `xml:"artist,omitempty" json:"artist,omitempty"`
	Album  []AlbumID3  `xml:"album,omitempty" json:"album,omitempty"`
	Song   []Child     `xml:"song,omitempty" json:"song,omitempty"`
}

type Genres struct {
	Genre []Genre `xml:"genre,omitempty" json:"genre,omitempty"`
}

type Genre struct {
	Value      string `xml:",chardata" json:"value"`
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
}

type Songs struct {
	Song []Child `xml:"song,omitempty" json:"song,omitempty"`
}

type NowPlaying struct {
	Entry []Child `xml:"entry,omitempty" json:"entry,omitempty"`
}

type ArtistInfo struct {
	Biography      string `xml:"biography,omitempty" json:"biography,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageURL string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageURL  string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type AlbumInfo struct {
	Notes          string `xml:"notes,omitempty" json:"notes,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageURL string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageURL  string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type ScanStatus struct {
	Scanning bool  `xml:"scanning,attr" json:"scanning"`
	Count    int64 `xml:"count,attr" json:"count"`
}

type Bookmarks struct {
	Bookmark []Bookmark `xml:"bookmark,omitempty" json:"bookmark,omitempty"`
}

type Bookmark struct {
	Entry    Child  `xml:"entry" json:"entry"`
	Position int64  `xml:"position,attr" json:"position"`
	Username string `xml:"username,attr,omitempty" json:"username,omitempty"`
	Comment  string `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Created  string `xml:"created,attr,omitempty" json:"created,omitempty"`
	Changed  string `xml:"changed,attr,omitempty" json:"changed,omitempty"`
}

type Podcasts struct {
	Channel []PodcastChannel `xml:"channel,omitempty" json:"channel,omitempty"`
}

type PodcastChannel struct {
	ID string `xml:"id,attr" json:"id"`
}

type NewestPodcasts struct {
	Episode []Child `xml:"episode,omitempty" json:"episode,omitempty"`
}

type InternetRadioStations struct {
	InternetRadioStation []InternetRadioStation `xml:"internetRadioStation,omitempty" json:"internetRadioStation,omitempty"`
}

type InternetRadioStation struct {
	ID          string `xml:"id,attr" json:"id"`
	Name        string `xml:"name,attr" json:"name"`
	StreamURL   string `xml:"streamUrl,attr" json:"streamUrl"`
	HomePageURL string `xml:"homePageUrl,attr,omitempty" json:"homePageUrl,omitempty"`
}

type Shares struct {
	Share []Share `xml:"share,omitempty" json:"share,omitempty"`
}

type Share struct {
	ID string `xml:"id,attr" json:"id"`
}

type Videos struct {
	Video []Child `xml:"video,omitempty" json:"video,omitempty"`
}

type Lyrics struct {
	Artist string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Title  string `xml:"title,attr,omitempty" json:"title,omitempty"`
	Value  string `xml:",chardata" json:"value,omitempty"`
}

type OpenSubsonicExtension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions,attr" json:"versions"`
}

// --- Response helpers ---

func newOKResponse() *Response {
	return &Response{
		Status:        "ok",
		Version:       subsonicAPIVersion,
		Type:          serverName,
		ServerVersion: serverVersion,
		OpenSubsonic:  true,
	}
}

func writeSubsonic(w http.ResponseWriter, r *http.Request, fill func(*Response)) {
	resp := newOKResponse()
	if fill != nil {
		fill(resp)
	}
	emit(w, r, resp)
}

func writeSubsonicError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	resp := newOKResponse()
	resp.Status = "failed"
	resp.Error = &ErrorPayload{Code: code, Message: msg}
	emit(w, r, resp)
}

func emit(w http.ResponseWriter, r *http.Request, resp *Response) {
	format := r.URL.Query().Get("f")
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		writeJSONEnvelope(w, resp)
	case "jsonp":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		cb := r.URL.Query().Get("callback")
		if cb == "" {
			cb = "callback"
		}
		io.WriteString(w, cb+"(")
		writeJSONEnvelope(w, resp)
		io.WriteString(w, ");")
	default:
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
		_ = xml.NewEncoder(w).Encode(resp)
	}
}

// writeJSONEnvelope marshals into the {"subsonic-response": ...} shape
// expected by Subsonic JSON clients.
func writeJSONEnvelope(w io.Writer, resp *Response) {
	wrapped := map[string]any{"subsonic-response": resp}
	_ = json.NewEncoder(w).Encode(wrapped)
}

// Helper for query params with default fallback.
func qInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	if err != nil {
		return def
	}
	return n
}
