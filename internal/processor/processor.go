package processor

import (
	"log"
	"os"

	"github.com/GiorgiMakharadze/video-stream-processor-golang.git/internal/ffmpeg"
	"github.com/GiorgiMakharadze/video-stream-processor-golang.git/internal/nginx"
	config "github.com/GiorgiMakharadze/video-stream-processor-golang.git/pkg"
)

type VideoTask struct {
	InputFile string
	Config    *config.Config
}

type VideoProcessor struct {
	TaskQueue   chan VideoTask
	WorkerCount int
}

func NewVideoProcessor(workerCount int) *VideoProcessor {
	return &VideoProcessor{
		TaskQueue:   make(chan VideoTask, 100),
		WorkerCount: workerCount,
	}
}

func (vp *VideoProcessor) Start() {
	for i := 0; i < vp.WorkerCount; i++ {
		go vp.worker(i)
	}
}

func (vp *VideoProcessor) worker(id int) {
	log.Printf("Video processor worker %d started", id)
	for task := range vp.TaskQueue {
		processVideo(task.InputFile, task.Config)
	}
}

func (vp *VideoProcessor) EnqueueTask(inputFile string, cfg *config.Config) {
	vp.TaskQueue <- VideoTask{
		InputFile: inputFile,
		Config:    cfg,
	}
}

func processVideo(inputFile string, cfg *config.Config) {
	outFile, err := os.CreateTemp("", "output_*.flv")
	if err != nil {
		log.Println("Error creating temporary output file:", err)
		return
	}
	outFile.Close()
	outputFile := outFile.Name()

	if err := ffmpeg.ConvertToFLV(inputFile, outputFile); err != nil {
		log.Println("FFmpeg conversion error:", err)
		return
	}
	log.Println("Conversion complete. FLV file saved as:", outputFile)

	if err := nginx.StreamFLVToRTMP(outputFile, cfg.RTMPURL); err != nil {
		log.Println("Error streaming FLV file to RTMP server:", err)
		return
	}
	log.Println("FLV video successfully streamed to RTMP server.")

	if err := os.Remove(inputFile); err != nil {
		log.Println("Error removing input temp file:", err)
	}
	if err := os.Remove(outputFile); err != nil {
		log.Println("Error removing output FLV file:", err)
	}
}
