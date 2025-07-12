package handlers

import (
	"encoding/json"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forum/middleware"
	"forum/models"
	"forum/repository"
	"forum/utils"
)

// Base directory for storing uploaded images
var ImageBaseDir = filepath.Join("ui", "static", "uploads", "images")

// ImageHandler handles image upload requests
type ImageHandler struct {
	Repo *repository.ImageRepository
}

func NewImageHandler(repo *repository.ImageRepository) *ImageHandler {
	return &ImageHandler{Repo: repo}
}

func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(21 << 20); err != nil {
		utils.ErrorResponse(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	postID := r.FormValue("post_id")
	if postID == "" {
		utils.ErrorResponse(w, "post_id required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		utils.ErrorResponse(w, "image field required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size > 20*1024*1024 {
		utils.ErrorResponse(w, "Image exceeds 20 MB limit", http.StatusBadRequest)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		file.Seek(0, io.SeekStart)
		mimeType = http.DetectContentType(buf[:n])
	}
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/gif" {
		utils.ErrorResponse(w, "Unsupported image type", http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		switch mimeType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		}
	}

	dateDir := time.Now().Format("2006-01-02")
	saveDir := filepath.Join(ImageBaseDir, user.ID, dateDir)
	thumbDir := filepath.Join(saveDir, "thumbnails")
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		utils.ErrorResponse(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	uuid := utils.GenerateUUID()
	filename := uuid + ext
	filePath := filepath.Join(saveDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		utils.ErrorResponse(w, "Failed to save image", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		utils.ErrorResponse(w, "Failed to save image", http.StatusInternalServerError)
		return
	}
	dst.Close()

	imgFile, err := os.Open(filePath)
	if err != nil {
		utils.ErrorResponse(w, "Failed to process image", http.StatusInternalServerError)
		return
	}
	defer imgFile.Close()

	var img image.Image
	switch mimeType {
	case "image/jpeg":
		img, err = jpeg.Decode(imgFile)
	case "image/png":
		img, err = png.Decode(imgFile)
	case "image/gif":
		img, err = gif.Decode(imgFile)
	}
	if err != nil {
		utils.ErrorResponse(w, "Failed to decode image", http.StatusBadRequest)
		return
	}

	thumb := createThumbnail(img, 150, 150)
	thumbPath := filepath.Join(thumbDir, filename)
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		utils.ErrorResponse(w, "Failed to save thumbnail", http.StatusInternalServerError)
		return
	}
	switch mimeType {
	case "image/jpeg":
		jpeg.Encode(thumbFile, thumb, &jpeg.Options{Quality: 80})
	case "image/png":
		png.Encode(thumbFile, thumb)
	case "image/gif":
		gif.Encode(thumbFile, thumb, nil)
	}
	thumbFile.Close()

	relPath := filepath.ToSlash(filepath.Join("uploads", "images", user.ID, dateDir, filename))
	relThumb := filepath.ToSlash(filepath.Join("uploads", "images", user.ID, dateDir, "thumbnails", filename))

	record := models.Image{
		PostID:        postID,
		UserID:        user.ID,
		Path:          "/static/" + relPath,
		ThumbnailPath: "/static/" + relThumb,
	}
	if _, err := h.Repo.Create(record); err != nil {
		utils.ErrorResponse(w, "Failed to save record", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"path":           record.Path,
		"thumbnail_path": record.ThumbnailPath,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func createThumbnail(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// fill background white
	white := image.NewUniform(image.White)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dst.Set(x, y, white.C)
		}
	}

	sb := src.Bounds()
	rw := width
	rh := height
	if sb.Dx()*height > sb.Dy()*width {
		rh = sb.Dy() * width / sb.Dx()
	} else {
		rw = sb.Dx() * height / sb.Dy()
	}
	resized := resizeNearest(src, rw, rh)
	offX := (width - rw) / 2
	offY := (height - rh) / 2
	for y := 0; y < rh; y++ {
		for x := 0; x < rw; x++ {
			dst.Set(x+offX, y+offY, resized.At(x, y))
		}
	}
	return dst
}

func resizeNearest(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + y*(sb.Dy())/h
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x*(sb.Dx())/w
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
