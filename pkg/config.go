package config

import (
	"os"
)

type Config struct {
	WebSocketPort string
	RTMPURL       string
	HLSURL        string
}

func LoadConfig() *Config {
	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "9090"
	}

	rtmpURL := os.Getenv("RTMP_URL")
	if rtmpURL == "" {
		rtmpURL = "rtmp://localhost/live/stream"
	}

	hlsURL := os.Getenv("HLS_URL")
	if hlsURL == "" {
		hlsURL = "http://localhost:8080/hls/live/stream.m3u8"
	}

	return &Config{
		WebSocketPort: wsPort,
		RTMPURL:       rtmpURL,
		HLSURL:        hlsURL,
	}
}
