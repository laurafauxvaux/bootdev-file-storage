package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	mediaType := header.Header.Get("Content-Type")

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video", err)
		return
	}
	if video.UserID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fileType, err := mime.ExtensionsByType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get file type", err)
		return
	}
	fileName := fmt.Sprintf("%s%s", videoIDString, fileType[0])

	destPath := filepath.Join(cfg.assetsRoot, fileName)

	newFile, err := os.Create(destPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create file", err)
		return
	}
	defer newFile.Close()

	if _, err := io.Copy(newFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to write file", err)
		return
	}

	thumbnailURLString := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileName)
	video.ThumbnailURL = &thumbnailURLString

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
