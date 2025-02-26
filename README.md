# Video Stream Processor

This project is a **WebSocket-based video processing server** built with **Go**. It allows clients to send video streams in chunks over WebSocket, processes the received video using **FFmpeg**, converts it to FLV format, and then streams the output to an **RTMP server (Nginx-RTMP)** for further handling (e.g., live streaming or storage).

## How It Works

1. **WebSocket Connection:**
   - Clients connect to the WebSocket server (`/ws` endpoint).
   - They send video data in binary chunks over the connection.
   
2. **Receiving Video Data:**
   - The server collects these chunks and writes them into a temporary file.
   
3. **Processing Video:**
   - Once the full video is received, a task is queued for processing.
   - The task is picked up by a worker from a pool of concurrent workers.
   - FFmpeg converts the received file into FLV format.

4. **Streaming to RTMP Server:**
   - The converted FLV file is streamed to an RTMP endpoint (configured in environment variables).
   - After streaming, the temporary files are deleted to free up space.

## Features
- **WebSocket support** for real-time video streaming.
- **Asynchronous processing** using a worker pool to handle multiple uploads efficiently.
- **FFmpeg integration** for video conversion to FLV format.
- **RTMP streaming support** for live video broadcasting.
- **Optimized resource management** with temporary file cleanup and background processing.

## Environment Variables
The server reads configurations from environment variables:
- `WS_PORT`: WebSocket server port (default: `9090`)
- `RTMP_URL`: RTMP server URL (default: `rtmp://localhost/live/stream`)
- `HLS_URL`: HLS playback URL (default: `http://localhost:8080/hls/live/stream.m3u8`)

## Usage
1. Start the Go WebSocket server.
2. Connect a client and start sending a video stream.
3. The server will process the video asynchronously and stream it to the RTMP server.

## Dependencies
- `gorilla/websocket` (WebSocket handling)
- `FFmpeg` (Video processing and conversion)
- `Nginx-RTMP` (Streaming server)

This system ensures **efficient, concurrent, and scalable** video processing while keeping the WebSocket server responsive.

