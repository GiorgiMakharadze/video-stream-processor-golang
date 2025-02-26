package ffmpeg

import (
	"bytes"
	"os/exec"
)

func ConvertToFLV(inputFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputFile,
		"-c:v", "libx264",
		"-f", "flv",
		outputFile,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
