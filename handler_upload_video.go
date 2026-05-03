package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const (
	maxUploadSize  = 1 << 30   // 1.0 GB limit for the whole request
	maxMemorySpill = 500 << 20 // 500 MB max in RAM before spilling to disk temp files
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Wrap the body to hard-cap the upload size and prevent connection hogging
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	// Set max memory to parse the video file in-memory, rest is written to the disk
	err = r.ParseMultipartForm(maxMemorySpill)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse the file on server", err)
		return
	}

	// Parse the video from request
	multiPartFile, multiPartFileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}
	defer multiPartFile.Close()

	// Get the mediaType from the request header
	fileType := multiPartFileHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(fileType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse MIME header", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusNotAcceptable, "Wrong media type uploaded", err)
		return
	}

	// Get video information using the videoID from the URL
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch video metadata", err)
		return
	}

	// Check if the request's userID matches the video's owner.
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized owner", err)
		return
	}

	random := make([]byte, 32)
	_, err = rand.Read(random)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create random string", err)
		return
	}

	randomString := base64.RawURLEncoding.EncodeToString(random)

	videoFileName := randomString + ".mp4"
	videoFilePath := filepath.Join(cfg.assetsRoot, videoFileName)

	destFile, err := os.Create(videoFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create video file on disk", err)
		return
	}
	// Note: We removed the `defer os.Remove(...)` so the file actually persists.
	defer destFile.Close()

	_, err = io.Copy(destFile, multiPartFile)
	if err != nil {
		// If the copy fails halfway, clean up the corrupted/partial file manually
		os.Remove(videoFilePath)
		respondWithError(w, http.StatusInternalServerError, "Couldn't save video data", err)
		return
	}

	prefix, err := getVideoAspectRatio(videoFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't rename video", err)
		return
	}

	finalVideoName := prefix + "-" + videoFileName
	finalVideoPath := filepath.Join(cfg.assetsRoot, finalVideoName)

	os.Rename(videoFilePath, finalVideoPath)

	videoURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, finalVideoName)

	video.VideoURL = &videoURL

	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
