package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"
)

// ProcessingJob represents a file processing job
type ProcessingJob struct {
	FileID     string
	JobType    string
	Payload    interface{}
	RetryCount int
	MaxRetries int
}

// WorkerPool manages concurrent processing workers
type WorkerPool struct {
	workers int
	jobs    chan ProcessingJob
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// Start worker pool for processing
func (app *App) startWorkers() {
	for i := 0; i < app.WorkerPool.workers; i++ {
		app.WorkerPool.wg.Add(1)
		go app.worker(i)
	}

	// Start a goroutine to check for pending jobs in Redis
	go app.processPendingJobs()
}

// Worker processes jobs from the queue
func (app *App) worker(id int) {
	defer app.WorkerPool.wg.Done()

	for {
		select {
		case <-app.WorkerPool.ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		case job := <-app.ProcessingQueue:
			log.Printf("Worker %d processing job: %s", id, job.FileID)
			app.processJob(job)
		}
	}
}

// Process a single job
func (app *App) processJob(job ProcessingJob) {
	switch job.JobType {
	case "file_process":
		app.processFileJob(job)
	case "duplicate_check":
		app.checkDuplicatesJob(job)
	case "generate_embeddings":
		app.generateEmbeddingsJob(job)
	case "extract_text":
		app.extractTextJob(job)
	case "analyze_image":
		app.analyzeImageJob(job)
	case "organize":
		app.organizeFileJob(job)
	default:
		log.Printf("Unknown job type: %s", job.JobType)
	}
}

// Process file job - main processing pipeline
func (app *App) processFileJob(job ProcessingJob) {
	req := job.Payload.(UploadRequest)
	
	// Update status to processing
	app.updateFileStatus(job.FileID, "processing", "analyzing")

	// Determine file type and route to appropriate processor
	fileType := app.determineFileType(req.MimeType, req.Filename)
	
	switch fileType {
	case "image":
		app.processImage(job.FileID, req)
	case "document":
		app.processDocument(job.FileID, req)
	case "video":
		app.processVideo(job.FileID, req)
	default:
		app.processGenericFile(job.FileID, req)
	}

	// Check for duplicates
	app.queueJob(ProcessingJob{
		FileID:  job.FileID,
		JobType: "duplicate_check",
		Payload: req,
	})

	// Generate embeddings for semantic search
	app.queueJob(ProcessingJob{
		FileID:  job.FileID,
		JobType: "generate_embeddings",
		Payload: req,
	})

	// Update status to completed
	app.updateFileStatus(job.FileID, "completed", "ready")
}

// Process image files
func (app *App) processImage(fileID string, req UploadRequest) {
	log.Printf("Processing image: %s", fileID)
	
	// Analyze image with Ollama vision model
	app.analyzeImageWithAI(fileID, req.StoragePath)
	
	// Generate thumbnail
	app.generateThumbnail(fileID, req.StoragePath)
	
	// Extract metadata
	app.extractImageMetadata(fileID, req.StoragePath)
}

// Process document files
func (app *App) processDocument(fileID string, req UploadRequest) {
	log.Printf("Processing document: %s", fileID)
	
	// Extract text content
	app.extractDocumentText(fileID, req.StoragePath)
	
	// Generate summary with AI
	app.generateDocumentSummary(fileID)
}

// Process video files
func (app *App) processVideo(fileID string, req UploadRequest) {
	log.Printf("Processing video: %s", fileID)
	
	// Extract key frames
	app.extractVideoFrames(fileID, req.StoragePath)
	
	// Generate thumbnail from first frame
	app.generateVideoThumbnail(fileID, req.StoragePath)
}

// Process generic files
func (app *App) processGenericFile(fileID string, req UploadRequest) {
	log.Printf("Processing generic file: %s", fileID)
	
	// Extract basic metadata
	app.extractFileMetadata(fileID, req.StoragePath)
}

// Organize file based on strategy
func (app *App) organizeFileJob(job ProcessingJob) {
	strategy := job.Payload.(string)
	
	switch strategy {
	case "by_type":
		app.organizeByType(job.FileID)
	case "by_date":
		app.organizeByDate(job.FileID)
	case "by_content":
		app.organizeByContent(job.FileID)
	case "smart":
		app.organizeSmartAI(job.FileID)
	default:
		app.organizeSmartAI(job.FileID)
	}
}

// Queue job for processing
func (app *App) queueJob(job ProcessingJob) {
	select {
	case app.ProcessingQueue <- job:
		// Job queued
	default:
		// Store in Redis for later
		ctx := context.Background()
		jobData, _ := json.Marshal(job)
		app.RedisClient.LPush(ctx, "pending_jobs", jobData)
	}
}

// Queue file for processing
func (app *App) queueFileProcessing(fileID string, req UploadRequest) {
	job := ProcessingJob{
		FileID:     fileID,
		JobType:    "file_process",
		Payload:    req,
		RetryCount: 0,
		MaxRetries: 3,
	}

	select {
	case app.ProcessingQueue <- job:
		log.Printf("File %s queued for processing", fileID)
	default:
		log.Printf("Processing queue full, file %s will be processed later", fileID)
		// Store in Redis for later processing
		ctx := context.Background()
		jobData, _ := json.Marshal(job)
		app.RedisClient.LPush(ctx, "pending_jobs", jobData)
	}
}

// Process pending jobs from Redis
func (app *App) processPendingJobs() {
	ctx := context.Background()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-app.WorkerPool.ctx.Done():
			return
		case <-ticker.C:
			// Try to get pending jobs from Redis
			for {
				result, err := app.RedisClient.RPop(ctx, "pending_jobs").Result()
				if err != nil {
					break // No more pending jobs
				}
				
				var job ProcessingJob
				if err := json.Unmarshal([]byte(result), &job); err == nil {
					select {
					case app.ProcessingQueue <- job:
						log.Printf("Queued pending job: %s", job.FileID)
					default:
						// Queue is full, put it back
						app.RedisClient.LPush(ctx, "pending_jobs", result)
						break
					}
				}
			}
		}
	}
}

// Helper function to determine file type
func (app *App) determineFileType(mimeType, filename string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "application/pdf") ||
	   strings.HasPrefix(mimeType, "application/msword") ||
	   strings.HasPrefix(mimeType, "application/vnd.") ||
	   strings.HasPrefix(mimeType, "text/") {
		return "document"
	}
	return "generic"
}

// Update file status in database
func (app *App) updateFileStatus(fileID, status, stage string) {
	query := `UPDATE files SET status = $1, processing_stage = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`
	app.DB.Exec(query, status, stage, fileID)
}

// Update file description
func (app *App) updateFileDescription(fileID, description string) {
	query := `UPDATE files SET description = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	app.DB.Exec(query, description, fileID)
}

// Update detected objects
func (app *App) updateFileObjects(fileID string, objects []string) {
	objectsJSON, _ := json.Marshal(objects)
	query := `UPDATE files SET detected_objects = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	app.DB.Exec(query, objectsJSON, fileID)
}

// Extract objects from AI description
func (app *App) extractObjectsFromDescription(description string) []string {
	// Simple extraction - in production, use NLP
	objects := []string{}
	keywords := []string{"person", "car", "building", "tree", "animal", "food", "text", "sign"}
	
	descLower := strings.ToLower(description)
	for _, keyword := range keywords {
		if strings.Contains(descLower, keyword) {
			objects = append(objects, keyword)
		}
	}
	return objects
}