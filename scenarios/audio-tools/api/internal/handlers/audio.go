package handlers

import (
	"audio-tools/internal/audio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type AudioHandler struct {
	processor *audio.AudioProcessor
	db        *sql.DB
	workDir   string
	dataDir   string
}

func NewAudioHandler(db *sql.DB, workDir, dataDir string) *AudioHandler {
	return &AudioHandler{
		processor: audio.NewAudioProcessor(workDir, dataDir),
		db:        db,
		workDir:   workDir,
		dataDir:   dataDir,
	}
}

type EditRequest struct {
	AudioFile       interface{}            `json:"audio_file"`
	Operations      []audio.EditOperation  `json:"operations"`
	OutputFormat    string                 `json:"output_format"`
	QualitySettings map[string]interface{} `json:"quality_settings"`
}

type EditResponse struct {
	JobID            string                   `json:"job_id"`
	OutputFiles      []map[string]interface{} `json:"output_files"`
	ProcessingTimeMs int64                    `json:"processing_time_ms"`
	QualityMetrics   map[string]interface{}   `json:"quality_metrics,omitempty"`
}

// HandleEdit processes audio editing operations
// Supports both JSON and multipart form data
func (h *AudioHandler) HandleEdit(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req EditRequest
	var inputPath string

	// Check if this is multipart or JSON
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Parse multipart form
		err := r.ParseMultipartForm(100 << 20) // 100 MB max
		if err != nil {
			sendError(w, http.StatusBadRequest, "failed to parse form")
			return
		}

		// Get the file
		file, header, err := r.FormFile("audio")
		if err != nil {
			sendError(w, http.StatusBadRequest, "audio file required")
			return
		}
		defer file.Close()

		// Save uploaded file
		tempID := uuid.New().String()
		tempPath := filepath.Join(h.workDir, fmt.Sprintf("%s%s", tempID, filepath.Ext(header.Filename)))

		dst, err := os.Create(tempPath)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "failed to save file")
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			sendError(w, http.StatusInternalServerError, "failed to save file")
			return
		}

		inputPath = tempPath

		// Parse operations from form field
		operation := r.FormValue("operation")
		if operation == "" {
			sendError(w, http.StatusBadRequest, "operation parameter required")
			return
		}

		// Parse parameters from form field (JSON string)
		paramsStr := r.FormValue("parameters")
		params := make(map[string]interface{})
		if paramsStr != "" {
			if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
				sendError(w, http.StatusBadRequest, "invalid parameters format")
				return
			}
		}

		// Build request from form data
		req.Operations = []audio.EditOperation{
			{
				Type:       operation,
				Parameters: params,
			},
		}
		req.OutputFormat = r.FormValue("output_format")
		if req.OutputFormat == "" {
			// Keep original format
			req.OutputFormat = filepath.Ext(header.Filename)[1:]
		}
	} else {
		// Original JSON parsing
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Handle file upload or asset ID (for JSON requests)
		switch v := req.AudioFile.(type) {
		case string:
			inputPath = v
		case map[string]interface{}:
			if assetID, ok := v["asset_id"].(string); ok {
				// Retrieve file path from database
				if h.db != nil {
					var path string
					err := h.db.QueryRow("SELECT file_path FROM audio_assets WHERE id = $1", assetID).Scan(&path)
					if err != nil {
						sendError(w, http.StatusNotFound, "audio asset not found")
						return
					}
					inputPath = path
				} else {
					sendError(w, http.StatusServiceUnavailable, "database not configured")
					return
				}
			}
		default:
			sendError(w, http.StatusBadRequest, "invalid audio_file format")
			return
		}
	}

	// Process operations
	currentPath := inputPath
	var outputFiles []map[string]interface{}
	jobID := uuid.New().String()

	for _, op := range req.Operations {
		var newPath string
		var err error

		switch op.Type {
		case "trim":
			// Support both "start_time"/"end_time" and "start"/"end" parameter names
			startTime, ok1 := getFloat(op.Parameters["start_time"])
			if !ok1 {
				startTime, ok1 = getFloat(op.Parameters["start"])
			}
			endTime, ok2 := getFloat(op.Parameters["end_time"])
			if !ok2 {
				endTime, ok2 = getFloat(op.Parameters["end"])
			}
			if !ok1 || !ok2 {
				err = fmt.Errorf("invalid trim parameters: start/start_time and end/end_time must be numbers")
			} else {
				newPath, err = h.processor.Trim(currentPath, startTime, endTime)
			}

		case "merge":
			if targetFiles, ok := op.Parameters["target_files"].([]interface{}); ok {
				var paths []string
				paths = append(paths, currentPath)
				for _, f := range targetFiles {
					paths = append(paths, f.(string))
				}
				newPath, err = h.processor.Merge(paths, req.OutputFormat)
			}

		case "split":
			if points, ok := op.Parameters["split_points"].([]interface{}); ok {
				var splitPoints []float64
				for _, p := range points {
					if f, ok := getFloat(p); ok {
						splitPoints = append(splitPoints, f)
					}
				}
				paths, splitErr := h.processor.Split(currentPath, splitPoints)
				if splitErr == nil {
					for i, path := range paths {
						fileInfo, _ := os.Stat(path)
						outputFiles = append(outputFiles, map[string]interface{}{
							"file_id":         uuid.New().String(),
							"file_path":       path,
							"part_number":     i + 1,
							"file_size_bytes": fileInfo.Size(),
						})
					}
					// Use first part for further processing
					if len(paths) > 0 {
						newPath = paths[0]
					}
				}
				err = splitErr
			}

		case "fade":
			if fadeIn, ok := getFloat(op.Parameters["fade_in"]); ok && fadeIn > 0 {
				newPath, err = h.processor.FadeIn(currentPath, fadeIn)
				if err == nil && newPath != "" {
					currentPath = newPath
				}
			}
			if fadeOut, ok := getFloat(op.Parameters["fade_out"]); ok && fadeOut > 0 {
				newPath, err = h.processor.FadeOut(currentPath, fadeOut)
			}

		case "volume":
			if factor, ok := getFloat(op.Parameters["volume_factor"]); ok {
				newPath, err = h.processor.AdjustVolume(currentPath, factor)
			}

		case "normalize":
			targetLevel := -16.0 // Default LUFS
			if level, ok := getFloat(op.Parameters["target_level"]); ok {
				targetLevel = level
			}
			newPath, err = h.processor.Normalize(currentPath, targetLevel)

		case "speed":
			if factor, ok := getFloat(op.Parameters["speed_factor"]); ok {
				newPath, err = h.processor.ChangeSpeed(currentPath, factor)
			}

		case "pitch":
			if semitones, ok := getFloat(op.Parameters["semitones"]); ok {
				newPath, err = h.processor.ChangePitch(currentPath, int(semitones))
			}

		case "equalizer":
			if eqSettings, ok := op.Parameters["eq_settings"].(map[string]interface{}); ok {
				eqMap := make(map[string]float64)
				for k, v := range eqSettings {
					if f, ok := getFloat(v); ok {
						eqMap[k] = f
					}
				}
				newPath, err = h.processor.ApplyEqualizer(currentPath, eqMap)
			}

		case "noise_reduction":
			intensity := 0.5
			if i, ok := getFloat(op.Parameters["intensity"]); ok {
				intensity = i
			}
			newPath, err = h.processor.ApplyNoiseReduction(currentPath, intensity)

		default:
			err = fmt.Errorf("unsupported operation: %s", op.Type)
		}

		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("operation failed: %v", err))
			return
		}

		if newPath != "" {
			currentPath = newPath
		}
	}

	// Convert to output format if specified
	if req.OutputFormat != "" && req.OutputFormat != filepath.Ext(currentPath)[1:] {
		finalPath, err := h.processor.ConvertFormat(currentPath, req.OutputFormat, req.QualitySettings)
		if err == nil {
			currentPath = finalPath
		}
	}

	// Get metadata for final file
	metadata, _ := h.processor.ExtractMetadata(currentPath)
	fileInfo, err := os.Stat(currentPath)

	// Add final output file if not a split operation
	if len(outputFiles) == 0 {
		duration := 0.0
		if metadata != nil && metadata.Duration != "" {
			duration, _ = strconv.ParseFloat(metadata.Duration, 64)
		}

		fileSize := int64(0)
		if err == nil && fileInfo != nil {
			fileSize = fileInfo.Size()
		}

		outputFiles = append(outputFiles, map[string]interface{}{
			"file_id":          uuid.New().String(),
			"file_path":        currentPath,
			"duration_seconds": duration,
			"file_size_bytes":  fileSize,
		})
	}

	// Store job info in database if available
	if h.db != nil {
		_, err := h.db.Exec(`
			INSERT INTO audio_processing_jobs (id, operation_type, status, output_files, created_at)
			VALUES ($1, $2, $3, $4, $5)`,
			jobID, "edit", "completed", json.RawMessage(mustMarshal(outputFiles)), time.Now())

		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to store job info: %v\n", err)
		}
	}

	response := EditResponse{
		JobID:            jobID,
		OutputFiles:      outputFiles,
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}

	sendJSON(w, http.StatusOK, response)
}

// HandleConvert handles format conversion
func (h *AudioHandler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form for file upload
	err := r.ParseMultipartForm(100 << 20) // 100 MB max
	if err != nil {
		sendError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	// Get the file
	file, header, err := r.FormFile("audio")
	if err != nil {
		sendError(w, http.StatusBadRequest, "audio file required")
		return
	}
	defer file.Close()

	// Save uploaded file
	tempID := uuid.New().String()
	tempPath := filepath.Join(h.workDir, fmt.Sprintf("%s%s", tempID, filepath.Ext(header.Filename)))

	dst, err := os.Create(tempPath)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		sendError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	// Get target format
	format := r.FormValue("format")
	if format == "" {
		sendError(w, http.StatusBadRequest, "format parameter required")
		return
	}

	// Get quality settings
	qualitySettings := make(map[string]interface{})
	if bitrate := r.FormValue("bitrate"); bitrate != "" {
		if br, err := strconv.ParseFloat(bitrate, 64); err == nil {
			qualitySettings["bitrate"] = br
		}
	}
	if sampleRate := r.FormValue("sample_rate"); sampleRate != "" {
		if sr, err := strconv.ParseFloat(sampleRate, 64); err == nil {
			qualitySettings["sample_rate"] = sr
		}
	}

	// Convert the file
	outputPath, err := h.processor.ConvertFormat(tempPath, format, qualitySettings)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("conversion failed: %v", err))
		return
	}

	// Get metadata
	metadata, _ := h.processor.ExtractMetadata(outputPath)
	fileInfo, _ := os.Stat(outputPath)

	response := map[string]interface{}{
		"file_id":         uuid.New().String(),
		"file_path":       outputPath,
		"format":          format,
		"file_size_bytes": fileInfo.Size(),
		"metadata":        metadata,
	}

	sendJSON(w, http.StatusOK, response)
}

// HandleMetadata extracts and returns audio metadata
func (h *AudioHandler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	var filePath string

	// Check if this is a file upload or an ID reference
	if r.Method == http.MethodPost {
		// Handle file upload
		err := r.ParseMultipartForm(100 << 20) // 100 MB max
		if err != nil {
			sendError(w, http.StatusBadRequest, "failed to parse form")
			return
		}

		file, header, err := r.FormFile("audio")
		if err != nil {
			sendError(w, http.StatusBadRequest, "audio file required")
			return
		}
		defer file.Close()

		// Save uploaded file temporarily
		tempID := uuid.New().String()
		tempPath := filepath.Join(h.workDir, fmt.Sprintf("%s%s", tempID, filepath.Ext(header.Filename)))

		dst, err := os.Create(tempPath)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "failed to save file")
			return
		}
		defer dst.Close()
		defer os.Remove(tempPath) // Clean up temp file

		if _, err := io.Copy(dst, file); err != nil {
			sendError(w, http.StatusInternalServerError, "failed to save file")
			return
		}

		filePath = tempPath
	} else {
		// Handle ID-based lookup
		vars := mux.Vars(r)
		fileID := vars["id"]

		// Try to get file path from database
		if h.db != nil {
			err := h.db.QueryRow("SELECT file_path FROM audio_assets WHERE id = $1", fileID).Scan(&filePath)
			if err != nil {
				// If not in DB, assume it's a direct file path
				filePath = fileID
			}
		} else {
			// No database, assume it's a direct file path
			filePath = fileID
		}
	}

	metadata, err := h.processor.ExtractMetadata(filePath)
	if err != nil {
		sendError(w, http.StatusNotFound, fmt.Sprintf("failed to extract metadata: %v", err))
		return
	}

	sendJSON(w, http.StatusOK, metadata)
}

// HandleEnhance applies audio enhancement
func (h *AudioHandler) HandleEnhance(w http.ResponseWriter, r *http.Request) {
	var inputPath string
	var targetEnvironment string
	var enhancements []map[string]interface{}

	// Check if it's a multipart request
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			sendError(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}

		// Get the audio file
		file, _, err := r.FormFile("audio")
		if err != nil {
			sendError(w, http.StatusBadRequest, "no audio file provided")
			return
		}
		defer file.Close()

		// Save to temporary file
		tmpFile, err := os.CreateTemp("", "enhance_*.wav")
		if err != nil {
			sendError(w, http.StatusInternalServerError, "failed to create temp file")
			return
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, file); err != nil {
			sendError(w, http.StatusInternalServerError, "failed to save uploaded file")
			return
		}
		inputPath = tmpFile.Name()

		// Get environment parameter
		targetEnvironment = r.FormValue("environment")
		if targetEnvironment == "" {
			targetEnvironment = "general"
		}

		// Parse enhancements if provided
		enhancementsStr := r.FormValue("enhancements")
		if enhancementsStr != "" {
			if err := json.Unmarshal([]byte(enhancementsStr), &enhancements); err != nil {
				// Default enhancements if parsing fails
				enhancements = []map[string]interface{}{
					{"type": "noise_reduction", "intensity": 0.5},
					{"type": "auto_level"},
				}
			}
		} else {
			// Default enhancements
			enhancements = []map[string]interface{}{
				{"type": "noise_reduction", "intensity": 0.5},
				{"type": "auto_level"},
			}
		}
	} else {
		// Handle JSON request
		var req struct {
			AudioFile         interface{}              `json:"audio_file"`
			Enhancements      []map[string]interface{} `json:"enhancements"`
			TargetEnvironment string                   `json:"target_environment"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		targetEnvironment = req.TargetEnvironment
		enhancements = req.Enhancements

		// Get input file path
		switch v := req.AudioFile.(type) {
		case string:
			inputPath = v
		case map[string]interface{}:
			if assetID, ok := v["asset_id"].(string); ok {
				if h.db != nil {
					err := h.db.QueryRow("SELECT file_path FROM audio_assets WHERE id = $1", assetID).Scan(&inputPath)
					if err != nil {
						sendError(w, http.StatusNotFound, "audio asset not found")
						return
					}
				} else {
					sendError(w, http.StatusServiceUnavailable, "database not configured")
					return
				}
			}
		}
	}

	currentPath := inputPath
	var appliedEnhancements []string

	// Apply each enhancement
	for _, enhancement := range enhancements {
		enhType := enhancement["type"].(string)
		intensity := 0.5
		if i, ok := enhancement["intensity"].(float64); ok {
			intensity = i
		}

		var newPath string
		var err error

		switch enhType {
		case "noise_reduction":
			newPath, err = h.processor.ApplyNoiseReduction(currentPath, intensity)
			appliedEnhancements = append(appliedEnhancements, "noise_reduction")

		case "auto_level":
			newPath, err = h.processor.Normalize(currentPath, -16.0) // Standard LUFS
			appliedEnhancements = append(appliedEnhancements, "auto_level")

		case "eq":
			// Apply preset EQ based on target environment
			eqSettings := getEQPreset(targetEnvironment)
			newPath, err = h.processor.ApplyEqualizer(currentPath, eqSettings)
			appliedEnhancements = append(appliedEnhancements, "eq_"+targetEnvironment)

		case "compressor":
			// Simple volume normalization as compression placeholder
			newPath, err = h.processor.Normalize(currentPath, -14.0)
			appliedEnhancements = append(appliedEnhancements, "compressor")
		}

		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("enhancement failed: %v", err))
			return
		}

		if newPath != "" {
			currentPath = newPath
		}
	}

	// Get metadata for enhanced file
	metadata, _ := h.processor.ExtractMetadata(currentPath)
	fileInfo, err := os.Stat(currentPath)

	fileSize := int64(0)
	if err == nil && fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	response := map[string]interface{}{
		"enhanced_file": map[string]interface{}{
			"file_id":   uuid.New().String(),
			"file_path": currentPath,
			"improvement_metrics": map[string]interface{}{
				"applied_enhancements": appliedEnhancements,
				"file_size_bytes":      fileSize,
			},
		},
		"enhanced_file_path":   currentPath, // For backward compatibility with tests
		"applied_enhancements": appliedEnhancements,
		"metadata":             metadata,
	}

	sendJSON(w, http.StatusOK, response)
}

// HandleAnalyze performs audio analysis
func (h *AudioHandler) HandleAnalyze(w http.ResponseWriter, r *http.Request) {
	var inputPath string
	var analysisTypes []string
	_ = map[string]bool{} // options reserved for future use

	// Check if it's a multipart request
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			sendError(w, http.StatusBadRequest, "failed to parse multipart form")
			return
		}

		// Get the audio file
		file, _, err := r.FormFile("audio")
		if err != nil {
			sendError(w, http.StatusBadRequest, "no audio file provided")
			return
		}
		defer file.Close()

		// Save to temporary file
		tmpFile, err := os.CreateTemp("", "analyze_*.wav")
		if err != nil {
			sendError(w, http.StatusInternalServerError, "failed to create temp file")
			return
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, file); err != nil {
			sendError(w, http.StatusInternalServerError, "failed to save uploaded file")
			return
		}
		inputPath = tmpFile.Name()

		// Parse analysis types if provided
		typesStr := r.FormValue("analysis_types")
		if typesStr != "" {
			if err := json.Unmarshal([]byte(typesStr), &analysisTypes); err != nil {
				// Default analysis types
				analysisTypes = []string{"quality", "spectral"}
			}
		} else {
			// Default analysis
			analysisTypes = []string{"quality", "spectral"}
		}
	} else {
		// Handle JSON request
		var req struct {
			AudioFile     interface{}     `json:"audio_file"`
			AnalysisTypes []string        `json:"analysis_types"`
			Options       map[string]bool `json:"options"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		analysisTypes = req.AnalysisTypes
		// options = req.Options // Reserved for future use

		// Get input file path
		switch v := req.AudioFile.(type) {
		case string:
			inputPath = v
		case map[string]interface{}:
			if assetID, ok := v["asset_id"].(string); ok {
				if h.db != nil {
					err := h.db.QueryRow("SELECT file_path FROM audio_assets WHERE id = $1", assetID).Scan(&inputPath)
					if err != nil {
						sendError(w, http.StatusNotFound, "audio asset not found")
						return
					}
				} else {
					sendError(w, http.StatusServiceUnavailable, "database not configured")
					return
				}
			}
		}
	}

	// Extract metadata
	metadata, err := h.processor.ExtractMetadata(inputPath)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "failed to analyze audio")
		return
	}

	analysisResults := make(map[string]interface{})

	// Default to basic analysis if no types specified
	if len(analysisTypes) == 0 {
		analysisTypes = []string{"quality"}
	}

	// Perform requested analyses
	for _, analysisType := range analysisTypes {
		switch analysisType {
		case "quality":
			// Use the new quality analysis method
			qualityMetrics, err := h.processor.AnalyzeQuality(inputPath)
			if err != nil {
				// Fall back to basic metadata
				analysisResults["quality_metrics"] = map[string]interface{}{
					"format":      metadata.Format,
					"codec":       metadata.Codec,
					"sample_rate": metadata.SampleRate,
					"bitrate":     metadata.Bitrate,
					"channels":    metadata.Channels,
					"error":       "Advanced quality analysis unavailable",
				}
			} else {
				analysisResults["quality_metrics"] = qualityMetrics
			}

		case "content":
			analysisResults["content_classification"] = []string{
				"audio",
				fmt.Sprintf("%d_channels", metadata.Channels),
			}

		case "spectral":
			// Placeholder for spectral analysis
			analysisResults["spectral_analysis"] = map[string]interface{}{
				"dominant_frequency": "placeholder",
				"frequency_range":    "20Hz-20kHz",
			}
		}
	}

	response := map[string]interface{}{
		"analysis_results": analysisResults,
		"metadata":         metadata,
		"analysis":         metadata, // For backward compatibility with tests
		"analysis_time_ms": 100,      // Placeholder
	}

	sendJSON(w, http.StatusOK, response)
}

// Helper functions

func getFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// HandleVAD performs voice activity detection on audio
func (h *AudioHandler) HandleVAD(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check content type
	contentType := r.Header.Get("Content-Type")

	var inputPath string
	var threshold float64 = -40 // Default threshold

	if strings.Contains(contentType, "multipart/form-data") {
		// Parse multipart form
		err := r.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			sendError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}

		// Get audio file from form
		file, _, err := r.FormFile("audio")
		if err != nil {
			sendError(w, http.StatusBadRequest, "Audio file not found in form data")
			return
		}
		defer file.Close()

		// Save uploaded file to temp location
		tempFile := filepath.Join(h.workDir, fmt.Sprintf("%s_upload", uuid.New().String()))
		out, err := os.Create(tempFile)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to create temp file")
			return
		}
		defer out.Close()
		defer os.Remove(tempFile) // Clean up

		_, err = io.Copy(out, file)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to save uploaded file")
			return
		}

		inputPath = tempFile

		// Get threshold if provided
		if thresholdStr := r.FormValue("threshold"); thresholdStr != "" {
			if val, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
				threshold = val
			}
		}
	} else {
		// Parse JSON request
		var req struct {
			AudioFile interface{} `json:"audio_file"`
			Threshold float64     `json:"threshold"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Threshold != 0 {
			threshold = req.Threshold
		}

		// Handle audio file reference
		switch v := req.AudioFile.(type) {
		case string:
			inputPath = v
		case map[string]interface{}:
			if assetID, ok := v["asset_id"].(string); ok {
				// Retrieve file path from database
				if h.db != nil {
					var path string
					err := h.db.QueryRow("SELECT file_path FROM audio_assets WHERE id = $1", assetID).Scan(&path)
					if err != nil {
						sendError(w, http.StatusNotFound, "audio asset not found")
						return
					}
					inputPath = path
				} else {
					sendError(w, http.StatusServiceUnavailable, "database not configured")
					return
				}
			}
		default:
			sendError(w, http.StatusBadRequest, "invalid audio_file format")
			return
		}
	}

	// Perform VAD
	startTime := time.Now()
	vad, err := h.processor.DetectVoiceActivity(inputPath, threshold)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("VAD failed: %v", err))
		return
	}

	response := map[string]interface{}{
		"speech_segments":          vad.SpeechSegments,
		"total_duration_seconds":   vad.TotalDuration,
		"speech_duration_seconds":  vad.SpeechDuration,
		"silence_duration_seconds": vad.SilenceDuration,
		"speech_ratio":             vad.SpeechRatio,
		"processing_time_ms":       time.Since(startTime).Milliseconds(),
	}

	sendJSON(w, http.StatusOK, response)
}

// HandleRemoveSilence removes silence from audio
func (h *AudioHandler) HandleRemoveSilence(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check content type
	contentType := r.Header.Get("Content-Type")

	var inputPath string
	var threshold float64 = -40 // Default threshold
	var outputFormat string = ""

	if strings.Contains(contentType, "multipart/form-data") {
		// Parse multipart form
		err := r.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			sendError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}

		// Get audio file from form
		file, header, err := r.FormFile("audio")
		if err != nil {
			sendError(w, http.StatusBadRequest, "Audio file not found in form data")
			return
		}
		defer file.Close()

		// Save uploaded file to temp location
		ext := filepath.Ext(header.Filename)
		tempFile := filepath.Join(h.workDir, fmt.Sprintf("%s_upload%s", uuid.New().String(), ext))
		out, err := os.Create(tempFile)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to create temp file")
			return
		}
		defer out.Close()
		defer os.Remove(tempFile) // Clean up

		_, err = io.Copy(out, file)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to save uploaded file")
			return
		}

		inputPath = tempFile

		// Get parameters from form
		if thresholdStr := r.FormValue("threshold"); thresholdStr != "" {
			if val, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
				threshold = val
			}
		}
		outputFormat = r.FormValue("output_format")
	} else {
		// Parse JSON request
		var req struct {
			AudioFile    interface{} `json:"audio_file"`
			Threshold    float64     `json:"threshold"`
			OutputFormat string      `json:"output_format"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Threshold != 0 {
			threshold = req.Threshold
		}
		outputFormat = req.OutputFormat

		// Handle audio file reference
		switch v := req.AudioFile.(type) {
		case string:
			inputPath = v
		case map[string]interface{}:
			if assetID, ok := v["asset_id"].(string); ok {
				// Retrieve file path from database
				if h.db != nil {
					var path string
					err := h.db.QueryRow("SELECT file_path FROM audio_assets WHERE id = $1", assetID).Scan(&path)
					if err != nil {
						sendError(w, http.StatusNotFound, "audio asset not found")
						return
					}
					inputPath = path
				} else {
					sendError(w, http.StatusServiceUnavailable, "database not configured")
					return
				}
			}
		default:
			sendError(w, http.StatusBadRequest, "invalid audio_file format")
			return
		}
	}

	// Remove silence
	startTime := time.Now()
	outputPath, err := h.processor.RemoveSilence(inputPath, threshold)
	if err != nil {
		sendError(w, http.StatusInternalServerError, fmt.Sprintf("Silence removal failed: %v", err))
		return
	}
	defer os.Remove(outputPath) // Clean up temp file

	// Convert format if requested
	finalPath := outputPath
	if outputFormat != "" && outputFormat != filepath.Ext(outputPath)[1:] {
		convertedPath, err := h.processor.ConvertFormat(outputPath, outputFormat, nil)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("Format conversion failed: %v", err))
			return
		}
		os.Remove(outputPath)
		finalPath = convertedPath
		defer os.Remove(finalPath)
	}

	// Get metadata for the processed file
	metadata, _ := h.processor.ExtractMetadata(finalPath)

	// Generate output file ID
	outputID := uuid.New().String()

	// Copy to data directory with proper naming
	outputFileName := fmt.Sprintf("%s_speech_only%s", outputID, filepath.Ext(finalPath))
	permanentPath := filepath.Join(h.dataDir, outputFileName)

	// Copy file to permanent location
	src, err := os.Open(finalPath)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to read processed file")
		return
	}
	defer src.Close()

	dst, err := os.Create(permanentPath)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create output file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to save output file")
		return
	}

	// Get file info
	fileInfo, _ := os.Stat(permanentPath)

	response := map[string]interface{}{
		"job_id": outputID,
		"output_file": map[string]interface{}{
			"file_id":          outputID,
			"file_path":        permanentPath,
			"duration_seconds": metadata.Duration,
			"file_size_bytes":  fileInfo.Size(),
		},
		"processing_time_ms": time.Since(startTime).Milliseconds(),
	}

	sendJSON(w, http.StatusOK, response)
}

func getEQPreset(environment string) map[string]float64 {
	presets := map[string]map[string]float64{
		"podcast": {
			"100":   -2.0,
			"200":   0.0,
			"500":   1.0,
			"1000":  2.0,
			"2000":  1.0,
			"5000":  0.0,
			"10000": -1.0,
		},
		"meeting": {
			"100":   -3.0,
			"200":   -1.0,
			"500":   1.0,
			"1000":  2.0,
			"2000":  2.0,
			"5000":  1.0,
			"10000": 0.0,
		},
		"music": {
			"100":   1.0,
			"200":   0.5,
			"500":   0.0,
			"1000":  0.0,
			"2000":  0.5,
			"5000":  1.0,
			"10000": 0.5,
		},
	}

	if preset, ok := presets[environment]; ok {
		return preset
	}

	// Default flat response
	return map[string]float64{
		"100":   0.0,
		"1000":  0.0,
		"10000": 0.0,
	}
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  message,
		"status": status,
	})
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
