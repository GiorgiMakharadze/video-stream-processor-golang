package pkg

import "os"

type Config struct {
	WebSocketPort string
	RTMPBaseURL   string
	HLSBaseURL    string
}

func LoadConfig() *Config {
	wsPort := os.Getenv("WS_PORT")
	if wsPort == "" {
		wsPort = "9090"
	}

	rtmpBaseURL := os.Getenv("RTMP_BASE_URL")
	if rtmpBaseURL == "" {
		rtmpBaseURL = "rtmp://localhost/live"
	}

	hlsBaseURL := os.Getenv("HLS_BASE_URL")
	if hlsBaseURL == "" {
		hlsBaseURL = "http://localhost:8080/live"
	}

	return &Config{
		WebSocketPort: wsPort,
		RTMPBaseURL:   rtmpBaseURL,
		HLSBaseURL:    hlsBaseURL,
	}
}
