# Kaze Roadmap: Low Overhead Improvements

This document outlines proposed improvements for the Kaze music server. All suggested features prioritize maintaining the current design ethos: minimal resource usage, no database dependency, and maximum efficiency on limited hardware like the Raspberry Pi 3B. The focus is on robustness, usability, and filling minor functional gaps.

## 🚀 Phase 1: Stability & Core Functionality (High Priority)

These improvements are critical for making the application reliable in production environments without adding significant overhead.

### 1. Implement Download Endpoint
*   **Description:** Implement the `/download` endpoint, which is currently mentioned in the `README` but missing from the codebase.
*   **Why:** This is a core Subsonic feature that relies purely on the file system. It involves reading the specified track ID, locating the file path, and serving it with the appropriate `Content-Disposition` headers for file saving.
*   **Overhead:** Extremely low. It's a direct file I/O operation.

### 2. Concurrent Index Safety (Concurrency Fix)
*   **Description:** Introduce a synchronization mechanism (e.g., `sync.RWMutex`) around the in-memory library index (`s.lib`).
*   **Why:** Currently, the index is scanned and updated during `Rescan()` while other HTTP handlers are reading from it. Without a lock, this could lead to data races and application crashes. An `RWMutex` allows multiple read operations simultaneously (high performance) but ensures exclusive access during writes (scans).
*   **Overhead:** Very low. Mutexes are designed for efficient concurrent access in Go.

### 3. Robust Scan Error Logging
*   **Description:** Modify the `scanner.go` to log specific details when a file is skipped or fails during metadata extraction (e.g., "Permission denied accessing /path/to/file", or "Failed to parse ID3 tags on file X").
*   **Why:** Silent failures during a scan are difficult to debug. Logging these errors allows the user to fix file system issues without needing to restart the entire service.
*   **Overhead:** Negligible, as it only involves writing to a log file upon error.

## 🛠️ Phase 2: Usability & Polish (Medium Priority)

These improvements enhance the user experience and maintenance without adding heavy processing load.

### 4. Configuration Validation
*   **Description:** Implement comprehensive validation of the `config.yml` file during startup. Check for required fields (like `music` library path), ensure the port is valid, and verify that the provided library paths are actually readable and accessible.
*   **Why:** Failing fast on startup when configuration is incorrect prevents mysterious runtime errors.
*   **Overhead:** Zero after startup.

### 5. Granular Scan Status Updates
*   **Description:** Enhance the `handleGetScanStatus` endpoint to provide more detailed progress (e.g., "Files Scanned: 1200/5000", "Current Folder: Artist X/Album Y").
*   **Why:** The current scan status is binary (scanning/not scanning). More detail helps the user understand how long the process will take.
*   **Overhead:** Low. Only requires updating the internal state variables tracked during the scan.

## ✨ Phase 3: Feature Expansion (Long Term)

These features require minor state management but stay within the low-overhead philosophy.

### 6. Basic Playback Control Stubs
*   **Description:** Implement stubs for basic playback control endpoints (`play`, `pause`, `stop`, `next`, `prev`).
*   **Why:** While Kaze doesn't manage the client state, providing these endpoints allows clients to initiate control commands. The server can respond with success/failure without needing to track complex session state.
*   **Overhead:** Minimal. Responses are simple status codes or success messages.

---

Do you have any questions about these proposed steps or require further technical discussion on how to implement any specific improvement?