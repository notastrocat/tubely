package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const (
	maxMemory = 10 << 20 // 10 MB
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// Set max memory to parse the thumbnail file
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse the file on server", err)
		return
	}

	// Parse the thumbnail from request
	multiPartFile, multiPartFileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get thumbnail", err)
		return
	}

	// Get the mediaType from the request header
	fileType := multiPartFileHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(fileType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse MIME header", err)
		return
	}

	if mediaType != "image/png" && mediaType != "image/jpeg" {
		respondWithError(w, http.StatusNotAcceptable, "Wrong media type uploaded", err)
		return
	}

	// Read the image into a byte array
	// imageData, err := io.ReadAll(multiPartFile)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't read thumbnail", err)
	// 	return
	// }

	// Get video information using the videoID from the URL
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch thumbnail", err)
		return
	}

	// Check if the request's userID matches the video's owner.
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Stop this nasty business", err)
		return
	}

	// Add thumbnail to the global map, using videoID as key
	// videoThumbnails[videoID] = thumbnail{
	// 	data: imageData,
	// 	mediaType: fileType,
	// }
	// storing in-memory is very bad

	// Encode the entire image into a base64 string to be able to store it inside the SQLite table
	// imageStr := base64.StdEncoding.EncodeToString(imageData)
	// storing in base64 is also bad as far as database queries are concerned.

	fileExt := strings.Split(fileType, "/")

	tnFileName := (videoIDString + "." + fileExt[len(fileExt)-1])
	tnFilePath := filepath.Join(cfg.assetsRoot, tnFileName)
	tnFile, err := os.Create(tnFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail", err)
		return
	}

	_, err = io.Copy(tnFile, multiPartFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save thumbnail", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, tnFileName)
	video.ThumbnailURL = &thumbnailURL

	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
