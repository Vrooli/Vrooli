package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

// ScraperOrchestrator manages web scraping operations
type ScraperOrchestrator struct {
	db          *sql.DB
	httpClient  *http.Client
	jobs        chan *ScrapeJob
	workers     int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	rateLimiter map[string]*time.Ticker
	mu          sync.RWMutex
}

// ScrapeJob represents a scraping task
type ScrapeJob struct {
	ID          string                 `json:"id"`
	URL         string                 `json:"url"`
	Type        string                 `json:"type"` // static, dynamic, api
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers"`
	Payload     map[string]interface{} `json:"payload"`
	Selectors   map[string]string      `json:"selectors"`
	Schedule    string                 `json:"schedule"`
	MaxRetries  int                    `json:"max_retries"`
	Timeout     int                    `json:"timeout_seconds"`
	UserAgent   string                 `json:"user_agent"`
	ProxyURL    string                 `json:"proxy_url"`
	JavaScript  bool                   `json:"javascript"`
	WaitFor     string                 `json:"wait_for"`
	CreatedAt   time.Time              `json:"created_at"`
	Status      string                 `json:"status"`
	RetryCount  int                    `json:"retry_count"`
}

// ScrapeResult contains the scraped data
type ScrapeResult struct {
	JobID       string                 `json:"job_id"`
	URL         string                 `json:"url"`
	StatusCode  int                    `json:"status_code"`
	Data        map[string]interface{} `json:"data"`
	HTML        string                 `json:"html"`
	Screenshot  []byte                 `json:"screenshot,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    int                    `json:"duration_ms"`
	ScrapedAt   time.Time              `json:"scraped_at"`
}

// NewScraperOrchestrator creates a new scraper orchestrator
func NewScraperOrchestrator(db *sql.DB) *ScraperOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ScraperOrchestrator{
		db:          db,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		jobs:        make(chan *ScrapeJob, 100),
		workers:     5,
		ctx:         ctx,
		cancel:      cancel,
		rateLimiter: make(map[string]*time.Ticker),
	}
}

// Start begins the scraper orchestrator
func (so *ScraperOrchestrator) Start() error {
	log.Println("Starting scraper orchestrator...")
	
	// Start worker pool
	for i := 0; i < so.workers; i++ {
		so.wg.Add(1)
		go so.worker()
	}
	
	// Start scheduled job processor
	go so.processScheduledJobs()
	
	// Start retry processor
	go so.processRetries()
	
	log.Printf("Scraper orchestrator started with %d workers", so.workers)
	return nil
}

// Stop gracefully shuts down the orchestrator
func (so *ScraperOrchestrator) Stop() {
	log.Println("Stopping scraper orchestrator...")
	
	so.cancel()
	close(so.jobs)
	so.wg.Wait()
	
	// Clean up rate limiters
	so.mu.Lock()
	for _, ticker := range so.rateLimiter {
		ticker.Stop()
	}
	so.mu.Unlock()
	
	log.Println("Scraper orchestrator stopped")
}

// SubmitJob adds a new scraping job to the queue
func (so *ScraperOrchestrator) SubmitJob(job *ScrapeJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	
	// Store job in database
	if err := so.storeJob(job); err != nil {
		return fmt.Errorf("failed to store job: %w", err)
	}
	
	// Add to queue
	select {
	case so.jobs <- job:
		log.Printf("Job %s submitted for URL: %s", job.ID, job.URL)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("job queue is full")
	}
}

// worker processes scraping jobs
func (so *ScraperOrchestrator) worker() {
	defer so.wg.Done()
	
	for job := range so.jobs {
		so.processJob(job)
	}
}

// processJob executes a scraping job
func (so *ScraperOrchestrator) processJob(job *ScrapeJob) {
	log.Printf("Processing job %s for URL: %s", job.ID, job.URL)
	
	// Apply rate limiting
	so.applyRateLimit(job.URL)
	
	// Update job status
	so.updateJobStatus(job.ID, "running")
	
	startTime := time.Now()
	var result *ScrapeResult
	var err error
	
	// Execute based on job type
	switch job.Type {
	case "dynamic", "javascript":
		result, err = so.scrapeDynamic(job)
	case "api":
		result, err = so.scrapeAPI(job)
	default:
		result, err = so.scrapeStatic(job)
	}
	
	duration := int(time.Since(startTime).Milliseconds())
	
	if err != nil {
		// Handle failure
		log.Printf("Job %s failed: %v", job.ID, err)
		result = &ScrapeResult{
			JobID:     job.ID,
			URL:       job.URL,
			Error:     err.Error(),
			Duration:  duration,
			ScrapedAt: time.Now(),
		}
		
		// Check if retry is needed
		if job.RetryCount < job.MaxRetries {
			so.scheduleRetry(job)
		} else {
			so.updateJobStatus(job.ID, "failed")
		}
	} else {
		// Success
		result.Duration = duration
		so.updateJobStatus(job.ID, "completed")
	}
	
	// Store result
	so.storeResult(result)
}

// scrapeStatic performs static HTML scraping
func (so *ScraperOrchestrator) scrapeStatic(job *ScrapeJob) (*ScrapeResult, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(so.ctx, job.Method, job.URL, nil)
	if err != nil {
		return nil, err
	}
	
	// Set headers
	for key, value := range job.Headers {
		req.Header.Set(key, value)
	}
	
	// Set user agent
	if job.UserAgent != "" {
		req.Header.Set("User-Agent", job.UserAgent)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; VrooliScraper/1.0)")
	}
	
	// Configure proxy if specified
	client := so.httpClient
	if job.ProxyURL != "" {
		proxyURL, err := url.Parse(job.ProxyURL)
		if err == nil {
			client = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
				Timeout: time.Duration(job.Timeout) * time.Second,
			}
		}
	}
	
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	
	// Extract data using selectors
	extractedData := make(map[string]interface{})
	for name, selector := range job.Selectors {
		if selector == "" {
			continue
		}
		
		// Handle different selector types
		if strings.HasPrefix(selector, "meta:") {
			// Meta tag extraction
			metaName := strings.TrimPrefix(selector, "meta:")
			content := doc.Find(fmt.Sprintf("meta[name='%s']", metaName)).AttrOr("content", "")
			extractedData[name] = content
		} else if strings.Contains(selector, "@") {
			// Attribute extraction
			parts := strings.Split(selector, "@")
			if len(parts) == 2 {
				elem := doc.Find(parts[0])
				extractedData[name] = elem.AttrOr(parts[1], "")
			}
		} else if strings.HasSuffix(selector, ":all") {
			// Multiple elements
			realSelector := strings.TrimSuffix(selector, ":all")
			var items []string
			doc.Find(realSelector).Each(func(i int, s *goquery.Selection) {
				items = append(items, strings.TrimSpace(s.Text()))
			})
			extractedData[name] = items
		} else {
			// Single element text
			extractedData[name] = strings.TrimSpace(doc.Find(selector).Text())
		}
	}
	
	return &ScrapeResult{
		JobID:      job.ID,
		URL:        job.URL,
		StatusCode: resp.StatusCode,
		Data:       extractedData,
		HTML:       string(body),
		ScrapedAt:  time.Now(),
	}, nil
}

// scrapeDynamic performs JavaScript-enabled scraping using Chrome
func (so *ScraperOrchestrator) scrapeDynamic(job *ScrapeJob) (*ScrapeResult, error) {
	// Create Chrome context
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent(job.UserAgent),
	}
	
	if job.ProxyURL != "" {
		opts = append(opts, chromedp.ProxyServer(job.ProxyURL))
	}
	
	allocCtx, cancel := chromedp.NewExecAllocator(so.ctx, opts...)
	defer cancel()
	
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	
	// Set timeout
	timeout := time.Duration(job.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	
	var htmlContent string
	var screenshot []byte
	extractedData := make(map[string]interface{})
	
	// Build Chrome actions
	actions := []chromedp.Action{
		chromedp.Navigate(job.URL),
	}
	
	// Wait for specific element if specified
	if job.WaitFor != "" {
		actions = append(actions, chromedp.WaitVisible(job.WaitFor))
	} else {
		actions = append(actions, chromedp.WaitReady("body"))
	}
	
	// Get HTML content
	actions = append(actions, chromedp.OuterHTML("html", &htmlContent))
	
	// Extract data using selectors
	for name, selector := range job.Selectors {
		if selector == "" {
			continue
		}
		
		var value string
		if strings.HasSuffix(selector, ":all") {
			// Multiple elements
			realSelector := strings.TrimSuffix(selector, ":all")
			var nodes []*cdp.Node
			actions = append(actions, chromedp.Nodes(realSelector, &nodes))
			// Process nodes after execution
		} else {
			// Single element
			actions = append(actions, chromedp.Text(selector, &value, chromedp.NodeVisible))
			extractedData[name] = value
		}
	}
	
	// Take screenshot if needed
	actions = append(actions, chromedp.CaptureScreenshot(&screenshot))
	
	// Execute Chrome actions
	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, err
	}
	
	return &ScrapeResult{
		JobID:      job.ID,
		URL:        job.URL,
		StatusCode: 200,
		Data:       extractedData,
		HTML:       htmlContent,
		Screenshot: screenshot,
		ScrapedAt:  time.Now(),
	}, nil
}

// scrapeAPI performs API endpoint scraping
func (so *ScraperOrchestrator) scrapeAPI(job *ScrapeJob) (*ScrapeResult, error) {
	// Prepare request body if needed
	var body io.Reader
	if job.Payload != nil && len(job.Payload) > 0 {
		jsonData, err := json.Marshal(job.Payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(so.ctx, job.Method, job.URL, body)
	if err != nil {
		return nil, err
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range job.Headers {
		req.Header.Set(key, value)
	}
	
	// Execute request
	client := so.httpClient
	if job.Timeout > 0 {
		client = &http.Client{
			Timeout: time.Duration(job.Timeout) * time.Second,
		}
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// Parse JSON response
	var responseData interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		// If not JSON, store as string
		responseData = string(respBody)
	}
	
	return &ScrapeResult{
		JobID:      job.ID,
		URL:        job.URL,
		StatusCode: resp.StatusCode,
		Data: map[string]interface{}{
			"response": responseData,
		},
		HTML:      string(respBody),
		ScrapedAt: time.Now(),
	}, nil
}

// processScheduledJobs handles scheduled scraping jobs
func (so *ScraperOrchestrator) processScheduledJobs() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-so.ctx.Done():
			return
		case <-ticker.C:
			so.checkScheduledJobs()
		}
	}
}

// checkScheduledJobs checks for and queues scheduled jobs
func (so *ScraperOrchestrator) checkScheduledJobs() {
	query := `
		SELECT id, url, type, method, headers, payload, selectors,
		       schedule, max_retries, timeout_seconds, user_agent,
		       proxy_url, javascript, wait_for
		FROM scrape_jobs
		WHERE status = 'scheduled' 
		  AND next_run <= NOW()
		ORDER BY priority DESC, created_at ASC
		LIMIT 10
	`
	
	rows, err := so.db.Query(query)
	if err != nil {
		log.Printf("Failed to query scheduled jobs: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var job ScrapeJob
		var headers, payload, selectors []byte
		
		err := rows.Scan(
			&job.ID, &job.URL, &job.Type, &job.Method,
			&headers, &payload, &selectors,
			&job.Schedule, &job.MaxRetries, &job.Timeout,
			&job.UserAgent, &job.ProxyURL, &job.JavaScript, &job.WaitFor,
		)
		if err != nil {
			continue
		}
		
		// Parse JSON fields
		json.Unmarshal(headers, &job.Headers)
		json.Unmarshal(payload, &job.Payload)
		json.Unmarshal(selectors, &job.Selectors)
		
		// Queue job
		select {
		case so.jobs <- &job:
			// Update next run time based on schedule
			so.updateNextRun(job.ID, job.Schedule)
		default:
			// Queue is full, skip
		}
	}
}

// processRetries handles failed jobs that need retry
func (so *ScraperOrchestrator) processRetries() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-so.ctx.Done():
			return
		case <-ticker.C:
			so.checkRetries()
		}
	}
}

// checkRetries checks for and queues jobs that need retry
func (so *ScraperOrchestrator) checkRetries() {
	query := `
		SELECT id, url, type, method, headers, payload, selectors,
		       max_retries, timeout_seconds, user_agent, proxy_url,
		       javascript, wait_for, retry_count
		FROM scrape_jobs
		WHERE status = 'retry_pending'
		  AND retry_count < max_retries
		  AND updated_at < NOW() - INTERVAL '1 minute' * POWER(2, retry_count)
		LIMIT 5
	`
	
	rows, err := so.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var job ScrapeJob
		var headers, payload, selectors []byte
		
		err := rows.Scan(
			&job.ID, &job.URL, &job.Type, &job.Method,
			&headers, &payload, &selectors,
			&job.MaxRetries, &job.Timeout,
			&job.UserAgent, &job.ProxyURL, &job.JavaScript,
			&job.WaitFor, &job.RetryCount,
		)
		if err != nil {
			continue
		}
		
		// Parse JSON fields
		json.Unmarshal(headers, &job.Headers)
		json.Unmarshal(payload, &job.Payload)
		json.Unmarshal(selectors, &job.Selectors)
		
		// Increment retry count
		job.RetryCount++
		
		// Queue for retry
		select {
		case so.jobs <- &job:
			log.Printf("Retrying job %s (attempt %d/%d)", job.ID, job.RetryCount, job.MaxRetries)
		default:
		}
	}
}

// applyRateLimit applies rate limiting for a domain
func (so *ScraperOrchestrator) applyRateLimit(targetURL string) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return
	}
	
	domain := u.Host
	
	so.mu.Lock()
	ticker, exists := so.rateLimiter[domain]
	so.mu.Unlock()
	
	if !exists {
		// Create new rate limiter for domain (1 request per second by default)
		ticker = time.NewTicker(1 * time.Second)
		so.mu.Lock()
		so.rateLimiter[domain] = ticker
		so.mu.Unlock()
	}
	
	// Wait for rate limit
	<-ticker.C
}

// scheduleRetry schedules a job for retry
func (so *ScraperOrchestrator) scheduleRetry(job *ScrapeJob) {
	job.RetryCount++
	query := `
		UPDATE scrape_jobs 
		SET status = 'retry_pending', 
		    retry_count = $2,
		    updated_at = NOW()
		WHERE id = $1
	`
	so.db.Exec(query, job.ID, job.RetryCount)
}

// Database operations

func (so *ScraperOrchestrator) storeJob(job *ScrapeJob) error {
	headers, _ := json.Marshal(job.Headers)
	payload, _ := json.Marshal(job.Payload)
	selectors, _ := json.Marshal(job.Selectors)
	
	query := `
		INSERT INTO scrape_jobs (
			id, url, type, method, headers, payload, selectors,
			schedule, max_retries, timeout_seconds, user_agent,
			proxy_url, javascript, wait_for, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW())
		ON CONFLICT (id) DO UPDATE SET
			updated_at = NOW()
	`
	
	_, err := so.db.Exec(query,
		job.ID, job.URL, job.Type, job.Method,
		headers, payload, selectors,
		job.Schedule, job.MaxRetries, job.Timeout,
		job.UserAgent, job.ProxyURL, job.JavaScript,
		job.WaitFor, "queued",
	)
	
	return err
}

func (so *ScraperOrchestrator) storeResult(result *ScrapeResult) {
	data, _ := json.Marshal(result.Data)
	
	query := `
		INSERT INTO scrape_results (
			job_id, url, status_code, data, html,
			screenshot, error, duration_ms, scraped_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	so.db.Exec(query,
		result.JobID, result.URL, result.StatusCode,
		data, result.HTML, result.Screenshot,
		result.Error, result.Duration, result.ScrapedAt,
	)
}

func (so *ScraperOrchestrator) updateJobStatus(jobID, status string) {
	query := `
		UPDATE scrape_jobs 
		SET status = $2, updated_at = NOW() 
		WHERE id = $1
	`
	so.db.Exec(query, jobID, status)
}

func (so *ScraperOrchestrator) updateNextRun(jobID, schedule string) {
	// Parse cron schedule and calculate next run
	// For simplicity, adding 1 hour for now
	query := `
		UPDATE scrape_jobs 
		SET next_run = NOW() + INTERVAL '1 hour'
		WHERE id = $1
	`
	so.db.Exec(query, jobID)
}