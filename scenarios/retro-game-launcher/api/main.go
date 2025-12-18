package main

import (
	"github.com/vrooli/api-core/preflight"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Game struct {
	ID           string    `json:"id" db:"id"`
	Title        string    `json:"title" db:"title"`
	Prompt       string    `json:"prompt" db:"prompt"`
	Description  *string   `json:"description" db:"description"`
	Code         string    `json:"code" db:"code"`
	Engine       string    `json:"engine" db:"engine"`
	AuthorID     *string   `json:"author_id" db:"author_id"`
	ParentGameID *string   `json:"parent_game_id" db:"parent_game_id"`
	IsRemix      bool      `json:"is_remix" db:"is_remix"`
	ThumbnailURL *string   `json:"thumbnail_url" db:"thumbnail_url"`
	PlayCount    int       `json:"play_count" db:"play_count"`
	RemixCount   int       `json:"remix_count" db:"remix_count"`
	Rating       *float64  `json:"rating" db:"rating"`
	Tags         []string  `json:"tags" db:"tags"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	Published    bool      `json:"published" db:"published"`
}

type User struct {
	ID                    string     `json:"id" db:"id"`
	Username              string     `json:"username" db:"username"`
	Email                 string     `json:"email" db:"email"`
	SubscriptionTier      string     `json:"subscription_tier" db:"subscription_tier"`
	GamesCreatedThisMonth int        `json:"games_created_this_month" db:"games_created_this_month"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	LastLogin             *time.Time `json:"last_login" db:"last_login"`
}

type GameGenerationRequest struct {
	Prompt      string   `json:"prompt"`
	Engine      string   `json:"engine"`
	Tags        []string `json:"tags"`
	AuthorID    *string  `json:"author_id"`
	RemixFromID *string  `json:"remix_from_id"`
}

type HighScore struct {
	ID              string    `json:"id" db:"id"`
	GameID          string    `json:"game_id" db:"game_id"`
	UserID          *string   `json:"user_id" db:"user_id"`
	Score           int       `json:"score" db:"score"`
	PlayTimeSeconds *int      `json:"play_time_seconds" db:"play_time_seconds"`
	AchievedAt      time.Time `json:"achieved_at" db:"achieved_at"`
	Metadata        *string   `json:"metadata" db:"metadata"`
}

type APIServer struct {
	db        *sql.DB
	ollamaURL string
}

func (s *APIServer) ensureStarterContent(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("server not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// Remove leftover placeholder games that shipped with early prototypes
	cleanupQuery := `
		DELETE FROM games
		WHERE LOWER(title) LIKE 'create a simple test%'
		   OR LOWER(prompt) LIKE 'create a simple test%'`

	if _, err := s.db.ExecContext(ctx, cleanupQuery); err != nil {
		return err
	}

	tagsColumnType, err := s.lookupColumnType(ctx, "games", "tags")
	if err != nil {
		// Default to ARRAY when the schema lookup fails (older test harnesses)
		tagsColumnType = "ARRAY"
	}

	type starterGame struct {
		Title       string
		Prompt      string
		Description string
		Code        string
		Engine      string
		Tags        []string
		PlayCount   int
		Rating      float64
	}

	starterGames := []starterGame{
		{
			Title:       "Neon Snake",
			Prompt:      "Create a neon-lit snake game with crunchy 8-bit sound effects and simple controls",
			Description: "Guide a glowing snake through a synthwave grid, collecting energy pellets while avoiding your tail.",
			Code: strings.TrimSpace(`
const canvas = document.createElement('canvas');
canvas.width = 420;
canvas.height = 420;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

let snake = [{ x: 10, y: 10 }];
let direction = { x: 1, y: 0 };
let food = { x: 5, y: 5 };
let speed = 160;

function drawBlock(x, y, color) {
	ctx.fillStyle = color;
	ctx.fillRect(x * 20, y * 20, 18, 18);
}

function update() {
	const head = { x: snake[0].x + direction.x, y: snake[0].y + direction.y };
	snake.unshift(head);

	if (head.x === food.x && head.y === food.y) {
		food = { x: Math.floor(Math.random() * 21), y: Math.floor(Math.random() * 21) };
	} else {
		snake.pop();
	}

	const hitWall = head.x < 0 || head.y < 0 || head.x > 20 || head.y > 20;
	const hitTail = snake.slice(1).some(segment => segment.x === head.x && segment.y === head.y);

	if (hitWall || hitTail) {
		snake = [{ x: 10, y: 10 }];
		direction = { x: 1, y: 0 };
	}
}

function draw() {
	ctx.fillStyle = '#080212';
	ctx.fillRect(0, 0, canvas.width, canvas.height);
	ctx.strokeStyle = '#1cf2ff';
	ctx.strokeRect(0, 0, canvas.width, canvas.height);
	drawBlock(food.x, food.y, '#ff00ff');
	snake.forEach((segment, index) => {
		const color = index === 0 ? '#00ff99' : '#00d16a';
		drawBlock(segment.x, segment.y, color);
	});
}

document.addEventListener('keydown', event => {
	const keyMap = {
		ArrowUp: { x: 0, y: -1 },
		ArrowDown: { x: 0, y: 1 },
		ArrowLeft: { x: -1, y: 0 },
		ArrowRight: { x: 1, y: 0 }
	};
	const next = keyMap[event.key];
	if (next && (next.x !== -direction.x || next.y !== -direction.y)) {
		direction = next;
	}
});

function loop() {
	update();
	draw();
	setTimeout(() => requestAnimationFrame(loop), speed);
}

loop();
`),
			Engine:    "javascript",
			Tags:      []string{"arcade", "snake", "classic"},
			PlayCount: 384,
			Rating:    4.7,
		},
		{
			Title:       "Pixel Breaker",
			Prompt:      "Build a brick breaker arcade game with neon particles and responsive paddle controls",
			Description: "Bounce energy orbs to clear progressive waves of neon bricks and chase the global high score.",
			Code: strings.TrimSpace(`
const canvas = document.createElement('canvas');
canvas.width = 640;
canvas.height = 480;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

const paddle = { x: 280, y: 440, w: 80, h: 16, speed: 8 };
const ball = { x: 320, y: 320, vx: 4, vy: -4, r: 8 };
const bricks = [];

for (let row = 0; row < 5; row++) {
	for (let col = 0; col < 10; col++) {
		bricks.push({ x: 48 + col * 56, y: 60 + row * 28, w: 48, h: 20, alive: true });
	}
}

function update() {
	ball.x += ball.vx;
	ball.y += ball.vy;

	if (ball.x < ball.r || ball.x > canvas.width - ball.r) ball.vx *= -1;
	if (ball.y < ball.r) ball.vy *= -1;

	if (ball.y > canvas.height) {
		ball.x = 320;
		ball.y = 320;
		ball.vx = 4;
		ball.vy = -4;
	}

	if (ball.y + ball.r >= paddle.y && ball.x >= paddle.x && ball.x <= paddle.x + paddle.w) {
		ball.vy *= -1;
		ball.y = paddle.y - ball.r;
	}

	bricks.forEach(brick => {
		if (!brick.alive) return;
		const withinX = ball.x > brick.x && ball.x < brick.x + brick.w;
		const withinY = ball.y - ball.r < brick.y + brick.h && ball.y + ball.r > brick.y;
		if (withinX && withinY) {
			brick.alive = false;
			ball.vy *= -1;
		}
	});
}

function draw() {
	ctx.fillStyle = '#08040c';
	ctx.fillRect(0, 0, canvas.width, canvas.height);
	ctx.fillStyle = '#0ff';
	ctx.fillRect(paddle.x, paddle.y, paddle.w, paddle.h);
	ctx.fillStyle = '#f0f';
	ctx.beginPath();
	ctx.arc(ball.x, ball.y, ball.r, 0, Math.PI * 2);
	ctx.fill();

	bricks.forEach(brick => {
		if (!brick.alive) return;
		ctx.fillStyle = '#ff0077';
		ctx.fillRect(brick.x, brick.y, brick.w, brick.h);
	});
}

document.addEventListener('mousemove', event => {
	const rect = canvas.getBoundingClientRect();
	paddle.x = Math.max(0, Math.min(canvas.width - paddle.w, event.clientX - rect.left - paddle.w / 2));
});

function loop() {
	update();
	draw();
	requestAnimationFrame(loop);
}

loop();
`),
			Engine:    "javascript",
			Tags:      []string{"arcade", "breakout", "neon"},
			PlayCount: 256,
			Rating:    4.6,
		},
		{
			Title:       "Galactic Defender",
			Prompt:      "Create a classic space shooter where players blast invading waves and collect power-ups",
			Description: "Battle neon alien squadrons, upgrade your ship mid-flight, and survive escalating cosmic assaults.",
			Code: strings.TrimSpace(`
const canvas = document.createElement('canvas');
canvas.width = 720;
canvas.height = 480;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

const ship = { x: 360, y: 420, w: 32, h: 32 };
const lasers = [];
const enemies = [];
let tick = 0;

function spawnWave() {
	for (let i = 0; i < 8; i++) {
		enemies.push({ x: 80 + i * 70, y: 60, w: 28, h: 18, vx: Math.sin(tick / 30 + i) * 2 });
	}
}

function update() {
	tick++;
	ship.x = Math.max(16, Math.min(canvas.width - 48, ship.x));

	lasers.forEach(laser => (laser.y -= 8));
	for (let i = lasers.length - 1; i >= 0; i--) {
		if (lasers[i].y < -20) lasers.splice(i, 1);
	}

	enemies.forEach(enemy => {
		enemy.y += 0.3;
		enemy.x += enemy.vx;
	});

	enemies.forEach(enemy => {
		lasers.forEach(laser => {
			const overlapX = Math.abs(laser.x - enemy.x) < 24;
			const overlapY = Math.abs(laser.y - enemy.y) < 18;
			if (!enemy.dead && !laser.dead && overlapX && overlapY) {
				enemy.dead = true;
				laser.dead = true;
			}
		});
	});

	for (let i = lasers.length - 1; i >= 0; i--) {
		if (lasers[i].dead) lasers.splice(i, 1);
	}
	for (let i = enemies.length - 1; i >= 0; i--) {
		if (enemies[i].dead) enemies.splice(i, 1);
	}

	if (tick % 240 === 0 || enemies.length === 0) {
		spawnWave();
	}
}

function draw() {
	ctx.fillStyle = '#02030b';
	ctx.fillRect(0, 0, canvas.width, canvas.height);

	ctx.fillStyle = '#31f3ff';
	ctx.fillRect(ship.x, ship.y, ship.w, ship.h);

	ctx.fillStyle = '#ff2266';
	lasers.forEach(laser => ctx.fillRect(laser.x, laser.y, 4, 12));

	enemies.forEach(enemy => {
		ctx.fillStyle = '#8f00ff';
		ctx.fillRect(enemy.x, enemy.y, enemy.w, enemy.h);
	});
}

document.addEventListener('mousemove', event => {
	const rect = canvas.getBoundingClientRect();
	ship.x = event.clientX - rect.left - ship.w / 2;
});

document.addEventListener('click', () => {
	lasers.push({ x: ship.x + ship.w / 2, y: ship.y });
});

function loop() {
	update();
	draw();
	requestAnimationFrame(loop);
}

spawnWave();
loop();
`),
			Engine:    "javascript",
			Tags:      []string{"shooter", "space", "retro"},
			PlayCount: 512,
			Rating:    4.8,
		},
		{
			Title:       "Turbo Tunnels",
			Prompt:      "Design a top-down hover bike racer dodging tunnels and collecting boost orbs",
			Description: "Dash through endless neon tunnels, weaving through obstacles and chaining boosts for max velocity.",
			Code: strings.TrimSpace(`
const canvas = document.createElement('canvas');
canvas.width = 800;
canvas.height = 480;
document.body.appendChild(canvas);
const ctx = canvas.getContext('2d');

const bike = { x: 400, y: 380, vx: 0 };
const obstacles = [];
let distance = 0;

function spawnObstacle() {
	const width = 160 + Math.random() * 200;
	const gap = Math.random() * (canvas.width - width);
	obstacles.push({ gapStart: gap, gapWidth: width, y: -40 });
}

function update() {
	distance += 1;
	bike.x += bike.vx;
	bike.x = Math.max(40, Math.min(canvas.width - 40, bike.x));

	obstacles.forEach(obstacle => {
		obstacle.y += 6;
		if (obstacle.y > canvas.height + 40) obstacle.remove = true;
	});

	obstacles.forEach(obstacle => {
		const colliding = obstacle.y > bike.y - 12 && obstacle.y < bike.y + 12;
		const outsideGap = bike.x < obstacle.gapStart || bike.x > obstacle.gapStart + obstacle.gapWidth;
		if (colliding && outsideGap) {
			distance = 0;
			bike.x = 400;
			bike.vx = 0;
			obstacles.length = 0;
		}
	});

	for (let i = obstacles.length - 1; i >= 0; i--) {
		if (obstacles[i].remove) obstacles.splice(i, 1);
	}

	if (distance % 45 === 0) spawnObstacle();
}

function draw() {
	ctx.fillStyle = '#05031a';
	ctx.fillRect(0, 0, canvas.width, canvas.height);

	ctx.strokeStyle = '#ff007a';
	ctx.lineWidth = 4;
	ctx.beginPath();
	ctx.moveTo(canvas.width / 2 - 40, 0);
	ctx.lineTo(canvas.width / 2 - 80, canvas.height);
	ctx.moveTo(canvas.width / 2 + 40, 0);
	ctx.lineTo(canvas.width / 2 + 80, canvas.height);
	ctx.stroke();

	ctx.fillStyle = '#00f0ff';
	ctx.beginPath();
	ctx.arc(bike.x, bike.y, 18, 0, Math.PI * 2);
	ctx.fill();

	ctx.fillStyle = '#ffef11';
	obstacles.forEach(obstacle => {
		ctx.fillRect(0, obstacle.y, obstacle.gapStart, 24);
		ctx.fillRect(obstacle.gapStart + obstacle.gapWidth, obstacle.y, canvas.width - obstacle.gapStart - obstacle.gapWidth, 24);
	});

	ctx.fillStyle = '#0ff';
	ctx.font = '18px monospace';
	ctx.fillText('Distance: ' + distance, 24, 36);
}

document.addEventListener('keydown', event => {
	if (event.key === 'ArrowLeft') bike.vx = -6;
	if (event.key === 'ArrowRight') bike.vx = 6;
});

document.addEventListener('keyup', event => {
	if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') bike.vx = 0;
});

function loop() {
	update();
	draw();
	requestAnimationFrame(loop);
}

loop();
`),
			Engine:    "javascript",
			Tags:      []string{"racing", "endless", "synthwave"},
			PlayCount: 298,
			Rating:    4.5,
		},
	}

	for _, game := range starterGames {
		var exists bool
		err := s.db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM games WHERE LOWER(title) = LOWER($1))`,
			game.Title,
		).Scan(&exists)
		if err != nil {
			return err
		}

		if exists {
			continue
		}

		tagsValue, err := prepareTagsValue(game.Tags, tagsColumnType)
		if err != nil {
			return err
		}

		now := time.Now()
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO games (
				id, title, prompt, description, code, engine,
				author_id, play_count, rating, tags, created_at, updated_at, published
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true)
		`,
			uuid.New().String(),
			game.Title,
			game.Prompt,
			game.Description,
			game.Code,
			game.Engine,
			nil,
			game.PlayCount,
			game.Rating,
			tagsValue,
			now,
			now,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *APIServer) lookupColumnType(ctx context.Context, table, column string) (string, error) {
	var dataType sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT data_type
		FROM information_schema.columns
		WHERE LOWER(table_name) = LOWER($1)
		  AND LOWER(column_name) = LOWER($2)
		LIMIT 1`, table, column).Scan(&dataType)
	if err != nil {
		return "", err
	}

	if !dataType.Valid {
		return "", fmt.Errorf("column %s.%s not found", table, column)
	}

	return strings.ToUpper(dataType.String), nil
}

func prepareTagsValue(tags []string, columnType string) (interface{}, error) {
	if strings.Contains(strings.ToUpper(columnType), "JSON") {
		payload, err := json.Marshal(tags)
		if err != nil {
			return nil, err
		}
		return string(payload), nil
	}

	if len(tags) == 0 {
		return "{}", nil
	}

	escaped := make([]string, len(tags))
	for i, tag := range tags {
		safe := strings.ReplaceAll(tag, "\"", "\\\"")
		escaped[i] = "\"" + safe + "\""
	}

	return "{" + strings.Join(escaped, ",") + "}", nil
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "retro-game-launcher",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Ollama URL - REQUIRED, no defaults
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	server := &APIServer{
		db:        db,
		ollamaURL: ollamaURL,
	}

	if err := server.ensureStarterContent(context.Background()); err != nil {
		log.Printf("⚠️  Unable to ensure starter content: %v", err)
	}

	router := mux.NewRouter()

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Health check
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Games endpoints
	api.HandleFunc("/games", server.getGames).Methods("GET")
	api.HandleFunc("/games", server.createGame).Methods("POST")
	api.HandleFunc("/games/{id}", server.getGame).Methods("GET")
	api.HandleFunc("/games/{id}", server.updateGame).Methods("PUT")
	api.HandleFunc("/games/{id}", server.deleteGame).Methods("DELETE")
	api.HandleFunc("/games/{id}/play", server.recordPlay).Methods("POST")
	api.HandleFunc("/games/{id}/remix", server.createRemix).Methods("POST")

	// Game generation
	api.HandleFunc("/generate", server.generateGame).Methods("POST")
	api.HandleFunc("/generate/status/{id}", server.getGenerationStatus).Methods("GET")

	// High scores
	api.HandleFunc("/games/{id}/scores", server.getHighScores).Methods("GET")
	api.HandleFunc("/games/{id}/scores", server.submitScore).Methods("POST")

	// Search and discovery
	api.HandleFunc("/search/games", server.searchGames).Methods("GET")
	api.HandleFunc("/featured", server.getFeaturedGames).Methods("GET")
	api.HandleFunc("/trending", server.getTrendingGames).Methods("GET")

	// Template and prompt management
	api.HandleFunc("/templates", server.getPromptTemplates).Methods("GET")
	api.HandleFunc("/templates/{id}", server.getPromptTemplate).Methods("GET")

	// Users (basic endpoints)
	api.HandleFunc("/users", server.createUser).Methods("POST")
	api.HandleFunc("/users/{id}", server.getUser).Methods("GET")

	log.Printf("🚀 Retro Game Launcher API starting on port %s", port)
	log.Printf("🎮 Database: %s", postgresURL)
	log.Printf("🧠 Ollama: %s", ollamaURL)

	handler := corsHandler(router)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// getEnv removed to prevent hardcoded defaults

func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]interface{}{
			"database": s.checkDatabase(),
			"ollama":   s.checkOllama(),
		},
	}

	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) checkDatabase() string {
	if err := s.db.Ping(); err != nil {
		return "unhealthy"
	}
	return "healthy"
}

func (s *APIServer) checkOllama() string {
	resp, err := http.Get(s.ollamaURL + "/api/tags")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) getGames(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE published = true 
		ORDER BY created_at DESC 
		LIMIT 50`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var tagsJSON []byte

		err := rows.Scan(
			&game.ID, &game.Title, &game.Prompt, &game.Description,
			&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
			&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
			&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
			&game.UpdatedAt, &game.Published,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse tags JSON array
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &game.Tags)
		}

		games = append(games, game)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (s *APIServer) createGame(w http.ResponseWriter, r *http.Request) {
	var game Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate UUID for new game
	game.ID = uuid.New().String()
	game.CreatedAt = time.Now()
	game.UpdatedAt = time.Now()

	tagsJSON, _ := json.Marshal(game.Tags)

	query := `
		INSERT INTO games (id, title, prompt, description, code, engine, 
		                  author_id, parent_game_id, is_remix, thumbnail_url,
		                  tags, created_at, updated_at, published)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, play_count, remix_count, rating`

	err := s.db.QueryRow(query,
		game.ID, game.Title, game.Prompt, game.Description, game.Code,
		game.Engine, game.AuthorID, game.ParentGameID, game.IsRemix,
		game.ThumbnailURL, tagsJSON, game.CreatedAt, game.UpdatedAt, game.Published,
	).Scan(&game.ID, &game.PlayCount, &game.RemixCount, &game.Rating)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}

func (s *APIServer) getGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE id = $1`

	var game Game
	var tagsJSON []byte

	err := s.db.QueryRow(query, gameID).Scan(
		&game.ID, &game.Title, &game.Prompt, &game.Description,
		&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
		&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
		&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
		&game.UpdatedAt, &game.Published,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse tags JSON array
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &game.Tags)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func (s *APIServer) generateGame(w http.ResponseWriter, r *http.Request) {
	var req GameGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if req.Engine == "" {
		req.Engine = "html5" // default engine
	}

	// Start game generation process
	generationID := s.startGameGeneration(req)

	response := map[string]interface{}{
		"generation_id":  generationID,
		"status":         "started",
		"prompt":         req.Prompt,
		"estimated_time": "45-60 seconds",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) recordPlay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	// Increment play count
	query := `UPDATE games SET play_count = play_count + 1 WHERE id = $1`
	_, err := s.db.Exec(query, gameID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "play recorded"})
}

func (s *APIServer) getFeaturedGames(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE published = true AND rating > 4.0
		ORDER BY rating DESC, play_count DESC 
		LIMIT 12`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var tagsJSON []byte

		err := rows.Scan(
			&game.ID, &game.Title, &game.Prompt, &game.Description,
			&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
			&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
			&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
			&game.UpdatedAt, &game.Published,
		)
		if err != nil {
			continue
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &game.Tags)
		}

		games = append(games, game)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (s *APIServer) searchGames(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		http.Error(w, "Search term required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE published = true AND (
			title ILIKE $1 OR 
			description ILIKE $1 OR 
			prompt ILIKE $1 OR
			$2 = ANY(tags)
		)
		ORDER BY 
			CASE WHEN title ILIKE $1 THEN 1 ELSE 2 END,
			play_count DESC
		LIMIT 20`

	searchPattern := "%" + searchTerm + "%"
	rows, err := s.db.Query(query, searchPattern, searchTerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var tagsJSON []byte

		err := rows.Scan(
			&game.ID, &game.Title, &game.Prompt, &game.Description,
			&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
			&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
			&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
			&game.UpdatedAt, &game.Published,
		)
		if err != nil {
			continue
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &game.Tags)
		}

		games = append(games, game)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

// Stub implementations for remaining endpoints
func (s *APIServer) updateGame(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) deleteGame(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) createRemix(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getGenerationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	generationID := vars["id"]

	status, exists := s.getGenerationStatusByID(generationID)
	if !exists {
		http.Error(w, "Generation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) getHighScores(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) submitScore(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getTrendingGames(w http.ResponseWriter, r *http.Request) {
	s.getFeaturedGames(w, r) // Use featured for now
}

func (s *APIServer) getPromptTemplates(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getPromptTemplate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) createUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
