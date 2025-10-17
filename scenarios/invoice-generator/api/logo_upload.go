package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	// MaxLogoSize is 5MB
	MaxLogoSize = 5 * 1024 * 1024
	// LogoStoragePath is where logos are stored
	LogoStoragePath = "./uploads/logos"
)

// Allowed logo file extensions and MIME types
var allowedLogoTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/svg+xml": ".svg",
	"image/webp": ".webp",
}

// LogoUploadResponse represents the response after successful upload
type LogoUploadResponse struct {
	Success  bool   `json:"success"`
	LogoID   string `json:"logo_id"`
	LogoURL  string `json:"logo_url"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	Message  string `json:"message,omitempty"`
}

// UploadLogoHandler handles logo file uploads
func UploadLogoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Ensure upload directory exists
	if err := os.MkdirAll(LogoStoragePath, 0755); err != nil {
		log.Printf("Error creating upload directory: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to initialize upload directory"})
		return
	}

	// Parse multipart form (max 10MB in memory)
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid multipart form data"})
		return
	}

	// Get the file from form
	file, header, err := r.FormFile("logo")
	if err != nil {
		log.Printf("Error retrieving file from form: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing or invalid 'logo' file in request"})
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > MaxLogoSize {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("File size exceeds maximum allowed size of %d bytes (5MB)", MaxLogoSize),
		})
		return
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	ext, allowed := allowedLogoTypes[contentType]
	if !allowed {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Invalid file type '%s'. Allowed types: JPEG, PNG, SVG, WebP", contentType),
		})
		return
	}

	// Generate unique filename
	logoID := uuid.New().String()
	fileName := fmt.Sprintf("%s%s", logoID, ext)
	filePath := filepath.Join(LogoStoragePath, fileName)

	// Create the file on disk
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	written, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("Error writing file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write file to disk"})
		return
	}

	// Get the API port for constructing URL
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		log.Printf("Error: API_PORT environment variable not set")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server configuration error: API_PORT not set"})
		return
	}

	// Construct logo URL
	logoURL := fmt.Sprintf("http://localhost:%s/api/logos/%s", apiPort, fileName)

	// Return success response
	response := LogoUploadResponse{
		Success:  true,
		LogoID:   logoID,
		LogoURL:  logoURL,
		FileName: fileName,
		FileSize: written,
		Message:  "Logo uploaded successfully",
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("Logo uploaded successfully: %s (size: %d bytes)", fileName, written)
}

// GetLogoHandler serves uploaded logo files
func GetLogoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileName := vars["filename"]

	// Validate filename (prevent directory traversal)
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid filename"})
		return
	}

	// Construct file path
	filePath := filepath.Join(LogoStoragePath, fileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Logo not found"})
		return
	}

	// Determine content type from file extension
	ext := filepath.Ext(fileName)
	var contentType string
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".svg":
		contentType = "image/svg+xml"
	case ".webp":
		contentType = "image/webp"
	default:
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filePath)
}

// DeleteLogoHandler deletes an uploaded logo
func DeleteLogoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	fileName := vars["filename"]

	// Validate filename (prevent directory traversal)
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid filename"})
		return
	}

	// Construct file path
	filePath := filepath.Join(LogoStoragePath, fileName)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Logo not found"})
		return
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		log.Printf("Error deleting file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete logo"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logo deleted successfully",
	})
	log.Printf("Logo deleted successfully: %s", fileName)
}

// ListLogosHandler lists all uploaded logos
func ListLogosHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Ensure directory exists
	if _, err := os.Stat(LogoStoragePath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"logos": []interface{}{},
			"count": 0,
		})
		return
	}

	// Read directory
	files, err := os.ReadDir(LogoStoragePath)
	if err != nil {
		log.Printf("Error reading logo directory: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list logos"})
		return
	}

	// Get API port for constructing URLs
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		log.Printf("Error: API_PORT environment variable not set")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server configuration error: API_PORT not set"})
		return
	}

	// Build response with file info
	type LogoInfo struct {
		FileName string `json:"file_name"`
		LogoURL  string `json:"logo_url"`
		Size     int64  `json:"size"`
	}

	logos := make([]LogoInfo, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		logos = append(logos, LogoInfo{
			FileName: file.Name(),
			LogoURL:  fmt.Sprintf("http://localhost:%s/api/logos/%s", apiPort, file.Name()),
			Size:     info.Size(),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logos": logos,
		"count": len(logos),
	})
}
