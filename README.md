# kaze

Minimal, filesystem-first music server speaking the Subsonic API. Companion
to [suika](https://github.com/raul-pastor/suika).

- No database. Library is scanned in-memory at startup.
- Expects an `Artist/Album/Track` directory layout.
- Cover art is read from `cover.jpg`, `folder.jpg`, `album.jpg`, or `front.jpg`
  inside the album directory — embedded art is ignored on purpose.
- Tags (ID3v2 / Vorbis / MP4) are read with [dhowden/tag](https://github.com/dhowden/tag)
  to fill in title/track/disc/year/genre. Filename is the fallback.
- Streams raw files. No transcoding.
- Designed for a Raspberry Pi 3B on a LAN — clients do the heavy lifting.

## Subsonic endpoints implemented

`ping`, `getLicense`, `getMusicFolders`, `getIndexes`, `getArtists`,
`getArtist`, `getAlbum`, `getAlbumList2`, `search3`, `getMusicDirectory`,
`getCoverArt`, `stream`, `download`.

Tested clients: Symfonium (Android), Feishin (desktop), Sonixd (desktop).

## Layout expected

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

## Configuration

`config/config.yml`:

```yaml
app:
  port: 4533
  ui_password: "changeme"
  jwt_secret: "change-me-in-production"

libraries:
  - friendly_name: "Music"
    path: "./music"
```

Environment variables can be substituted with `${VAR}` syntax.

## Run

```
go run ./cmd/kaze -config config/config.yml
```

Or with Docker:

```
docker build -t kaze .
docker run -p 4533:4533 \
  -v ./config:/app/config:ro \
  -v /path/to/music:/data/music:ro \
  kaze
```

Open `http://localhost:4533` for the landing page (server stats and a copy
button for the Subsonic URL), then point your Subsonic client at the same
host with `/rest` as the protocol path.

## Caching

There isn't any, by design. Clients (Symfonium, Feishin, etc.) cache album
art and metadata locally — kaze just answers fresh from its in-memory index
every time. Rescans happen on process restart.
