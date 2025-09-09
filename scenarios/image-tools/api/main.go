package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
	
	"image-tools/plugins"
	"image-tools/plugins/jpeg"
	"image-tools/plugins/png"
	"image-tools/plugins/webp"
	"image-tools/plugins/svg"
)

type Server struct {
	app      *fiber.App
	registry *plugins.PluginRegistry
	storage  StorageService
}

type StorageService interface {
	Save(key string, data []byte) (string, error)
	Get(key string) ([]byte, error)
	Delete(key string) error
}

type CompressRequest struct {
	Quality int    `json:"quality"`
	Format  string `json:"format,omitempty"`
}

type ResizeRequest struct {
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	MaintainAspect bool   `json:"maintain_aspect"`
	Algorithm      string `json:"algorithm,omitempty"`
}

type ConvertRequest struct {
	TargetFormat string                 `json:"target_format"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

type BatchRequest struct {
	Operations []Operation `json:"operations"`
}

type Operation struct {
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options"`
}

func NewServer() *Server {
	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024, // 100MB
	})
	
	app.Use(logger.New())
	app.Use(cors.New())
	
	registry := plugins.NewRegistry()
	registry.Register(jpeg.New())
	registry.Register(png.New())
	registry.Register(webp.New())
	registry.Register(svg.New())
	
	return &Server{
		app:      app,
		registry: registry,
	}
}

func (s *Server) SetupRoutes() {
	api := s.app.Group("/api/v1")
	
	api.Get("/health", s.handleHealth)
	api.Get("/plugins", s.handleListPlugins)
	
	image := api.Group("/image")
	image.Post("/compress", s.handleCompress)
	image.Post("/resize", s.handleResize)
	image.Post("/convert", s.handleConvert)
	image.Post("/metadata", s.handleMetadata)
	image.Post("/batch", s.handleBatch)
	image.Post("/info", s.handleInfo)
}

func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
		"plugins": len(s.registry.ListPlugins()),
	})
}

func (s *Server) handleListPlugins(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"plugins": s.registry.ListPlugins(),
	})
}

func (s *Server) handleCompress(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No image provided"})
	}
	
	var req CompressRequest
	if err := c.BodyParser(&req); err != nil {
		req.Quality = 85
	}
	
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer src.Close()
	
	format := getFileFormat(file.Filename)
	plugin, ok := s.registry.GetPlugin(format)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("Unsupported format: %s", format)})
	}
	
	options := plugins.ProcessOptions{
		Quality: req.Quality,
	}
	
	result, err := plugin.Compress(src, options)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	
	outputKey := fmt.Sprintf("compressed/%s.%s", uuid.New().String(), format)
	url, err := s.saveToStorage(outputKey, result.OutputData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save compressed image"})
	}
	
	return c.JSON(fiber.Map{
		"url":             url,
		"original_size":   result.OriginalSize,
		"compressed_size": result.ProcessedSize,
		"savings_percent": result.SavingsPercent,
		"format":          format,
	})
}

func (s *Server) handleResize(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No image provided"})
	}
	
	var req ResizeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid resize parameters"})
	}
	
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer src.Close()
	
	format := getFileFormat(file.Filename)
	plugin, ok := s.registry.GetPlugin(format)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("Unsupported format: %s", format)})
	}
	
	options := plugins.ProcessOptions{
		Width:          req.Width,
		Height:         req.Height,
		MaintainAspect: req.MaintainAspect,
		Algorithm:      req.Algorithm,
	}
	
	result, err := plugin.Resize(src, options)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	
	outputKey := fmt.Sprintf("resized/%s.%s", uuid.New().String(), format)
	url, err := s.saveToStorage(outputKey, result.OutputData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save resized image"})
	}
	
	return c.JSON(fiber.Map{
		"url": url,
		"dimensions": fiber.Map{
			"width":  result.OutputInfo.Width,
			"height": result.OutputInfo.Height,
		},
		"size": result.ProcessedSize,
	})
}

func (s *Server) handleConvert(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No image provided"})
	}
	
	var req ConvertRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid convert parameters"})
	}
	
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer src.Close()
	
	sourceFormat := getFileFormat(file.Filename)
	
	if sourceFormat == req.TargetFormat {
		return c.Status(400).JSON(fiber.Map{"error": "Source and target formats are the same"})
	}
	
	targetPlugin, ok := s.registry.GetPlugin(req.TargetFormat)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("Unsupported target format: %s", req.TargetFormat)})
	}
	
	options := plugins.ProcessOptions{
		CustomOptions: req.Options,
	}
	
	result, err := targetPlugin.Convert(src, req.TargetFormat, options)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	
	outputKey := fmt.Sprintf("converted/%s.%s", uuid.New().String(), req.TargetFormat)
	url, err := s.saveToStorage(outputKey, result.OutputData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save converted image"})
	}
	
	return c.JSON(fiber.Map{
		"url":    url,
		"format": req.TargetFormat,
		"size":   result.ProcessedSize,
	})
}

func (s *Server) handleMetadata(c *fiber.Ctx) error {
	action := c.Query("action", "strip")
	
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No image provided"})
	}
	
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer src.Close()
	
	format := getFileFormat(file.Filename)
	plugin, ok := s.registry.GetPlugin(format)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("Unsupported format: %s", format)})
	}
	
	if action == "read" {
		info, err := plugin.GetInfo(src)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(info)
	}
	
	result, err := plugin.StripMetadata(src)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	
	outputKey := fmt.Sprintf("stripped/%s.%s", uuid.New().String(), format)
	url, err := s.saveToStorage(outputKey, result.OutputData)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save stripped image"})
	}
	
	return c.JSON(fiber.Map{
		"url":           url,
		"size":          result.ProcessedSize,
		"metadata_removed": true,
	})
}

func (s *Server) handleBatch(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse multipart form"})
	}
	
	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "No images provided"})
	}
	
	var req BatchRequest
	if err := json.Unmarshal([]byte(c.FormValue("operations")), &req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid batch operations"})
	}
	
	results := make([]fiber.Map, 0, len(files))
	totalSavings := int64(0)
	
	for _, file := range files {
		result, err := s.processSingleImage(file, req.Operations)
		if err != nil {
			results = append(results, fiber.Map{
				"filename": file.Filename,
				"error":    err.Error(),
			})
			continue
		}
		
		results = append(results, result)
		if savings, ok := result["savings_bytes"].(int64); ok {
			totalSavings += savings
		}
	}
	
	return c.JSON(fiber.Map{
		"results":       results,
		"total_savings": totalSavings,
		"processed":     len(results),
	})
}

func (s *Server) handleInfo(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No image provided"})
	}
	
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer src.Close()
	
	format := getFileFormat(file.Filename)
	plugin, ok := s.registry.GetPlugin(format)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("Unsupported format: %s", format)})
	}
	
	info, err := plugin.GetInfo(src)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	
	return c.JSON(info)
}

func (s *Server) processSingleImage(file *multipart.FileHeader, operations []Operation) (fiber.Map, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	
	originalSize := int64(len(data))
	currentData := data
	format := getFileFormat(file.Filename)
	
	for _, op := range operations {
		reader := bytes.NewReader(currentData)
		plugin, ok := s.registry.GetPlugin(format)
		if !ok {
			return nil, fmt.Errorf("unsupported format: %s", format)
		}
		
		var result *plugins.ProcessResult
		
		switch op.Type {
		case "compress":
			quality := 85
			if q, ok := op.Options["quality"].(float64); ok {
				quality = int(q)
			}
			result, err = plugin.Compress(reader, plugins.ProcessOptions{Quality: quality})
			
		case "resize":
			width := int(op.Options["width"].(float64))
			height := int(op.Options["height"].(float64))
			maintainAspect := false
			if ma, ok := op.Options["maintain_aspect"].(bool); ok {
				maintainAspect = ma
			}
			result, err = plugin.Resize(reader, plugins.ProcessOptions{
				Width:          width,
				Height:         height,
				MaintainAspect: maintainAspect,
			})
			
		case "strip_metadata":
			result, err = plugin.StripMetadata(reader)
			
		default:
			return nil, fmt.Errorf("unknown operation: %s", op.Type)
		}
		
		if err != nil {
			return nil, err
		}
		
		currentData = result.OutputData
	}
	
	outputKey := fmt.Sprintf("batch/%s/%s", uuid.New().String(), file.Filename)
	url, err := s.saveToStorage(outputKey, currentData)
	if err != nil {
		return nil, err
	}
	
	processedSize := int64(len(currentData))
	
	return fiber.Map{
		"filename":      file.Filename,
		"url":           url,
		"original_size": originalSize,
		"processed_size": processedSize,
		"savings_bytes": originalSize - processedSize,
		"savings_percent": float64(originalSize - processedSize) / float64(originalSize) * 100,
	}, nil
}

func (s *Server) saveToStorage(key string, data []byte) (string, error) {
	if s.storage != nil {
		return s.storage.Save(key, data)
	}
	
	dir := "/tmp/image-tools"
	os.MkdirAll(dir, 0755)
	
	path := filepath.Join(dir, key)
	os.MkdirAll(filepath.Dir(path), 0755)
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("file://%s", path), nil
}

func getFileFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	ext = strings.TrimPrefix(ext, ".")
	
	if ext == "jpg" {
		return "jpeg"
	}
	
	return ext
}

func main() {
	server := NewServer()
	server.SetupRoutes()
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Image Tools API starting on port %s", port)
	log.Fatal(server.app.Listen(":" + port))
}