# Video Stream Processor

This project is a server application designed to receive video streams from a frontend via WebSocket, process the video using FFmpeg to convert it into FLV format, and then stream the converted video to an Nginx server (typically used for RTMP streaming or further storage).

## Overview

The application is built in Go and leverages a worker pool architecture to handle multiple simultaneous uploads efficiently. When a client connects via WebSocket, the server receives video chunks, writes them to a temporary file, and enqueues a processing task. Worker goroutines then process these tasks asynchronously by converting the video using FFmpeg and streaming the resulting FLV file to an RTMP endpoint hosted by Nginx.

## Architecture

The project consists of three main components:

1. **WebSocket Server**  
   - Uses the [gorilla/websocket](https://github.com/gorilla/websocket) library to handle WebSocket connections.
   - Receives binary video chunks from clients and stores them in a temporary file.

2. **Worker Pool for Video Processing**  
   - Implements a task queue to decouple the receipt of data from the intensive video processing work.
   - Each task involves converting the video file to FLV format using FFmpeg and streaming it to an RTMP endpoint via an Nginx server.

3. **FFmpeg Integration and Nginx Streaming**  
   - Uses FFmpeg commands to convert and stream video.
   - The conversion step encodes video to FLV, and the streaming step sends the converted file to a configured RTMP server.

## Project Structure

Below is a suggested folder structure:

