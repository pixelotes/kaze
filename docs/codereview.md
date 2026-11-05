# Code Review: Kaze Music Server Architecture

## Overview

Kaze is a minimal, filesystem-first music server designed to implement the Subsonic API. Its core design philosophy is to be lightweight and efficient, avoiding traditional database layers by scanning the entire music library into memory upon startup.

### Key Characteristics
*   **Filesystem-First:** All data retrieval relies directly on the structure of the music files (`Artist/Album/Track`).
*   **In-Memory Index:** The library index is built and stored in memory at startup, ensuring fast lookups during runtime.
*   **Minimal Overhead:** It streams raw files without transcoding, aiming for low-power environments (like Raspberry Pi).
*   **No External Dependencies (Database):** The system relies solely on file system access and tag parsing.

## Application Components

The codebase is structured into several modules that handle specific responsibilities:

### 1. Configuration (`config/` and `internal/config`)
This component handles the application's setup.
*   **Function:** Loads environment variables and the `config/config.yml` file.
*   **Responsibility:** Stores critical settings such as the server port, library paths, and security credentials (e.g., Subsonic passwords and JWT secrets).

### 2. Media and Library Handling (`internal/media`)
This is the data layer responsible for interacting with the physical music library.
*   **`scanner.go`:** This module orchestrates the initial scan of the specified library paths. It walks the filesystem according to the expected `Artist/Album/Track` layout and builds the application's in-memory representation of the library.
*   **`tags.go`:** Responsible for metadata extraction. It utilizes the `dhowden/tag` library to read and parse metadata (title, artist, album, genre, etc.) from various file formats (ID3v2, Vorbis, MP4).
*   **`cover.go`:** Specifically handles the logic for locating and reading album cover art, checking for common filenames (`cover.jpg`, `folder.jpg`, etc.).
*   **`types.go`:** Defines the core data structures used to represent Artists, Albums, and Tracks within the in-memory index.

### 3. Server and API (`internal/server`)
This layer handles the network requests from clients and translates them into actions performed by the media layer.
*   **`server.go`:** The main HTTP server entry point. It initializes the server and routes incoming requests to the appropriate handlers.
*   **`subsonic.go`:** This is the implementation of the Subsonic API endpoints (e.g., `/artists`, `/album`, `/stream`). When a client requests data, this module queries the in-memory index provided by `internal/media`.
*   **`auth.go`:** Manages security. It validates client credentials (like Subsonic licenses or passwords) before allowing access to the API.
*   **`web.go`:** Handles serving static files, providing the user interface (UI) for server statistics and client connection helpers from the `web/` directory.

## Component Interaction Workflow

### Startup Sequence
1.  **Initialization:** The server starts and loads the configuration (`internal/config`).
2.  **Scanning:** The `internal/media` module's `scanner.go` is triggered, initiating a deep scan of all configured music library paths.
3.  **Indexing:** As files are found, `tags.go` and `cover.go` extract metadata and art, populating the in-memory index defined in `types.go`.
4.  **Ready:** The HTTP server (`server.go`) starts listening on the configured port.

### Runtime Request Flow (e.g., "List Artists")
1.  A Subsonic client sends an API request (e.g., `GET /artists`).
2.  The `internal/server` routes the request to the `subsonic.go` handler.
3.  The `subsonic.go` handler checks the request authenticity via `auth.go`.
4.  If authorized, `subsonic.go` queries the in-memory index managed by `internal/media`.
5.  The relevant data is retrieved, formatted according to the Subsonic API specification, and sent back to the client.

### Streaming Flow (e.g., "Play Track X")
1.  A client requests a stream for a specific file.
2.  `subsonic.go` locates the file path in the media index.
3.  The server reads the raw file content from the filesystem (via `internal/media`) and streams it directly back to the client without modification or transcoding.
