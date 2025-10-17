package services

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	
	"email-triage/models"
)

// SearchService handles vector search operations with Qdrant
type SearchService struct {
	client         pb.PointsClient
	collectionName string
	conn           *grpc.ClientConn
}

// NewSearchService creates a new SearchService instance
func NewSearchService(qdrantURL string) *SearchService {
	// Extract host:port from URL
	// Default to localhost:6334 for gRPC (REST is on 6333)
	host := "localhost:6334" // Qdrant gRPC port
	if qdrantURL != "" {
		// Parse URL and convert to gRPC port
		// If REST URL is http://localhost:6333, gRPC is localhost:6334
		if qdrantURL == "http://localhost:6333" {
			host = "localhost:6334"
		}
	}
	
	// Create gRPC connection
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to Qdrant: %v", err)
		return nil
	}
	
	client := pb.NewPointsClient(conn)
	
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
	collectionsClient := pb.NewCollectionsClient(ss.conn)
	_, err := collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{
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
	
	collectionsClient := pb.NewCollectionsClient(ss.conn)
	
	// Check if collection exists
	_, err := collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{
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
	_, err = collectionsClient.Create(ctx, &pb.CreateCollection{
		CollectionName: ss.collectionName,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     384, // Dimension for all-MiniLM-L6-v2 embeddings
					Distance: pb.Distance_Cosine,
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
	payload := map[string]*pb.Value{
		"email_id":       {Kind: &pb.Value_StringValue{StringValue: email.ID}},
		"account_id":     {Kind: &pb.Value_StringValue{StringValue: email.AccountID}},
		"subject":        {Kind: &pb.Value_StringValue{StringValue: email.Subject}},
		"sender":         {Kind: &pb.Value_StringValue{StringValue: email.SenderEmail}},
		"priority_score": {Kind: &pb.Value_DoubleValue{DoubleValue: email.PriorityScore}},
		"processed_at":   {Kind: &pb.Value_IntegerValue{IntegerValue: email.ProcessedAt.Unix()}},
	}
	
	// Create point
	point := &pb.PointStruct{
		Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: email.ID}},
		Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: embedding}}},
		Payload: payload,
	}
	
	// Upsert point
	_, err := ss.client.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: ss.collectionName,
		Wait:          &[]bool{true}[0], // Wait for indexing to complete
		Points:        []*pb.PointStruct{point},
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
	filter := &pb.Filter{
		// For now, we'll search all emails and filter by user in the application layer
		// TODO: Implement proper user filtering
	}
	
	// Perform search
	searchResult, err := ss.client.Search(ctx, &pb.SearchPoints{
		CollectionName: ss.collectionName,
		Vector:         queryEmbedding,
		Limit:          uint64(limit),
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
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

// StoreEmailVector stores an email vector in Qdrant
func (ss *SearchService) StoreEmailVector(vectorID string, embedding []float32, payload map[string]interface{}) error {
	if ss.client == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Convert payload to Qdrant format
	pbPayload := make(map[string]*pb.Value)
	for key, val := range payload {
		switch v := val.(type) {
		case string:
			pbPayload[key] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: v}}
		case float64:
			pbPayload[key] = &pb.Value{Kind: &pb.Value_DoubleValue{DoubleValue: v}}
		case int64:
			pbPayload[key] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: v}}
		case int:
			pbPayload[key] = &pb.Value{Kind: &pb.Value_IntegerValue{IntegerValue: int64(v)}}
		}
	}
	
	// Create point
	point := &pb.PointStruct{
		Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: vectorID}},
		Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: embedding}}},
		Payload: pbPayload,
	}
	
	// Upsert point
	_, err := ss.client.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: ss.collectionName,
		Wait:          &[]bool{true}[0], // Wait for indexing to complete
		Points:        []*pb.PointStruct{point},
	})
	
	if err != nil {
		return fmt.Errorf("failed to store email vector: %w", err)
	}
	
	return nil
}

// DeleteEmailVector removes an email vector from Qdrant
func (ss *SearchService) DeleteEmailVector(emailID string) error {
	if ss.client == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := ss.client.Delete(ctx, &pb.DeletePoints{
		CollectionName: ss.collectionName,
		Wait:          &[]bool{true}[0],
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{
						{PointIdOptions: &pb.PointId_Uuid{Uuid: emailID}},
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
func getStringFromPayload(payload map[string]*pb.Value, key string) string {
	if val, ok := payload[key]; ok {
		if strVal := val.GetStringValue(); strVal != "" {
			return strVal
		}
	}
	return ""
}

func getDoubleFromPayload(payload map[string]*pb.Value, key string) float64 {
	if val, ok := payload[key]; ok {
		return val.GetDoubleValue()
	}
	return 0
}

func getIntFromPayload(payload map[string]*pb.Value, key string) int64 {
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