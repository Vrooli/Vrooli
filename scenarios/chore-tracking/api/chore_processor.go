package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

type ChoreProcessor struct {
	db               *sql.DB
	schedulerManager *SchedulerManager
	rewardManager    *RewardManager
	pointsCalculator *PointsCalculator
}

type SchedulerManager struct {
	db *sql.DB
}

type RewardManager struct {
	db *sql.DB
}

type PointsCalculator struct {
	db *sql.DB
}

type ChoreAssignment struct {
	ID         int       `json:"id"`
	ChoreID    int       `json:"chore_id"`
	UserID     int       `json:"user_id"`
	DueDate    time.Time `json:"due_date"`
	Status     string    `json:"status"`
	Points     int       `json:"points"`
	CreatedAt  time.Time `json:"created_at"`
}

type ScheduleRequest struct {
	UserID      int       `json:"user_id"`
	WeekStart   time.Time `json:"week_start"`
	Preferences SchedulePreferences `json:"preferences"`
}

type SchedulePreferences struct {
	MaxDailyChores int    `json:"max_daily_chores"`
	PreferredTime  string `json:"preferred_time"` // morning, afternoon, evening
	SkipDays       []int  `json:"skip_days"`      // Days of week to skip (0=Sunday)
}

type AchievementProgress struct {
	UserID         int     `json:"user_id"`
	AchievementID  int     `json:"achievement_id"`
	CurrentValue   int     `json:"current_value"`
	TargetValue    int     `json:"target_value"`
	Progress       float64 `json:"progress"`
}

func NewChoreProcessor(db *sql.DB) *ChoreProcessor {
	return &ChoreProcessor{
		db:               db,
		schedulerManager: &SchedulerManager{db: db},
		rewardManager:    &RewardManager{db: db},
		pointsCalculator: &PointsCalculator{db: db},
	}
}

// GenerateWeeklySchedule generates a weekly chore schedule for a user (replaces n8n chore-scheduler workflow)
func (cp *ChoreProcessor) GenerateWeeklySchedule(ctx context.Context, req ScheduleRequest) ([]ChoreAssignment, error) {
	// Get available chores based on frequency
	chores, err := cp.getAvailableChores(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available chores: %w", err)
	}

	// Get user's historical performance
	performance, err := cp.getUserPerformance(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user performance: %w", err)
	}

	// Generate schedule
	assignments := []ChoreAssignment{}
	weekStart := req.WeekStart
	
	for day := 0; day < 7; day++ {
		currentDate := weekStart.AddDate(0, 0, day)
		dayOfWeek := int(currentDate.Weekday())
		
		// Skip days if specified
		skipDay := false
		for _, skipDayNum := range req.Preferences.SkipDays {
			if dayOfWeek == skipDayNum {
				skipDay = true
				break
			}
		}
		if skipDay {
			continue
		}

		// Select chores for this day
		dailyChores := cp.selectDailyChores(chores, req.Preferences.MaxDailyChores, performance)
		
		// Create assignments
		for _, chore := range dailyChores {
			assignment := ChoreAssignment{
				ChoreID:   chore.ID,
				UserID:    req.UserID,
				DueDate:   currentDate,
				Status:    "pending",
				Points:    chore.Points,
				CreatedAt: time.Now(),
			}
			
			// Insert into database
			query := `
				INSERT INTO chore_assignments (chore_id, user_id, due_date, status, points, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id`
			
			err := cp.db.QueryRowContext(ctx, query, assignment.ChoreID, assignment.UserID,
				assignment.DueDate, assignment.Status, assignment.Points, assignment.CreatedAt).
				Scan(&assignment.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to create assignment: %w", err)
			}
			
			assignments = append(assignments, assignment)
		}
	}

	return assignments, nil
}

// CalculatePoints calculates points for completed chores (replaces n8n points-calculator workflow)
func (cp *ChoreProcessor) CalculatePoints(ctx context.Context, userID int, choreID int) (int, error) {
	// Get chore details
	var basePoints int
	var difficulty string
	query := `SELECT points, difficulty FROM chores WHERE id = $1`
	err := cp.db.QueryRowContext(ctx, query, choreID).Scan(&basePoints, &difficulty)
	if err != nil {
		return 0, fmt.Errorf("failed to get chore details: %w", err)
	}

	// Calculate bonus points based on streak
	streak, err := cp.getUserStreak(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user streak: %w", err)
	}

	// Apply multipliers
	points := basePoints
	
	// Difficulty multiplier
	switch difficulty {
	case "easy":
		points = int(float64(points) * 1.0)
	case "medium":
		points = int(float64(points) * 1.2)
	case "hard":
		points = int(float64(points) * 1.5)
	}
	
	// Streak bonus
	if streak > 0 {
		streakBonus := int(math.Min(float64(streak)*0.1, 0.5) * float64(points))
		points += streakBonus
	}

	// Update user's total points
	updateQuery := `UPDATE users SET total_points = total_points + $1 WHERE id = $2`
	_, err = cp.db.ExecContext(ctx, updateQuery, points, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to update user points: %w", err)
	}

	return points, nil
}

// ProcessAchievements checks and awards achievements (replaces n8n achievement-tracker workflow)
func (cp *ChoreProcessor) ProcessAchievements(ctx context.Context, userID int) ([]Achievement, error) {
	// Get all achievements
	query := `SELECT id, name, description, icon, points, criteria FROM achievements`
	rows, err := cp.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get achievements: %w", err)
	}
	defer rows.Close()

	var newAchievements []Achievement
	
	for rows.Next() {
		var achievement Achievement
		var criteriaJSON string
		err := rows.Scan(&achievement.ID, &achievement.Name, &achievement.Description,
			&achievement.Icon, &achievement.Points, &criteriaJSON)
		if err != nil {
			continue
		}

		// Check if user already has this achievement
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM user_achievements WHERE user_id = $1 AND achievement_id = $2)`
		cp.db.QueryRowContext(ctx, checkQuery, userID, achievement.ID).Scan(&exists)
		if exists {
			continue
		}

		// Parse criteria and check if met
		var criteria map[string]interface{}
		json.Unmarshal([]byte(criteriaJSON), &criteria)
		
		if cp.checkAchievementCriteria(ctx, userID, criteria) {
			// Award achievement
			now := time.Now()
			insertQuery := `
				INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
				VALUES ($1, $2, $3)`
			_, err = cp.db.ExecContext(ctx, insertQuery, userID, achievement.ID, now)
			if err == nil {
				achievement.UnlockedAt = &now
				newAchievements = append(newAchievements, achievement)
				
				// Award points for achievement
				updatePointsQuery := `UPDATE users SET total_points = total_points + $1 WHERE id = $2`
				cp.db.ExecContext(ctx, updatePointsQuery, achievement.Points, userID)
			}
		}
	}

	return newAchievements, nil
}

// ManageRewards handles reward redemption (replaces n8n reward-manager workflow)
func (cp *ChoreProcessor) ManageRewards(ctx context.Context, userID int, rewardID int) error {
	// Get reward cost
	var cost int
	var available bool
	query := `SELECT cost, available FROM rewards WHERE id = $1`
	err := cp.db.QueryRowContext(ctx, query, rewardID).Scan(&cost, &available)
	if err != nil {
		return fmt.Errorf("failed to get reward details: %w", err)
	}

	if !available {
		return fmt.Errorf("reward is not available")
	}

	// Check user's points
	var userPoints int
	pointsQuery := `SELECT total_points FROM users WHERE id = $1`
	err = cp.db.QueryRowContext(ctx, pointsQuery, userID).Scan(&userPoints)
	if err != nil {
		return fmt.Errorf("failed to get user points: %w", err)
	}

	if userPoints < cost {
		return fmt.Errorf("insufficient points: have %d, need %d", userPoints, cost)
	}

	// Deduct points and record redemption
	tx, err := cp.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Deduct points
	_, err = tx.ExecContext(ctx, `UPDATE users SET total_points = total_points - $1 WHERE id = $2`, cost, userID)
	if err != nil {
		return fmt.Errorf("failed to deduct points: %w", err)
	}

	// Record redemption
	_, err = tx.ExecContext(ctx, `
		INSERT INTO reward_redemptions (user_id, reward_id, points_spent, redeemed_at)
		VALUES ($1, $2, $3, $4)`,
		userID, rewardID, cost, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record redemption: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper functions
func (cp *ChoreProcessor) getAvailableChores(ctx context.Context, userID int) ([]Chore, error) {
	query := `
		SELECT c.id, c.title, c.description, c.points, c.difficulty, c.category, c.frequency
		FROM chores c
		WHERE c.status = 'active'
		ORDER BY c.frequency, c.difficulty`
	
	rows, err := cp.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chores []Chore
	for rows.Next() {
		var chore Chore
		err := rows.Scan(&chore.ID, &chore.Title, &chore.Description, &chore.Points,
			&chore.Difficulty, &chore.Category, &chore.Frequency)
		if err != nil {
			continue
		}
		chores = append(chores, chore)
	}

	return chores, nil
}

func (cp *ChoreProcessor) getUserPerformance(ctx context.Context, userID int) (map[int]float64, error) {
	query := `
		SELECT chore_id, 
		       COUNT(CASE WHEN status = 'completed' THEN 1 END)::FLOAT / COUNT(*) as completion_rate
		FROM chore_assignments
		WHERE user_id = $1
		  AND created_at > NOW() - INTERVAL '30 days'
		GROUP BY chore_id`
	
	rows, err := cp.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	performance := make(map[int]float64)
	for rows.Next() {
		var choreID int
		var rate float64
		if err := rows.Scan(&choreID, &rate); err == nil {
			performance[choreID] = rate
		}
	}

	return performance, nil
}

func (cp *ChoreProcessor) selectDailyChores(chores []Chore, maxCount int, performance map[int]float64) []Chore {
	// Simple selection algorithm - can be enhanced with ML
	selected := []Chore{}
	
	// Shuffle chores for variety
	rand.Seed(time.Now().UnixNano())
	shuffled := make([]Chore, len(chores))
	copy(shuffled, chores)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Select up to maxCount chores, prioritizing by frequency and performance
	for _, chore := range shuffled {
		if len(selected) >= maxCount {
			break
		}
		
		// Skip chores with very low completion rate
		if rate, exists := performance[chore.ID]; exists && rate < 0.2 {
			continue
		}
		
		// Add chore based on frequency
		if chore.Frequency == "daily" || (chore.Frequency == "weekly" && rand.Float64() < 0.3) {
			selected = append(selected, chore)
		}
	}

	return selected
}

func (cp *ChoreProcessor) getUserStreak(ctx context.Context, userID int) (int, error) {
	var streak int
	query := `SELECT current_streak FROM users WHERE id = $1`
	err := cp.db.QueryRowContext(ctx, query, userID).Scan(&streak)
	if err != nil {
		return 0, err
	}
	return streak, nil
}

func (cp *ChoreProcessor) checkAchievementCriteria(ctx context.Context, userID int, criteria map[string]interface{}) bool {
	// Check different types of criteria
	if criteriaType, ok := criteria["type"].(string); ok {
		switch criteriaType {
		case "total_points":
			if target, ok := criteria["target"].(float64); ok {
				var points int
				query := `SELECT total_points FROM users WHERE id = $1`
				cp.db.QueryRowContext(ctx, query, userID).Scan(&points)
				return float64(points) >= target
			}
		case "streak":
			if target, ok := criteria["target"].(float64); ok {
				var streak int
				query := `SELECT current_streak FROM users WHERE id = $1`
				cp.db.QueryRowContext(ctx, query, userID).Scan(&streak)
				return float64(streak) >= target
			}
		case "chores_completed":
			if target, ok := criteria["target"].(float64); ok {
				var count int
				query := `SELECT COUNT(*) FROM chore_assignments WHERE user_id = $1 AND status = 'completed'`
				cp.db.QueryRowContext(ctx, query, userID).Scan(&count)
				return float64(count) >= target
			}
		}
	}
	
	return false
}