// +build testing


package main

import (
	"testing"
)

// TestGenerateRecipeEmbedding tests embedding generation
func TestGenerateRecipeEmbedding(t *testing.T) {
	recipe := Recipe{
		ID:    "test-id",
		Title: "Test Recipe",
	}

	// Should not panic
	generateRecipeEmbedding(recipe)
}

// TestDeleteRecipeEmbedding tests embedding deletion
func TestDeleteRecipeEmbedding(t *testing.T) {
	// Should not panic
	deleteRecipeEmbedding("test-id")
}

// TestPerformSemanticSearch tests semantic search
func TestPerformSemanticSearch(t *testing.T) {
	req := SearchRequest{
		Query:  "test query",
		UserID: "test-user",
		Limit:  10,
	}

	results := performSemanticSearch(req)
	if results == nil {
		t.Error("Expected non-nil results")
	}
}

// TestInterpretQuery tests query interpretation
func TestInterpretQuery(t *testing.T) {
	result := interpretQuery("chocolate cake")
	if result == "" {
		t.Error("Expected non-empty interpretation")
	}
}

// TestGenerateRecipeWithAI tests AI recipe generation
func TestGenerateRecipeWithAI(t *testing.T) {
	req := GenerateRequest{
		Prompt: "healthy breakfast",
		UserID: "test-user",
	}

	recipe := generateRecipeWithAI(req)
	if recipe.ID == "" {
		t.Error("Expected generated recipe to have ID")
	}
	if recipe.Source != "ai_generated" {
		t.Errorf("Expected source to be 'ai_generated', got %s", recipe.Source)
	}
}

// TestFetchRecipe tests recipe fetching
func TestFetchRecipe(t *testing.T) {
	recipe := fetchRecipe("test-id")
	if recipe.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", recipe.ID)
	}
}

// TestModifyRecipeWithAI tests AI recipe modification
func TestModifyRecipeWithAI(t *testing.T) {
	original := Recipe{
		ID:    "original-id",
		Title: "Original Recipe",
	}

	req := ModifyRequest{
		ModificationType: "make_vegan",
		UserID:           "test-user",
	}

	modified := modifyRecipeWithAI(original, req)
	if modified.ParentID != original.ID {
		t.Errorf("Expected parent_id to be %s, got %s", original.ID, modified.ParentID)
	}
	if modified.Source != "modified" {
		t.Errorf("Expected source to be 'modified', got %s", modified.Source)
	}
}

// TestDescribeChanges tests change description
func TestDescribeChanges(t *testing.T) {
	original := Recipe{Title: "Original"}
	modified := Recipe{Title: "Modified", Source: "modified"}

	changes := describeChanges(original, modified)
	if len(changes) == 0 {
		t.Error("Expected non-empty changes list")
	}
}

// TestAggregateIngredients tests ingredient aggregation
func TestAggregateIngredients(t *testing.T) {
	recipeIDs := []string{"id1", "id2"}

	ingredients := aggregateIngredients(recipeIDs)
	if ingredients == nil {
		t.Error("Expected non-nil ingredients list")
	}
}

// TestOrganizeByCategory tests ingredient categorization
func TestOrganizeByCategory(t *testing.T) {
	ingredients := []Ingredient{
		{Name: "flour", Amount: 2, Unit: "cups"},
		{Name: "eggs", Amount: 3, Unit: "whole"},
	}

	organized := organizeByCategory(ingredients)
	if organized == nil {
		t.Error("Expected non-nil organized map")
	}
}

// TestFetchUserPreferences tests user preferences fetching
func TestFetchUserPreferences(t *testing.T) {
	prefs := fetchUserPreferences("test-user")
	if prefs == nil {
		t.Error("Expected non-nil preferences")
	}

	if _, ok := prefs["dietary_restrictions"]; !ok {
		t.Error("Expected dietary_restrictions field")
	}
	if _, ok := prefs["favorite_cuisines"]; !ok {
		t.Error("Expected favorite_cuisines field")
	}
}

// TestUpdateUserPreferences tests user preferences updating
func TestUpdateUserPreferences(t *testing.T) {
	prefs := map[string]interface{}{
		"dietary_restrictions": []string{"vegetarian"},
		"favorite_cuisines":    []string{"Italian"},
	}

	// Should not panic
	updateUserPreferences("test-user", prefs)
}

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	// Save original db
	originalDB := db

	// Test with missing environment variables
	// This should not panic, but may fail to connect
	initDB()

	// Restore original db
	db = originalDB
}

// TestRecipeStructs tests recipe data structures
func TestRecipeStructs(t *testing.T) {
	// Test Ingredient struct
	ingredient := Ingredient{
		Name:   "flour",
		Amount: 2.5,
		Unit:   "cups",
		Notes:  "sifted",
	}

	if ingredient.Name != "flour" {
		t.Errorf("Expected name 'flour', got %s", ingredient.Name)
	}

	// Test NutritionInfo struct
	nutrition := NutritionInfo{
		Calories: 250,
		Protein:  8,
		Carbs:    35,
		Fat:      10,
		Fiber:    2,
		Sugar:    5,
		Sodium:   300,
	}

	if nutrition.Calories != 250 {
		t.Errorf("Expected calories 250, got %d", nutrition.Calories)
	}

	// Test RecipeRating struct
	rating := RecipeRating{
		ID:        "rating-id",
		RecipeID:  "recipe-id",
		UserID:    "user-id",
		Rating:    5,
		Notes:     "Delicious!",
		Anonymous: false,
	}

	if rating.Rating != 5 {
		t.Errorf("Expected rating 5, got %d", rating.Rating)
	}
}

// TestSearchRequest tests search request struct
func TestSearchRequest(t *testing.T) {
	req := SearchRequest{
		Query:  "pasta",
		UserID: "user-id",
		Limit:  10,
	}

	req.Filters.Dietary = []string{"vegetarian"}
	req.Filters.MaxTime = 30
	req.Filters.Ingredients = []string{"tomatoes"}

	if req.Query != "pasta" {
		t.Errorf("Expected query 'pasta', got %s", req.Query)
	}

	if len(req.Filters.Dietary) != 1 {
		t.Errorf("Expected 1 dietary filter, got %d", len(req.Filters.Dietary))
	}
}

// TestGenerateRequest tests generate request struct
func TestGenerateRequest(t *testing.T) {
	req := GenerateRequest{
		Prompt:              "healthy breakfast",
		UserID:              "user-id",
		DietaryRestrictions: []string{"vegan"},
		AvailableIngredients: []string{"oats", "bananas"},
		Style:               "quick and easy",
	}

	if req.Prompt != "healthy breakfast" {
		t.Errorf("Expected prompt 'healthy breakfast', got %s", req.Prompt)
	}

	if len(req.DietaryRestrictions) != 1 {
		t.Errorf("Expected 1 dietary restriction, got %d", len(req.DietaryRestrictions))
	}
}

// TestModifyRequest tests modify request struct
func TestModifyRequest(t *testing.T) {
	req := ModifyRequest{
		ModificationType: "make_vegan",
		UserID:           "user-id",
	}

	if req.ModificationType != "make_vegan" {
		t.Errorf("Expected modification type 'make_vegan', got %s", req.ModificationType)
	}
}
