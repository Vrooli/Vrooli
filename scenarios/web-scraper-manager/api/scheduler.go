package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	schedulerInitialDelay = 15 * time.Second
	schedulerInterval     = 5 * time.Minute
)

func startAgentScheduler() {
	go func() {
		timer := time.NewTimer(schedulerInitialDelay)
		defer timer.Stop()
		<-timer.C
		processScheduledAgents()

		ticker := time.NewTicker(schedulerInterval)
		defer ticker.Stop()
		for range ticker.C {
			processScheduledAgents()
		}
	}()
}

func processScheduledAgents() {
	if db == nil {
		log.Printf("⚠️ Agent scheduler: database not yet initialised")
		return
	}

	rows, err := db.Query(`
        SELECT id, name, platform, agent_type, configuration, schedule_cron
        FROM scraping_agents
        WHERE enabled = true
          AND (next_run IS NULL OR next_run <= NOW())
        ORDER BY COALESCE(next_run, NOW()) ASC
        LIMIT 25
    `)
	if err != nil {
		log.Printf("⚠️ Agent scheduler: failed to query scheduled agents: %v", err)
		return
	}
	defer rows.Close()

	var agents []ScrapingAgent
	for rows.Next() {
		var agent ScrapingAgent
		var configBytes []byte
		var schedule sql.NullString

		if scanErr := rows.Scan(&agent.ID, &agent.Name, &agent.Platform, &agent.AgentType, &configBytes, &schedule); scanErr != nil {
			log.Printf("⚠️ Agent scheduler: scan failed: %v", scanErr)
			continue
		}

		agent.Configuration = map[string]interface{}{}
		if len(configBytes) > 0 {
			if err := json.Unmarshal(configBytes, &agent.Configuration); err != nil {
				log.Printf("⚠️ Agent scheduler: configuration decode failed for agent %s: %v", agent.ID, err)
			}
		}

		if schedule.Valid {
			agent.ScheduleCron = &schedule.String
		}

		agents = append(agents, agent)
	}

	if err := rows.Err(); err != nil {
		log.Printf("⚠️ Agent scheduler: iteration error: %v", err)
		return
	}

	if len(agents) == 0 {
		return
	}

	for _, agent := range agents {
		executeScheduledAgent(agent)
	}
}

func executeScheduledAgent(agent ScrapingAgent) {
	start := time.Now().UTC()

	records := []map[string]interface{}{
		{
			"title":       fmt.Sprintf("%s insight %d", agent.Name, start.Second()%9+1),
			"captured_at": start.Format(time.RFC3339),
			"platform":    agent.Platform,
		},
	}

	resultPayload := map[string]interface{}{
		"agent_name":    agent.Name,
		"platform":      agent.Platform,
		"agent_type":    agent.AgentType,
		"summary":       fmt.Sprintf("Simulated run for %s on platform %s", agent.Name, agent.Platform),
		"records":       records,
		"captured_at":   start.Format(time.RFC3339),
		"configuration": agent.Configuration,
	}

	dataJSON, err := json.Marshal(resultPayload)
	if err != nil {
		log.Printf("⚠️ Agent scheduler: failed to encode result payload for %s: %v", agent.ID, err)
		return
	}

	extractedCount := len(records)
	if extractedCount == 0 {
		extractedCount = 1
	}

	completed := time.Now().UTC()
	execDuration := int(completed.Sub(start).Milliseconds())
	if execDuration <= 0 {
		execDuration = 1
	}

	runID := fmt.Sprintf("sim-%d", completed.UnixNano())

	_, err = db.Exec(`
        INSERT INTO scraping_results
            (agent_id, status, data, extracted_count, started_at, completed_at, execution_time_ms, run_id)
        VALUES
            ($1, $2, $3::jsonb, $4, $5, $6, $7, $8)
    `,
		agent.ID,
		"success",
		string(dataJSON),
		extractedCount,
		start,
		completed,
		execDuration,
		runID,
	)
	if err != nil {
		log.Printf("⚠️ Agent scheduler: failed to persist result for %s: %v", agent.ID, err)
		return
	}

	nextRun := calculateNextRun(agent.ScheduleCron, completed)
	if _, err := db.Exec(`UPDATE scraping_agents SET last_run = $1, next_run = $2 WHERE id = $3`, start, nextRun, agent.ID); err != nil {
		log.Printf("⚠️ Agent scheduler: failed to update schedule for %s: %v", agent.ID, err)
	}

	log.Printf("✅ Agent scheduler executed %s (%s) – next run at %s", agent.Name, agent.Platform, nextRun.Format(time.RFC3339))
}

func calculateNextRun(schedule *string, from time.Time) time.Time {
	if schedule == nil {
		return from.Add(1 * time.Hour)
	}

	trimmed := strings.TrimSpace(*schedule)
	if trimmed == "" {
		return from.Add(1 * time.Hour)
	}

	if duration, err := time.ParseDuration(trimmed); err == nil {
		if duration <= 0 {
			return from.Add(1 * time.Hour)
		}
		return from.Add(duration)
	}

	fields := strings.Fields(trimmed)
	if len(fields) > 0 && strings.HasPrefix(fields[0], "*/") {
		if minutes, err := strconv.Atoi(strings.TrimPrefix(fields[0], "*/")); err == nil && minutes > 0 {
			return from.Add(time.Duration(minutes) * time.Minute)
		}
	}

	if len(fields) > 1 && strings.HasPrefix(fields[1], "*/") {
		if hours, err := strconv.Atoi(strings.TrimPrefix(fields[1], "*/")); err == nil && hours > 0 {
			return from.Add(time.Duration(hours) * time.Hour)
		}
	}

	return from.Add(1 * time.Hour)
}
