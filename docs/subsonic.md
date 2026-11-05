# Subsonic API Endpoint Implementation Status

This report provides a status overview of the implemented and stubbed Subsonic API endpoints within Kaze, aligning with the project's goal of providing a minimal and lightweight server for low-power hardware.

Due to the resource constraints and the focus on core functionality, many advanced or non-essential Subsonic features are currently stubs (placeholders) and do not maintain state or perform complex logic.

## Endpoint Status Table

| Endpoint | Status | Notes |
| :--- | :--- | :--- |
| `/ping` | **Implemented** | Standard health check. |
| `/getLicense` | **Implemented** | Basic license check implementation. |
| `/getMusicFolders` | **Implemented** | Retrieves configured library paths. |
| `/getIndexes` | **Implemented** | Provides bucket structure for browsing. |
| `/getArtists` | **Implemented** | Lists artists indexed by name buckets. |
| `/getArtist` | **Implemented** | Retrieves detailed artist information. |
| `/getAlbum` | **Implemented** | Retrieves detailed album information. |
| `/getAlbumList2` | **Implemented** | Supports multiple sorting and filtering options. |
| `/search3` | **Implemented** | Core search functionality for artists, albums, and tracks. |
| `/getMusicDirectory` | **Implemented** | Legacy browsing interface for nested folders. |
| `/getCoverArt` | **Implemented** | Serves cover art from the file system. |
| `/stream` | **Implemented** | Streams raw media files directly from disk. |
| `/download` | **Missing** | Listed in `README.md` but no handler was found in the current codebase. |
| `/handleStartScan` | **Implemented** | Initiates the library scan in the background. |
| `/handleGetScanStatus` | **Implemented** | Reports current status of the scan. |
| `***` | ***Stubs*** | ***Placeholders for features that require user state or complex logic.*** |
| `/getPlaylists` | **Stub** | Returns an empty list. |
| `/getPlaylist` | **Stub** | Returns a 404 response. |
| `/createPlaylist` | **Stub** | No operation performed. |
| `/updatePlaylist` | **Stub** | No operation performed. |
| `/deletePlaylist` | **Stub** | No operation performed. |
| `/getUser` | **Stub** | Returns a hardcoded user profile. |
| `/scrobble` | **Stub** | No operation performed. |
| `/star` / `/unstar` | **Stub** | No operation performed. |
| `/setRating` | **Stub** | No operation performed. |
| `/getStarred` / `/getStarred2` | **Stub** | Returns an empty list. |
| `/getGenres` | **Stub** | Calculates and returns all available genres. |
| `/getRandomSongs` | **Stub** | Uses a deterministic hash for pseudo-random selection. |
| `/getSongsByGenre` | **Stub** | Filters and returns songs based on genre. |
| `/getNowPlaying` | **Stub** | Returns an empty "now playing" state. |
| `/getTopSongs` | **Stub** | Returns an empty list of top songs. |
| `/getSimilarSongs` / `/getSimilarSongs2` | **Stub** | Returns empty lists. |
| `/getArtistInfo` / `/getArtistInfo2` | **Stub** | Returns empty information. |
| `/getAlbumInfo` / `/getAlbumInfo2` | **Stub** | Returns empty information. |
| `/getAvatar` | **Stub** | Returns a 404 as no avatars are supported. |
| `/getLyrics` | **Stub** | No lyric extraction logic implemented. |
| `/getPodcasts` / `...` | **Stub** | Placeholder endpoints for podcast and radio features. |
| `/getOpenSubsonicExtensions` | **Stub** | Declares support without claiming any extensions. |

## Conclusion

Kaze successfully implements the core music library and streaming capabilities (`getArtists`, `stream`, `search3`, etc.). This allows users to browse and play music reliably. The extensive use of stubs is a deliberate architectural choice to keep the application footprint minimal and fast, deferring complex state-management features (like user profiles, play history, or detailed genre statistics) until they are deemed strictly necessary or the hardware is upgraded.

***

Do you have any questions or require further analysis on specific endpoints?