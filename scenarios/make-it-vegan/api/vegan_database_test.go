package main

import (
	"testing"
)

// TestInitVeganDatabase tests database initialization
func TestInitVeganDatabase(t *testing.T) {
	t.Run("DatabaseInitialization", func(t *testing.T) {
		db := InitVeganDatabase()

		if db == nil {
			t.Fatal("Expected database to be initialized")
		}

		if db.NonVeganIngredients == nil {
			t.Error("Expected NonVeganIngredients to be initialized")
		}

		if db.VeganAlternatives == nil {
			t.Error("Expected VeganAlternatives to be initialized")
		}

		if db.CommonSubstitutes == nil {
			t.Error("Expected CommonSubstitutes to be initialized")
		}
	})

	t.Run("NonVeganIngredientsPopulated", func(t *testing.T) {
		db := InitVeganDatabase()

		expectedIngredients := []string{"milk", "eggs", "butter", "cheese", "chicken", "beef", "honey"}
		for _, ingredient := range expectedIngredients {
			if _, exists := db.NonVeganIngredients[ingredient]; !exists {
				t.Errorf("Expected ingredient '%s' to be in non-vegan list", ingredient)
			}
		}

		// Verify all have reasons
		for ingredient, reason := range db.NonVeganIngredients {
			if reason == "" {
				t.Errorf("Ingredient '%s' has no reason", ingredient)
			}
		}
	})

	t.Run("VeganAlternativesPopulated", func(t *testing.T) {
		db := InitVeganDatabase()

		expectedCategories := []string{"milk", "eggs", "butter", "cheese", "honey"}
		for _, category := range expectedCategories {
			if alts, exists := db.VeganAlternatives[category]; !exists {
				t.Errorf("Expected alternatives for '%s'", category)
			} else if len(alts) == 0 {
				t.Errorf("Expected non-empty alternatives for '%s'", category)
			}
		}

		// Verify alternative structure
		if alts, exists := db.VeganAlternatives["milk"]; exists && len(alts) > 0 {
			alt := alts[0]
			if alt.Name == "" {
				t.Error("Alternative name should not be empty")
			}
			if alt.Rating == 0 {
				t.Error("Alternative should have a rating")
			}
		}
	})

	t.Run("CommonSubstitutesPopulated", func(t *testing.T) {
		db := InitVeganDatabase()

		expectedSubstitutes := []string{"1 egg", "1 cup milk", "1 tbsp butter", "honey"}
		for _, sub := range expectedSubstitutes {
			if _, exists := db.CommonSubstitutes[sub]; !exists {
				t.Errorf("Expected common substitute for '%s'", sub)
			}
		}

		// Verify all have substitutes
		for original, substitute := range db.CommonSubstitutes {
			if substitute == "" {
				t.Errorf("Substitute for '%s' is empty", original)
			}
		}
	})
}

// TestCheckIngredientsLogic tests ingredient checking logic
func TestCheckIngredientsLogic(t *testing.T) {
	db := InitVeganDatabase()

	tests := []struct {
		name             string
		ingredients      string
		expectedVegan    bool
		expectedNonVegan int
		expectedReasons  int
		containsItems    []string
		notContainsItems []string
	}{
		{
			name:             "AllVeganIngredients",
			ingredients:      "flour, sugar, salt, water",
			expectedVegan:    true,
			expectedNonVegan: 0,
			expectedReasons:  0,
		},
		{
			name:             "AllNonVeganIngredients",
			ingredients:      "milk, eggs, butter",
			expectedVegan:    false,
			expectedNonVegan: 3,
			expectedReasons:  3,
			containsItems:    []string{"milk", "eggs", "butter"},
		},
		{
			name:             "MixedIngredients",
			ingredients:      "flour, milk, sugar, eggs",
			expectedVegan:    false,
			expectedNonVegan: 2,
			expectedReasons:  2,
			containsItems:    []string{"milk", "eggs"},
			notContainsItems: []string{"flour", "sugar"},
		},
		{
			name:             "VeganMilkException",
			ingredients:      "soy milk, almond milk, oat milk",
			expectedVegan:    true,
			expectedNonVegan: 0,
			expectedReasons:  0,
		},
		{
			name:             "VeganButterException",
			ingredients:      "peanut butter, almond butter, cashew butter",
			expectedVegan:    true,
			expectedNonVegan: 0,
			expectedReasons:  0,
		},
		{
			name:             "CoconutMilkException",
			ingredients:      "coconut milk, coconut butter",
			expectedVegan:    true,
			expectedNonVegan: 0,
			expectedReasons:  0,
		},
		{
			name:             "EmptyIngredients",
			ingredients:      "",
			expectedVegan:    true,
			expectedNonVegan: 0,
			expectedReasons:  0,
		},
		{
			name:             "SingleNonVeganItem",
			ingredients:      "cheese",
			expectedVegan:    false,
			expectedNonVegan: 1,
			expectedReasons:  1,
			containsItems:    []string{"cheese"},
		},
		{
			name:             "CaseInsensitivity",
			ingredients:      "MILK, Eggs, ButTer",
			expectedVegan:    false,
			expectedNonVegan: 3,
			expectedReasons:  3,
		},
		{
			name:             "WhitespaceHandling",
			ingredients:      "  flour  ,  milk  ,  sugar  ",
			expectedVegan:    false,
			expectedNonVegan: 1,
			expectedReasons:  1,
			// Note: CheckIngredients trims whitespace from ingredients
			// So "  milk  " becomes "milk" in the result
		},
		{
			name:             "DairyProducts",
			ingredients:      "yogurt, cream, whey, casein",
			expectedVegan:    false,
			expectedNonVegan: 4,
			expectedReasons:  4,
		},
		{
			name:             "MeatProducts",
			ingredients:      "chicken, beef, pork, fish",
			expectedVegan:    false,
			expectedNonVegan: 4,
			expectedReasons:  4,
		},
		{
			name:             "OtherAnimalProducts",
			ingredients:      "honey, gelatin, beeswax",
			expectedVegan:    false,
			expectedNonVegan: 3,
			expectedReasons:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isVegan, nonVeganItems, reasons := db.CheckIngredients(tt.ingredients)

			if isVegan != tt.expectedVegan {
				t.Errorf("Expected isVegan=%v, got %v", tt.expectedVegan, isVegan)
			}

			if len(nonVeganItems) != tt.expectedNonVegan {
				t.Errorf("Expected %d non-vegan items, got %d: %v", tt.expectedNonVegan, len(nonVeganItems), nonVeganItems)
			}

			if len(reasons) != tt.expectedReasons {
				t.Errorf("Expected %d reasons, got %d: %v", tt.expectedReasons, len(reasons), reasons)
			}

			// Check that expected items are present
			for _, item := range tt.containsItems {
				found := false
				for _, nonVegan := range nonVeganItems {
					if nonVegan == item {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find '%s' in non-vegan items, got %v", item, nonVeganItems)
				}
			}

			// Check that items should not be present
			for _, item := range tt.notContainsItems {
				for _, nonVegan := range nonVeganItems {
					if nonVegan == item {
						t.Errorf("Did not expect to find '%s' in non-vegan items", item)
					}
				}
			}

			// Verify reasons match items
			if len(nonVeganItems) != len(reasons) {
				t.Errorf("Number of items (%d) should match number of reasons (%d)", len(nonVeganItems), len(reasons))
			}
		})
	}
}

// TestGetAlternatives tests alternative retrieval
func TestGetAlternatives(t *testing.T) {
	db := InitVeganDatabase()

	tests := []struct {
		name               string
		ingredient         string
		expectAlternatives bool
		minAlternatives    int
	}{
		{
			name:               "MilkAlternatives",
			ingredient:         "milk",
			expectAlternatives: true,
			minAlternatives:    1,
		},
		{
			name:               "EggsAlternatives",
			ingredient:         "eggs",
			expectAlternatives: true,
			minAlternatives:    1,
		},
		{
			name:               "ButterAlternatives",
			ingredient:         "butter",
			expectAlternatives: true,
			minAlternatives:    1,
		},
		{
			name:               "CheeseAlternatives",
			ingredient:         "cheese",
			expectAlternatives: true,
			minAlternatives:    1,
		},
		{
			name:               "HoneyAlternatives",
			ingredient:         "honey",
			expectAlternatives: true,
			minAlternatives:    1,
		},
		{
			name:               "UnknownIngredient",
			ingredient:         "unknown-ingredient",
			expectAlternatives: false,
			minAlternatives:    0,
		},
		{
			name:               "CaseInsensitive",
			ingredient:         "MILK",
			expectAlternatives: true,
			minAlternatives:    1,
		},
		{
			name:               "WithWhitespace",
			ingredient:         "  eggs  ",
			expectAlternatives: true,
			minAlternatives:    1,
		},
		{
			name:               "PartialMatch",
			ingredient:         "whole milk",
			expectAlternatives: true,
			minAlternatives:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alternatives := db.GetAlternatives(tt.ingredient)

			if tt.expectAlternatives {
				if len(alternatives) < tt.minAlternatives {
					t.Errorf("Expected at least %d alternatives, got %d", tt.minAlternatives, len(alternatives))
				}

				// Verify alternative structure
				for i, alt := range alternatives {
					if alt.Name == "" {
						t.Errorf("Alternative %d has empty name", i)
					}
					if alt.Rating == 0 {
						t.Errorf("Alternative %d (%s) has zero rating", i, alt.Name)
					}
					if alt.Rating < 0 || alt.Rating > 5 {
						t.Errorf("Alternative %d (%s) has invalid rating: %f", i, alt.Name, alt.Rating)
					}
				}
			} else {
				if len(alternatives) != 0 {
					t.Errorf("Expected no alternatives for '%s', got %d", tt.ingredient, len(alternatives))
				}
			}
		})
	}
}

// TestGetQuickSubstitute tests quick substitute retrieval
func TestGetQuickSubstitute(t *testing.T) {
	db := InitVeganDatabase()

	tests := []struct {
		name             string
		ingredient       string
		expectSubstitute bool
	}{
		{
			name:             "OneEggSubstitute",
			ingredient:       "1 egg",
			expectSubstitute: true,
		},
		{
			name:             "OneCupMilkSubstitute",
			ingredient:       "1 cup milk",
			expectSubstitute: true,
		},
		{
			name:             "OneTablespoonButterSubstitute",
			ingredient:       "1 tbsp butter",
			expectSubstitute: true,
		},
		{
			name:             "HoneySubstitute",
			ingredient:       "honey",
			expectSubstitute: true,
		},
		{
			name:             "UnknownIngredient",
			ingredient:       "unknown-ingredient",
			expectSubstitute: false,
		},
		{
			name:             "CaseInsensitive",
			ingredient:       "1 EGG",
			expectSubstitute: true,
		},
		{
			name:             "PartialMatch",
			ingredient:       "egg",
			expectSubstitute: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			substitute := db.GetQuickSubstitute(tt.ingredient)

			if tt.expectSubstitute {
				if substitute == "" || substitute == "No direct substitute found - check alternatives list" {
					t.Errorf("Expected substitute for '%s', got '%s'", tt.ingredient, substitute)
				}
			} else {
				if substitute != "No direct substitute found - check alternatives list" {
					t.Errorf("Expected no substitute message for '%s', got '%s'", tt.ingredient, substitute)
				}
			}
		})
	}
}

// TestGetNutritionalInsights tests nutritional information retrieval
func TestGetNutritionalInsights(t *testing.T) {
	db := InitVeganDatabase()

	t.Run("NutritionalInfoComplete", func(t *testing.T) {
		info := db.GetNutritionalInsights()

		if info.Protein == "" {
			t.Error("Expected protein information")
		}
		if info.B12 == "" {
			t.Error("Expected B12 information")
		}
		if info.Iron == "" {
			t.Error("Expected iron information")
		}
		if info.Calcium == "" {
			t.Error("Expected calcium information")
		}
		if info.Omega3 == "" {
			t.Error("Expected omega-3 information")
		}
		if len(info.Considerations) == 0 {
			t.Error("Expected considerations to be populated")
		}
		if len(info.GoodSources) == 0 {
			t.Error("Expected good sources to be populated")
		}
	})

	t.Run("NutritionalInfoConsiderations", func(t *testing.T) {
		info := db.GetNutritionalInsights()

		// Verify minimum number of considerations
		if len(info.Considerations) < 3 {
			t.Errorf("Expected at least 3 considerations, got %d", len(info.Considerations))
		}

		// Verify no empty considerations
		for i, consideration := range info.Considerations {
			if consideration == "" {
				t.Errorf("Consideration %d is empty", i)
			}
		}
	})

	t.Run("NutritionalInfoGoodSources", func(t *testing.T) {
		info := db.GetNutritionalInsights()

		// Verify minimum number of good sources
		if len(info.GoodSources) < 3 {
			t.Errorf("Expected at least 3 good sources, got %d", len(info.GoodSources))
		}

		// Verify no empty sources
		for i, source := range info.GoodSources {
			if source == "" {
				t.Errorf("Good source %d is empty", i)
			}
		}

		// Verify sources contain key nutrients
		expectedNutrients := []string{"Protein", "Iron", "Calcium", "Omega-3", "B12"}
		for _, nutrient := range expectedNutrients {
			found := false
			for _, source := range info.GoodSources {
				if len(source) >= len(nutrient) && source[:len(nutrient)] == nutrient {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find sources for %s", nutrient)
			}
		}
	})
}

// TestAlternativeStructure tests the Alternative struct fields
func TestAlternativeStructure(t *testing.T) {
	db := InitVeganDatabase()

	t.Run("AlternativeFieldsPopulated", func(t *testing.T) {
		// Test milk alternatives as they're well-populated
		if alts, exists := db.VeganAlternatives["milk"]; exists {
			for i, alt := range alts {
				if alt.Name == "" {
					t.Errorf("Alternative %d: Name is empty", i)
				}
				if alt.Rating == 0 {
					t.Errorf("Alternative %d (%s): Rating is zero", i, alt.Name)
				}
				if alt.Rating < 0 || alt.Rating > 5 {
					t.Errorf("Alternative %d (%s): Invalid rating %f (must be 0-5)", i, alt.Name, alt.Rating)
				}
				// Notes field is optional but should be reasonable
				if alt.Notes == "" {
					t.Errorf("Alternative %d (%s): Notes field is empty", i, alt.Name)
				}
			}
		} else {
			t.Error("Expected milk alternatives to exist")
		}
	})

	t.Run("AlternativesHaveUniqueNames", func(t *testing.T) {
		for category, alternatives := range db.VeganAlternatives {
			names := make(map[string]bool)
			for _, alt := range alternatives {
				if names[alt.Name] {
					t.Errorf("Category %s has duplicate alternative name: %s", category, alt.Name)
				}
				names[alt.Name] = true
			}
		}
	})
}

// TestEdgeCases tests edge cases in ingredient checking
func TestEdgeCases(t *testing.T) {
	db := InitVeganDatabase()

	tests := []struct {
		name          string
		ingredients   string
		expectedVegan bool
	}{
		{
			name:          "OnlyCommas",
			ingredients:   ",,,,",
			expectedVegan: true,
		},
		{
			name:          "ExtraWhitespace",
			ingredients:   "   flour   ,   sugar   ,   salt   ",
			expectedVegan: true,
		},
		{
			name:          "SingleComma",
			ingredients:   ",",
			expectedVegan: true,
		},
		{
			name:          "TrailingComma",
			ingredients:   "flour, sugar,",
			expectedVegan: true,
		},
		{
			name:          "LeadingComma",
			ingredients:   ",flour, sugar",
			expectedVegan: true,
		},
		{
			name:          "MixedCaseVeganException",
			ingredients:   "SOY MILK, ALMOND MILK",
			expectedVegan: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isVegan, _, _ := db.CheckIngredients(tt.ingredients)
			if isVegan != tt.expectedVegan {
				t.Errorf("Expected isVegan=%v for '%s', got %v", tt.expectedVegan, tt.ingredients, isVegan)
			}
		})
	}
}
