# kaze

Self-hosted music server speaking the Subsonic API, with a small web player, served straight from the filesystem.

Part of [MangoStack](https://github.com/pixelotes/MangoStack), a self-hosted media stack for the Raspberry Pi.

## Footprint

- Memory: a few MB; negligible next to Navidrome or Airsonic. MangoStack runs it with `GOGC=20` and `GOMEMLIMIT=32MiB`. What drives memory: the in-memory library index, which grows with the number of tracks.
- No cache, by design. Clients (Symfonium, Feishin, etc.) cache art and metadata themselves; kaze answers from its in-memory index every time.
- No database. The library is scanned into memory at startup and on `startScan` / `scanLibrary`.
- No transcoding: files are streamed as-is, so CPU stays low.
- Disk writes: none. Libraries can be mounted read-only.

## Features

- Subsonic API under `/rest` (both `/rest/foo` and `/rest/foo.view`)
- Tags (ID3v2 / Vorbis / MP4) read with [dhowden/tag](https://github.com/dhowden/tag) for title/track/disc/year/genre; filename is the fallback
- Cover art from `cover.jpg`, `folder.jpg`, `album.jpg` or `front.jpg` in the album directory; embedded art is ignored on purpose
- Web UI: landing page with server stats and a copy button for the Subsonic URL, directory browsing, in-browser player, downloads
- Single shared password + JWT auth, optional named users
- Multiple libraries

## Quick start

The container reads its config from `/app/config/config.yml` and ships without a config or library, so mount both:

```yaml
services:
  kaze:
    image: pixelotes/kaze:arm64
    ports:
      - "4533:4533"
    volumes:
      - ./config:/app/config:ro
      - /path/to/music:/app/media:ro
    restart: unless-stopped
```

Set the library `path` in `config.yml` to `/app/media`. Inside MangoStack, kaze is already wired in `docker-compose.yaml`.

## Configuration

`config/config.yml`:

```yaml
app:
  port: 4533  # default 4533
  ui_password: "${UI_PASSWORD}"  # required unless users are defined
  jwt_secret: "${JWT_SECRET}"  # default "change-me-in-production"

libraries:
  - friendly_name: "Music"  # required
    path: "/app/media"  # required, relative paths resolved at startup
```

`${VAR}` and `$VAR` references are replaced with environment variables when the file is loaded; unset variables are left as-is.

Optional `users:` (list of `username` / `password`) replaces the single shared password with named logins. A `libraries` list per user is parsed but not enforced yet; every user sees every library.

Login returns a JWT valid for 30 days, signed with `jwt_secret`.

Subsonic clients log in with any username and the shared password (or a named user's credentials). Plain `p=`, `p=enc:<hex>` and salted token `t=` + `s=` are accepted.

## Usage

Open `http://<host>:4533` for the web UI, then point your Subsonic client at the same host (`/rest` is the protocol path).

Implemented endpoints: `ping`, `getLicense`, `getMusicFolders`, `getIndexes`, `getArtists`, `getArtist`, `getAlbum`, `getAlbumList2`, `search3`, `getMusicDirectory`, `getCoverArt`, `stream`, `download`, `getGenres`, `getRandomSongs`, `getSongsByGenre`, `getUser`, `startScan` / `scanLibrary`, `getScanStatus`.

Accepted but empty or no-op (so clients do not error): playlists, stars, ratings, scrobble, bookmarks, now playing, top/similar songs, artist/album info, avatar, podcasts, radio, shares, videos, lyrics, `getOpenSubsonicExtensions`.

Tested clients: Symfonium (Android), Feishin (desktop), Sonixd (desktop).

Expected layout (`Artist/Album/Track`):

```
music/
├── Artist Name/
│   ├── Album Name/
│   │   ├── 01 - Title.flac
│   │   ├── 02 - Title.flac
│   │   └── cover.jpg
│   └── Other Album/
│       └── ...
└── Other Artist/
    └── ...
```

## Building from source

```sh
go build -o kaze ./cmd/kaze
./kaze -config config/config.yml
```

The binary serves the UI from `./web`, so run it from the repo root (or next to a copy of `web/`). Docker:

```sh
docker build -t kaze .
```

## Limits

- Do not expose it to the internet. It assumes a trusted LAN: one shared password (or a short plain-text user list), no rate limiting, no TLS.
- Change `jwt_secret` from the default; anyone who knows it can mint tokens.
- No transcoding, playlists, stars or scrobbling.
- Changes on disk are picked up only on restart or a client-triggered scan.

## Part of MangoStack

| App | Role |
|---|---|
| [Scarf](https://github.com/pixelotes/scarf) | Indexer proxy (Torznab) |
| [Parasol](https://github.com/pixelotes/parasol) | Optional Cloudflare solver for Scarf |
| [Reel](https://github.com/pixelotes/reel) | Movie & TV automation |
| [Tango](https://github.com/pixelotes/tango) | Web UI and REST API for aria2c |
| [RMS](https://github.com/pixelotes/rms) | Media server (Jellyfin API) |
| [Suika](https://github.com/pixelotes/suika) | Manga reader |
| [Kaze](https://github.com/pixelotes/kaze) | Music server (Subsonic API) |
| [Sumi](https://github.com/pixelotes/sumi) | EPUB reader |
| [Dashboarr](https://github.com/pixelotes/dashboarr) | Homepage |

## License

MIT
