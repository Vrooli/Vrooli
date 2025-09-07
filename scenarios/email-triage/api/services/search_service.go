package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	
	"email-triage/models"
)

// SearchService handles vector search operations with Qdrant
type SearchService struct {
	client         qdrant.PointsClient
	collectionName string
	conn           *grpc.ClientConn
}

// NewSearchService creates a new SearchService instance
func NewSearchService(qdrantURL string) *SearchService {
	// Extract host:port from URL
	// For simplicity, assuming format like "http://localhost:6333"
	host := "localhost:6334" // Qdrant gRPC port
	if qdrantURL != "" {
		// Parse URL and extract host - simplified for now
		host = "localhost:6334"
	}
	
	// Create gRPC connection
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to Qdrant: %v", err)
		return nil
	}
	
	client := qdrant.NewPointsClient(conn)
	
	return &SearchService{
		client:         client,
		collectionName: "emails",
		conn:           conn,
	}
}

// Close closes the connection to Qdrant
func (ss *SearchService) Close() error {
	if ss.conn != nil {
		return ss.conn.Close()
	}
	return nil
}

// HealthCheck verifies Qdrant connection
func (ss *SearchService) HealthCheck() bool {
	if ss.client == nil {
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Try to get collection info
	collectionsClient := qdrant.NewCollectionsClient(ss.conn)
	_, err := collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: ss.collectionName,
	})
	
	// Collection not existing is OK, connection error is not
	return err == nil || isNotFoundError(err)
}

// CreateCollection creates the emails collection if it doesn't exist
func (ss *SearchService) CreateCollection() error {
	if ss.client == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	collectionsClient := qdrant.NewCollectionsClient(ss.conn)
	
	// Check if collection exists
	_, err := collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: ss.collectionName,
	})
	
	if err == nil {
		// Collection already exists
		return nil
	}
	
	if !isNotFoundError(err) {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	
	// Create collection
	_, err = collectionsClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: ss.collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     384, // Dimension for all-MiniLM-L6-v2 embeddings
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	
	log.Printf("Created Qdrant collection: %s", ss.collectionName)
	return nil
}

// IndexEmail stores an email vector in Qdrant for semantic search
func (ss *SearchService) IndexEmail(email *models.ProcessedEmail, embedding []float32) error {
	if ss.client == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Prepare point payload
	payload := map[string]*qdrant.Value{
		"email_id":       {Kind: &qdrant.Value_StringValue{StringValue: email.ID}},
		"account_id":     {Kind: &qdrant.Value_StringValue{StringValue: email.AccountID}},
		"subject":        {Kind: &qdrant.Value_StringValue{StringValue: email.Subject}},
		"sender":         {Kind: &qdrant.Value_StringValue{StringValue: email.SenderEmail}},
		"priority_score": {Kind: &qdrant.Value_DoubleValue{DoubleValue: email.PriorityScore}},
		"processed_at":   {Kind: &qdrant.Value_IntegerValue{IntegerValue: email.ProcessedAt.Unix()}},
	}
	
	// Create point
	point := &qdrant.PointStruct{
		Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: email.ID}},
		Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: embedding}}},
		Payload: payload,
	}
	
	// Upsert point
	_, err := ss.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: ss.collectionName,
		Wait:          &[]bool{true}[0], // Wait for indexing to complete
		Points:        []*qdrant.PointStruct{point},
	})
	
	if err != nil {
		return fmt.Errorf("failed to index email: %w", err)
	}
	
	return nil
}

// SearchEmails performs semantic search on indexed emails
func (ss *SearchService) SearchEmails(userID string, queryEmbedding []float32, limit int) ([]*models.SearchResult, error) {
	if ss.client == nil {
		return nil, fmt.Errorf("Qdrant client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Create filter for user's emails only (through account_id)
	// This is a simplified version - in production you'd need to join with accounts table
	filter := &qdrant.Filter{
		// For now, we'll search all emails and filter by user in the application layer
		// TODO: Implement proper user filtering
	}
	
	// Perform search
	searchResult, err := ss.client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: ss.collectionName,
		Vector:         queryEmbedding,
		Limit:          uint64(limit),
		Filter:         filter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})
	
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Convert results
	var results []*models.SearchResult
	for _, scored := range searchResult.Result {
		result := &models.SearchResult{
			EmailID:         getStringFromPayload(scored.Payload, "email_id"),
			Subject:         getStringFromPayload(scored.Payload, "subject"),
			Sender:          getStringFromPayload(scored.Payload, "sender"),
			SimilarityScore: float64(scored.Score),
			PriorityScore:   getDoubleFromPayload(scored.Payload, "priority_score"),
		}
		
		// Convert timestamp
		if timestamp := getIntFromPayload(scored.Payload, "processed_at"); timestamp > 0 {
			result.ProcessedAt = time.Unix(timestamp, 0)
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// DeleteEmailVector removes an email vector from Qdrant
func (ss *SearchService) DeleteEmailVector(emailID string) error {
	if ss.client == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := ss.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: ss.collectionName,
		Wait:          &[]bool{true}[0],
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						{PointIdOptions: &qdrant.PointId_Uuid{Uuid: emailID}},
					},
				},
			},
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to delete email vector: %w", err)
	}
	
	return nil
}

// Helper functions for payload extraction
func getStringFromPayload(payload map[string]*qdrant.Value, key string) string {
	if val, ok := payload[key]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			return strVal
		}
	}
	return ""
}

func getDoubleFromPayload(payload map[string]*qdrant.Value, key string) float64 {
	if val, ok := payload[key]; ok {
		return val.GetDoubleValue()
	}
	return 0
}

func getIntFromPayload(payload map[string]*qdrant.Value, key string) int64 {
	if val, ok := payload[key]; ok {
		return val.GetIntegerValue()
	}
	return 0
}

func isNotFoundError(err error) bool {
	// Simplified error checking - in production you'd check specific error types
	return err != nil && (fmt.Sprintf("%v", err) == "collection not found" || 
		fmt.Sprintf("%v", err) == "not found")
}