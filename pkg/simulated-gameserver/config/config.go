package config

import "flag"

var (
	EnablePlayerTracking = flag.Bool("enable_player_tracking", false, "Enable player tracking with Agones")
	PlayerTrackingSlots  = flag.Int64("player_tracking_slots", 50, "Number of slots/max players")
	HighDensity          = flag.Bool("high_density", true, "Enable high density mode (multiple GameServers on one instance)")
)

func Init() {
	flag.Parse()
}
