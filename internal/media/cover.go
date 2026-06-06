package media

import (
	"os"
	"path/filepath"
	"strings"
)

// FindCover returns the path of a cover image inside dirPath, or "" if none.
// Priority: cover.* > folder.* > album.* > front.* > any other image at the root.
func FindCover(dirPath string) string {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return ""
	}

	priorities := []string{"cover", "folder", "album", "front"}
	for _, prefix := range priorities {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			ext := filepath.Ext(name)
			base := strings.TrimSuffix(name, ext)
			if base == prefix {
				if _, ok := imageExts[ext]; ok {
					return filepath.Join(dirPath, entry.Name())
				}
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := imageExts[ext]; ok {
			return filepath.Join(dirPath, entry.Name())
		}
	}

	return ""
}

// CoverContentType returns the MIME type for an image path.
func CoverContentType(path string) string {
	if ct, ok := imageExts[strings.ToLower(filepath.Ext(path))]; ok {
		return ct
	}
	return "application/octet-stream"
}
