package main

import (
	"strings"
)

// VeganDatabase holds information about ingredients
type VeganDatabase struct {
	NonVeganIngredients map[string]string // ingredient -> reason
	VeganAlternatives   map[string][]Alternative
	CommonSubstitutes   map[string]string
}

// Alternative represents a vegan substitute
type Alternative struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	BestFor      string  `json:"bestFor"`
	Adjustments  string  `json:"adjustments"`
	Availability string  `json:"availability"`
	Rating       float64 `json:"rating"`
	Notes        string  `json:"notes"`
}

// InitVeganDatabase creates and populates the vegan database
func InitVeganDatabase() *VeganDatabase {
	db := &VeganDatabase{
		NonVeganIngredients: make(map[string]string),
		VeganAlternatives:   make(map[string][]Alternative),
		CommonSubstitutes:   make(map[string]string),
	}

	// Populate non-vegan ingredients
	db.NonVeganIngredients = map[string]string{
		// Dairy
		"milk":        "Dairy product from cows",
		"cheese":      "Dairy product containing milk",
		"butter":      "Dairy fat from milk",
		"yogurt":      "Cultured dairy product",
		"cream":       "Dairy product high in fat",
		"whey":        "Milk protein byproduct",
		"casein":      "Milk protein",
		"lactose":     "Milk sugar",
		"ghee":        "Clarified butter",
		"ice cream":   "Frozen dairy dessert",
		
		// Eggs
		"eggs":        "Animal product from chickens",
		"egg whites":  "Protein from eggs",
		"egg yolks":   "Fat and nutrients from eggs",
		"albumin":     "Egg white protein",
		"mayonnaise":  "Contains eggs",
		"meringue":    "Made with egg whites",
		
		// Meat & Fish
		"chicken":     "Poultry meat",
		"beef":        "Cow meat",
		"pork":        "Pig meat",
		"fish":        "Sea animal",
		"lamb":        "Sheep meat",
		"turkey":      "Poultry meat",
		"bacon":       "Cured pork",
		"gelatin":     "Animal collagen",
		"anchovy":     "Small fish",
		"shellfish":   "Sea creatures",
		
		// Other Animal Products
		"honey":       "Bee product",
		"beeswax":     "Produced by bees",
		"shellac":     "Insect secretion",
		"carmine":     "Red dye from insects",
		"vitamin d3":  "Often from sheep's wool",
		"omega-3":     "Often from fish oil",
		"lanolin":     "From sheep's wool",
	}

	// Populate vegan alternatives
	db.VeganAlternatives = map[string][]Alternative{
		"milk": {
			{Name: "Oat milk", Description: "Creamy, neutral flavor", BestFor: "Coffee, cereal, baking", Adjustments: "None needed", Availability: "Widely available", Rating: 5.0, Notes: "Most versatile dairy milk alternative"},
			{Name: "Soy milk", Description: "High protein, versatile", BestFor: "All purposes", Adjustments: "None needed", Availability: "Widely available", Rating: 5.0, Notes: "Closest nutritional match to dairy milk"},
			{Name: "Almond milk", Description: "Light, nutty flavor", BestFor: "Smoothies, cereal", Adjustments: "Less protein than dairy", Availability: "Widely available", Rating: 4.0, Notes: "Lower calorie option"},
			{Name: "Coconut milk", Description: "Rich and creamy", BestFor: "Curries, desserts", Adjustments: "Strong flavor", Availability: "Widely available", Rating: 4.0, Notes: "Best for rich dishes"},
		},
		"eggs": {
			{Name: "Flax eggs", Description: "1 tbsp ground flax + 3 tbsp water", BestFor: "Baking (binding)", Adjustments: "Let sit 5 minutes", Availability: "Easy to make", Rating: 4.0, Notes: "Great binding agent for baking"},
			{Name: "Chia eggs", Description: "1 tbsp chia seeds + 3 tbsp water", BestFor: "Baking (binding)", Adjustments: "Let sit 15 minutes", Availability: "Easy to make", Rating: 4.0, Notes: "Similar to flax eggs with added nutrients"},
			{Name: "Aquafaba", Description: "Chickpea liquid", BestFor: "Meringues, mayo", Adjustments: "Whips like egg whites", Availability: "From canned chickpeas", Rating: 5.0, Notes: "Amazing for foam and whipped textures"},
			{Name: "Tofu scramble", Description: "Crumbled firm tofu", BestFor: "Scrambled eggs", Adjustments: "Season with turmeric, nutritional yeast", Availability: "Widely available", Rating: 5.0, Notes: "Perfect scrambled egg substitute"},
			{Name: "JUST Egg", Description: "Mung bean-based liquid", BestFor: "Scrambles, omelets", Adjustments: "Cooks like eggs", Availability: "Grocery stores", Rating: 5.0, Notes: "Most realistic egg substitute"},
		},
		"butter": {
			{Name: "Vegan butter", Description: "Plant-based spread", BestFor: "All purposes", Adjustments: "1:1 replacement", Availability: "Widely available", Rating: 5.0, Notes: "Identical usage to dairy butter"},
			{Name: "Coconut oil", Description: "Solid at room temp", BestFor: "Baking, cooking", Adjustments: "1:1 replacement", Availability: "Widely available", Rating: 4.0, Notes: "Great for flaky pastries"},
			{Name: "Olive oil", Description: "Liquid fat", BestFor: "Cooking, some baking", Adjustments: "Use 3/4 amount", Availability: "Widely available", Rating: 4.0, Notes: "Healthier option with fruity flavor"},
		},
		"cheese": {
			{Name: "Nutritional yeast", Description: "Cheesy, nutty flavor", BestFor: "Seasoning, sauces", Adjustments: "Not a direct substitute", Availability: "Health food stores", Rating: 4.0, Notes: "Adds umami and B vitamins"},
			{Name: "Cashew cheese", Description: "Creamy, rich", BestFor: "Spreads, sauces", Adjustments: "Soak cashews first", Availability: "Make at home", Rating: 5.0, Notes: "Most authentic texture when homemade"},
			{Name: "Vegan cheese", Description: "Various brands available", BestFor: "Melting, slicing", Adjustments: "1:1 replacement", Availability: "Increasingly common", Rating: 4.0, Notes: "Quality varies by brand"},
		},
		"honey": {
			{Name: "Maple syrup", Description: "Tree sap sweetener", BestFor: "All purposes", Adjustments: "Slightly thinner", Availability: "Widely available", Rating: 5.0, Notes: "Natural sweetener with minerals"},
			{Name: "Agave nectar", Description: "Plant-based syrup", BestFor: "Beverages, baking", Adjustments: "Sweeter than honey", Availability: "Widely available", Rating: 4.0, Notes: "Lower glycemic index"},
			{Name: "Date syrup", Description: "Made from dates", BestFor: "Baking, drizzling", Adjustments: "Rich flavor", Availability: "Health food stores", Rating: 4.0, Notes: "Whole food sweetener with fiber"},
		},
	}

	// Common quick substitutes
	db.CommonSubstitutes = map[string]string{
		"1 egg":          "1 flax egg (1 tbsp ground flax + 3 tbsp water)",
		"1 cup milk":     "1 cup plant milk (soy, oat, almond)",
		"1 tbsp butter":  "1 tbsp vegan butter or coconut oil",
		"1 cup yogurt":   "1 cup coconut or soy yogurt",
		"1 cup cream":    "1 cup full-fat coconut milk",
		"gelatin":        "agar agar (use same amount)",
		"honey":          "maple syrup or agave (use same amount)",
	}

	return db
}

// CheckIngredients analyzes if ingredients are vegan
func (db *VeganDatabase) CheckIngredients(ingredients string) (bool, []string, []string) {
	ingredientList := strings.Split(strings.ToLower(ingredients), ",")
	var nonVeganFound []string
	var reasons []string

	// Known vegan exceptions that contain non-vegan words
	veganExceptions := map[string]bool{
		"soy milk":      true,
		"almond milk":   true,
		"oat milk":      true,
		"rice milk":     true,
		"coconut milk":  true,
		"almond butter": true,
		"peanut butter": true,
		"cashew butter": true,
		"sunflower butter": true,
		"coconut butter": true,
		"cocoa butter":  true,
		"shea butter":   true,
	}

	for _, ingredient := range ingredientList {
		ingredient = strings.TrimSpace(ingredient)
		
		// Check if it's a known vegan exception
		isException := false
		for exception := range veganExceptions {
			if strings.Contains(ingredient, exception) {
				isException = true
				break
			}
		}
		
		if !isException {
			// Check against non-vegan ingredients
			for nonVegan, reason := range db.NonVeganIngredients {
				// Use word boundary matching for better accuracy
				if ingredient == nonVegan || 
				   strings.HasPrefix(ingredient, nonVegan + " ") ||
				   strings.HasSuffix(ingredient, " " + nonVegan) ||
				   strings.Contains(ingredient, " " + nonVegan + " ") {
					nonVeganFound = append(nonVeganFound, ingredient)
					reasons = append(reasons, reason)
					break
				}
			}
		}
	}

	isVegan := len(nonVeganFound) == 0
	return isVegan, nonVeganFound, reasons
}

// GetAlternatives returns vegan alternatives for an ingredient
func (db *VeganDatabase) GetAlternatives(ingredient string) []Alternative {
	ingredient = strings.ToLower(strings.TrimSpace(ingredient))
	
	// Check direct match
	if alts, exists := db.VeganAlternatives[ingredient]; exists {
		return alts
	}

	// Check if ingredient contains any known non-vegan item
	for key, alts := range db.VeganAlternatives {
		if strings.Contains(ingredient, key) {
			return alts
		}
	}

	return []Alternative{}
}

// GetQuickSubstitute returns a simple substitution
func (db *VeganDatabase) GetQuickSubstitute(ingredient string) string {
	ingredient = strings.ToLower(strings.TrimSpace(ingredient))
	
	if sub, exists := db.CommonSubstitutes[ingredient]; exists {
		return sub
	}

	// Check partial matches
	for key, sub := range db.CommonSubstitutes {
		if strings.Contains(ingredient, key) || strings.Contains(key, ingredient) {
			return sub
		}
	}

	return "No direct substitute found - check alternatives list"
}