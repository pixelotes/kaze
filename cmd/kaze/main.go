// Command kaze is a minimal, filesystem-first Subsonic-compatible music server.
//
// Usage:
//
//	kaze -config config/config.yml
package main

import (
	"flag"
	"log"
	"time"

	"kaze/internal/config"
	"kaze/internal/media"
	"kaze/internal/server"
)

func main() {
	configPath := flag.String("config", "config/config.yml", "path to the YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("loading config %s: %v", *configPath, err)
	}

	// Each configured library becomes a Subsonic music folder. Scanning also
	// registers it, so Library.Rescan (startScan) can walk it again later.
	// A missing or unreadable library is logged and skipped rather than
	// stopping the server.
	lib := media.NewLibrary()
	start := time.Now()
	for _, l := range cfg.Libraries {
		if err := lib.Scan(l.FriendlyName, l.Path); err != nil {
			log.Printf("library %q: %v", l.FriendlyName, err)
		}
	}
	st := lib.Stats()
	log.Printf("library ready in %s: %d artists / %d albums / %d tracks",
		time.Since(start).Round(time.Millisecond), st.Artists, st.Albums, st.Tracks)

	log.Fatal(server.New(cfg, lib).Start())
}
