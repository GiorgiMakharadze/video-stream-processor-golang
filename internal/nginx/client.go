package nginx

import (
	"log"
	"os/exec"
)

func StreamFLVToRTMP(flvPath, rtmpURL string) error {
	cmd := exec.Command("ffmpeg",
		"-re",
		"-i", flvPath,
		"-c:v", "libx264",
		"-f", "flv",
		rtmpURL,
	)

	log.Printf("Streaming FLV file %s to RTMP endpoint %s", flvPath, rtmpURL)

	if err := cmd.Run(); err != nil {
		log.Println("Error streaming FLV file:", err)
		return err
	}
	return nil
}
