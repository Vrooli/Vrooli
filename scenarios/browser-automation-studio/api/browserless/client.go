package browserless

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
)

// Client wraps resource-browserless CLI for browser automation
type Client struct {
	log         *logrus.Logger
	repo        database.Repository
	storage     *storage.MinIOClient
	browserless string // browserless service URL
}

// NodeProcessor defines how to execute different workflow node types
type NodeProcessor interface {
	ProcessNode(ctx context.Context, node map[string]interface{}, session *BrowserSession) error
}

// BrowserSession represents an active browser session
type BrowserSession struct {
	ID          string
	ExecutionID uuid.UUID
	URL         string
	Created     time.Time
}

// NewClient creates a new browserless client
func NewClient(log *logrus.Logger, repo database.Repository) *Client {
	browserlessURL := os.Getenv("BROWSERLESS_URL")
	if browserlessURL == "" {
		// Try to get from resource env vars
		if port := os.Getenv("BROWSERLESS_PORT"); port != "" {
			host := os.Getenv("BROWSERLESS_HOST")
			if host == "" {
				host = "localhost"
			}
			browserlessURL = fmt.Sprintf("http://%s:%s", host, port)
		} else {
			log.Fatal("❌ BROWSERLESS_URL or BROWSERLESS_PORT environment variable is required")
		}
	}

	// Initialize MinIO client - log error but don't fail if MinIO is not available
	storageClient, err := storage.NewMinIOClient(log)
	if err != nil {
		log.WithError(err).Warn("Failed to initialize MinIO client - screenshot storage will be disabled")
	}

	return &Client{
		log:         log,
		repo:        repo,
		storage:     storageClient,
		browserless: browserlessURL,
	}
}

// ExecuteWorkflow executes a complete workflow using browserless
func (c *Client) ExecuteWorkflow(ctx context.Context, execution *database.Execution, workflow *database.Workflow) error {
	c.log.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"workflow_id":  workflow.ID,
	}).Info("Starting browserless workflow execution")

	// Parse flow definition
	nodes, ok := workflow.FlowDefinition["nodes"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid workflow nodes")
	}

	// Parse edges for future use (e.g., following proper execution order)
	_, ok = workflow.FlowDefinition["edges"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid workflow edges")
	}

	// Create browser session
	session := &BrowserSession{
		ID:          uuid.New().String(),
		ExecutionID: execution.ID,
		Created:     time.Now(),
	}

	// Process nodes in order (simplified - real implementation would follow edges)
	for i, nodeData := range nodes {
		node, ok := nodeData.(map[string]interface{})
		if !ok {
			continue
		}

		nodeType, _ := node["type"].(string)
		c.log.WithFields(logrus.Fields{
			"node_type": nodeType,
			"node_id":   node["id"],
			"step":      i + 1,
		}).Info("Processing workflow node")

		// Update execution progress
		execution.CurrentStep = fmt.Sprintf("Processing %s node", nodeType)
		execution.Progress = (i + 1) * 100 / len(nodes)

		if err := c.repo.UpdateExecution(ctx, execution); err != nil {
			c.log.WithError(err).Error("Failed to update execution progress")
		}

		// Process node based on type
		if err := c.processNode(ctx, node, session); err != nil {
			c.log.WithError(err).WithField("node_type", nodeType).Error("Failed to process node")
			
			// Log error
			logEntry := &database.ExecutionLog{
				ExecutionID: execution.ID,
				Level:       "error",
				StepName:    nodeType,
				Message:     fmt.Sprintf("Failed to execute %s: %s", nodeType, err.Error()),
			}
			c.repo.CreateExecutionLog(ctx, logEntry)
			
			return fmt.Errorf("node execution failed: %w", err)
		}

		// Log success
		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "info",
			StepName:    nodeType,
			Message:     fmt.Sprintf("Successfully executed %s node", nodeType),
		}
		c.repo.CreateExecutionLog(ctx, logEntry)
	}

	c.log.WithField("execution_id", execution.ID).Info("Browserless workflow execution completed")
	return nil
}

// processNode processes a single workflow node
func (c *Client) processNode(ctx context.Context, node map[string]interface{}, session *BrowserSession) error {
	nodeType, _ := node["type"].(string)
	data, _ := node["data"].(map[string]interface{})

	switch nodeType {
	case "navigate":
		return c.processNavigateNode(ctx, data, session)
	case "click":
		return c.processClickNode(ctx, data, session)
	case "type":
		return c.processTypeNode(ctx, data, session)
	case "screenshot":
		return c.processScreenshotNode(ctx, data, session)
	case "wait":
		return c.processWaitNode(ctx, data, session)
	case "extract":
		return c.processExtractNode(ctx, data, session)
	default:
		return fmt.Errorf("unknown node type: %s", nodeType)
	}
}

// processNavigateNode handles navigation
func (c *Client) processNavigateNode(ctx context.Context, data map[string]interface{}, session *BrowserSession) error {
	url, ok := data["url"].(string)
	if !ok {
		return fmt.Errorf("navigate node missing url")
	}

	session.URL = url
	c.log.WithField("url", url).Info("Navigating to URL")

	// TODO: Use resource-browserless CLI to navigate
	// For now, simulate with a basic check
	cmd := exec.CommandContext(ctx, "curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", url)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	statusCode := strings.TrimSpace(string(output))
	if !strings.HasPrefix(statusCode, "2") && !strings.HasPrefix(statusCode, "3") {
		return fmt.Errorf("navigation failed with status code: %s", statusCode)
	}

	return nil
}

// processClickNode handles clicking elements
func (c *Client) processClickNode(ctx context.Context, data map[string]interface{}, session *BrowserSession) error {
	selector, ok := data["selector"].(string)
	if !ok {
		return fmt.Errorf("click node missing selector")
	}

	c.log.WithField("selector", selector).Info("Clicking element")

	// TODO: Implement with resource-browserless CLI
	// This would execute a command like: resource-browserless click <selector>
	c.log.WithField("selector", selector).Info("Simulated click action")

	return nil
}

// processTypeNode handles typing text into inputs
func (c *Client) processTypeNode(ctx context.Context, data map[string]interface{}, session *BrowserSession) error {
	selector, ok := data["selector"].(string)
	if !ok {
		return fmt.Errorf("type node missing selector")
	}

	text, ok := data["text"].(string)
	if !ok {
		return fmt.Errorf("type node missing text")
	}

	c.log.WithFields(logrus.Fields{
		"selector": selector,
		"text":     text,
	}).Info("Typing text into element")

	// TODO: Implement with resource-browserless CLI
	// This would execute: resource-browserless type <selector> <text>
	c.log.WithFields(logrus.Fields{
		"selector": selector,
		"text":     text,
	}).Info("Simulated type action")

	return nil
}

// processScreenshotNode captures screenshots
func (c *Client) processScreenshotNode(ctx context.Context, data map[string]interface{}, session *BrowserSession) error {
	name, _ := data["name"].(string)
	if name == "" {
		name = fmt.Sprintf("screenshot_%d", time.Now().Unix())
	}

	c.log.WithField("name", name).Info("Capturing screenshot")

	// TODO: Implement actual screenshot capture with resource-browserless CLI
	// For now, simulate screenshot capture and create mock image data
	mockImageData := []byte("mock screenshot data") // This would be actual PNG data

	var screenshotInfo *storage.ScreenshotInfo
	var err error

	// Store in MinIO if available
	if c.storage != nil {
		screenshotInfo, err = c.storage.StoreScreenshot(ctx, session.ExecutionID, name, mockImageData, "image/png")
		if err != nil {
			c.log.WithError(err).Error("Failed to store screenshot in MinIO")
			// Fall back to mock URLs
			screenshotInfo = &storage.ScreenshotInfo{
				URL:          fmt.Sprintf("/screenshots/%s/%s.png", session.ExecutionID, name),
				ThumbnailURL: fmt.Sprintf("/screenshots/%s/%s_thumb.png", session.ExecutionID, name),
				SizeBytes:    int64(len(mockImageData)),
				Width:        1920,
				Height:       1080,
			}
		}
	} else {
		// MinIO not available, use mock URLs
		screenshotInfo = &storage.ScreenshotInfo{
			URL:          fmt.Sprintf("/screenshots/%s/%s.png", session.ExecutionID, name),
			ThumbnailURL: fmt.Sprintf("/screenshots/%s/%s_thumb.png", session.ExecutionID, name),
			SizeBytes:    int64(len(mockImageData)),
			Width:        1920,
			Height:       1080,
		}
	}

	// Create database record
	screenshot := &database.Screenshot{
		ID:           uuid.New(),
		ExecutionID:  session.ExecutionID,
		StepName:     name,
		Timestamp:    time.Now(),
		StorageURL:   screenshotInfo.URL,
		ThumbnailURL: screenshotInfo.ThumbnailURL,
		Width:        screenshotInfo.Width,
		Height:       screenshotInfo.Height,
		SizeBytes:    screenshotInfo.SizeBytes,
	}

	return c.repo.CreateScreenshot(ctx, screenshot)
}

// processWaitNode handles waiting for conditions
func (c *Client) processWaitNode(ctx context.Context, data map[string]interface{}, session *BrowserSession) error {
	waitType, _ := data["type"].(string)
	
	switch waitType {
	case "time":
		duration, ok := data["duration"].(float64)
		if !ok {
			duration = 1000 // Default 1 second
		}
		time.Sleep(time.Duration(duration) * time.Millisecond)
		c.log.WithField("duration", duration).Info("Waited for specified time")
		
	case "element":
		selector, _ := data["selector"].(string)
		c.log.WithField("selector", selector).Info("Simulated wait for element")
		// TODO: Implement with resource-browserless CLI
		
	default:
		time.Sleep(1 * time.Second)
		c.log.Info("Default wait executed")
	}

	return nil
}

// processExtractNode extracts data from pages
func (c *Client) processExtractNode(ctx context.Context, data map[string]interface{}, session *BrowserSession) error {
	selector, ok := data["selector"].(string)
	if !ok {
		return fmt.Errorf("extract node missing selector")
	}

	attribute, _ := data["attribute"].(string)
	if attribute == "" {
		attribute = "text" // Default to text content
	}

	c.log.WithFields(logrus.Fields{
		"selector":  selector,
		"attribute": attribute,
	}).Info("Extracting data from element")

	// TODO: Implement with resource-browserless CLI
	// This would execute: resource-browserless extract <selector> <attribute>
	
	// For now, create a mock extracted data record
	extractedData := &database.ExtractedData{
		ID:          uuid.New(),
		ExecutionID: session.ExecutionID,
		StepName:    "extract",
		Timestamp:   time.Now(),
		DataKey:     fmt.Sprintf("extracted_%s", attribute),
		DataValue: database.JSONMap{
			"selector":  selector,
			"attribute": attribute,
			"value":     "mock_extracted_value",
		},
		DataType: "text",
	}

	return c.repo.CreateExtractedData(ctx, extractedData)
}

// CheckBrowserlessHealth checks if resource-browserless is available
func (c *Client) CheckBrowserlessHealth() error {
	// Try to run resource-browserless status
	cmd := exec.Command("resource-browserless", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("resource-browserless not available: %w", err)
	}
	return nil
}