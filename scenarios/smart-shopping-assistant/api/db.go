package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type Database struct {
	postgres *sql.DB
	redis    *redis.Client
}

func NewDatabase() (*Database, error) {
	// Connect to PostgreSQL
	postgresHost := os.Getenv("POSTGRES_HOST")
	if postgresHost == "" {
		postgresHost = "localhost"
	}
	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=postgres sslmode=disable",
		postgresHost, postgresPort)

	pgDB, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Warning: PostgreSQL connection failed: %v", err)
		// Continue without database for now
		pgDB = nil
	} else {
		// Test connection
		err = pgDB.Ping()
		if err != nil {
			log.Printf("Warning: PostgreSQL ping failed: %v", err)
			pgDB = nil
		}
	}

	// Connect to Redis
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "",
		DB:       0,
	})

	// Test Redis connection
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		rdb = nil
	}

	return &Database{
		postgres: pgDB,
		redis:    rdb,
	}, nil
}

func (db *Database) SearchProducts(ctx context.Context, query string, budgetMax float64) ([]Product, error) {
	// Check cache first
	if db.redis != nil {
		cacheKey := fmt.Sprintf("search:%s:%.2f", query, budgetMax)
		cached, err := db.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var products []Product
			if err := json.Unmarshal([]byte(cached), &products); err == nil {
				log.Printf("Cache hit for query: %s", query)
				return products, nil
			}
		}
	}

	// If database is available, search there
	if db.postgres != nil {
		// TODO: Implement actual database search
		// For now, return enhanced mock data
	}

	// Try deep research integration (mock for now since scenario doesn't exist)
	deepResearchData := db.callDeepResearch(ctx, query, budgetMax)
	if deepResearchData != nil {
		// Cache the result
		if db.redis != nil {
			cacheKey := fmt.Sprintf("search:%s:%.2f", query, budgetMax)
			data, _ := json.Marshal(deepResearchData)
			db.redis.Set(ctx, cacheKey, data, 5*time.Minute)
		}
		return deepResearchData, nil
	}

	// Return enhanced mock data with actual search term
	products := []Product{
		{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("Premium %s", query),
			Category:      "Electronics",
			Description:   fmt.Sprintf("High-quality %s within your budget", query),
			CurrentPrice:  budgetMax * 0.9,
			OriginalPrice: budgetMax * 1.2,
			AffiliateLink: fmt.Sprintf("https://example.com/product/%s?ref=smartshop", uuid.New().String()),
			ReviewsSummary: &ReviewsSummary{
				AverageRating: 4.5,
				TotalReviews:  1250,
				Pros:          []string{"Great quality", "Good value", "Fast shipping"},
				Cons:          []string{"Limited colors"},
			},
		},
		{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("Budget %s", query),
			Category:      "Electronics",
			Description:   fmt.Sprintf("Affordable %s option", query),
			CurrentPrice:  budgetMax * 0.6,
			OriginalPrice: budgetMax * 0.75,
			AffiliateLink: fmt.Sprintf("https://example.com/product/%s?ref=smartshop", uuid.New().String()),
			ReviewsSummary: &ReviewsSummary{
				AverageRating: 4.2,
				TotalReviews:  850,
				Pros:          []string{"Affordable", "Good for basic use"},
				Cons:          []string{"Basic features only"},
			},
		},
	}

	// Cache results
	if db.redis != nil {
		cacheKey := fmt.Sprintf("search:%s:%.2f", query, budgetMax)
		data, _ := json.Marshal(products)
		db.redis.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return products, nil
}

func (db *Database) FindAlternatives(ctx context.Context, productID string, price float64) []Alternative {
	alternatives := []Alternative{
		{
			Product: Product{
				ID:           uuid.New().String(),
				Name:         "Generic Alternative",
				Category:     "Electronics",
				CurrentPrice: price * 0.6,
			},
			AlternativeType: "generic",
			SavingsAmount:   price * 0.4,
			Reason:          "Similar features at lower price point",
		},
		{
			Product: Product{
				ID:           uuid.New().String(),
				Name:         "Refurbished Option",
				Category:     "Electronics",
				CurrentPrice: price * 0.7,
			},
			AlternativeType: "refurbished",
			SavingsAmount:   price * 0.3,
			Reason:          "Certified refurbished with warranty",
		},
		{
			Product: Product{
				ID:           uuid.New().String(),
				Name:         "Previous Generation",
				Category:     "Electronics",
				CurrentPrice: price * 0.5,
			},
			AlternativeType: "older_model",
			SavingsAmount:   price * 0.5,
			Reason:          "Last year's model with 90% of features",
		},
	}

	return alternatives
}

func (db *Database) GetPriceHistory(ctx context.Context, productID string) PriceInsights {
	// TODO: Implement actual price history from database
	return PriceInsights{
		CurrentTrend:   "declining",
		BestTimeToWait: false,
		PredictedDrop:  10.0,
		HistoricalLow:  89.99,
		HistoricalHigh: 149.99,
	}
}

func (db *Database) GenerateAffiliateLinks(products []Product) []AffiliateLink {
	links := []AffiliateLink{}
	retailers := []string{"Amazon", "BestBuy", "Walmart", "Target"}

	for _, product := range products {
		for _, retailer := range retailers {
			links = append(links, AffiliateLink{
				ProductID:  product.ID,
				Retailer:   retailer,
				URL:        fmt.Sprintf("https://%s.com/p/%s?ref=smartshop", retailer, product.ID),
				Commission: 2.0 + float64(len(retailer)%3),
			})
		}
	}

	return links
}

// callDeepResearch simulates integration with deep-research scenario
// In production, this would call the actual deep-research API
func (db *Database) callDeepResearch(ctx context.Context, query string, budgetMax float64) []Product {
	// Check if deep-research scenario is available
	deepResearchPort := os.Getenv("DEEP_RESEARCH_API_PORT")
	if deepResearchPort == "" {
		// Deep research not available, return nil to fall back to basic search
		log.Printf("Deep research scenario not available, using basic search")
		return nil
	}

	// In a real implementation, this would make an HTTP call to deep-research
	// For now, return enhanced mock data that simulates deep research
	products := []Product{
		{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("AI-Researched %s", query),
			Category:      "Electronics",
			Description:   fmt.Sprintf("Deep analysis found this %s with best value-to-performance ratio", query),
			CurrentPrice:  budgetMax * 0.85, // 85% of budget
			OriginalPrice: budgetMax * 1.1,  // Was 10% over budget
			AffiliateLink: fmt.Sprintf("https://example.com/deep-research/%s?ref=smartshop", uuid.New().String()),
			Features: map[string]interface{}{
				"ai_score":        95,
				"value_rating":    "Excellent",
				"research_depth":  "Comprehensive",
				"sources_checked": 47,
			},
			ReviewsSummary: &ReviewsSummary{
				AverageRating: 4.7,
				TotalReviews:  3421,
				Pros:          []string{"Best in class", "AI verified quality", "Excellent value"},
				Cons:          []string{"High demand, limited stock"},
			},
		},
		{
			ID:            uuid.New().String(),
			Name:          fmt.Sprintf("Deep-Research Alternative %s", query),
			Category:      "Electronics",
			Description:   "AI found this hidden gem with 90% of features at 60% cost",
			CurrentPrice:  budgetMax * 0.6,
			OriginalPrice: budgetMax * 0.75,
			AffiliateLink: fmt.Sprintf("https://example.com/deep-alt/%s?ref=smartshop", uuid.New().String()),
			Features: map[string]interface{}{
				"ai_score":       88,
				"value_rating":   "Outstanding",
				"research_depth": "Comprehensive",
			},
			ReviewsSummary: &ReviewsSummary{
				AverageRating: 4.4,
				TotalReviews:  1876,
				Pros:          []string{"Incredible value", "Nearly identical performance", "Available now"},
				Cons:          []string{"Lesser known brand"},
			},
		},
	}

	log.Printf("Deep research completed for query: %s (simulated)", query)
	return products
}

func (db *Database) Close() {
	if db.postgres != nil {
		db.postgres.Close()
	}
	if db.redis != nil {
		db.redis.Close()
	}
}
