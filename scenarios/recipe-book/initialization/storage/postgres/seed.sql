-- Recipe Book Seed Data

-- Insert sample recipes
INSERT INTO recipes (id, title, description, ingredients, instructions, prep_time, cook_time, servings, tags, cuisine, dietary_info, nutrition, visibility, source) VALUES
(
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    'Classic Chocolate Chip Cookies',
    'Soft, chewy chocolate chip cookies that everyone loves',
    '[
        {"name": "all-purpose flour", "amount": 2.25, "unit": "cups"},
        {"name": "baking soda", "amount": 1, "unit": "tsp"},
        {"name": "salt", "amount": 1, "unit": "tsp"},
        {"name": "butter", "amount": 1, "unit": "cup", "notes": "softened"},
        {"name": "granulated sugar", "amount": 0.75, "unit": "cup"},
        {"name": "brown sugar", "amount": 0.75, "unit": "cup", "notes": "packed"},
        {"name": "vanilla extract", "amount": 1, "unit": "tsp"},
        {"name": "eggs", "amount": 2, "unit": "large"},
        {"name": "chocolate chips", "amount": 2, "unit": "cups"}
    ]',
    '[
        "Preheat oven to 375°F (190°C)",
        "In a small bowl, mix flour, baking soda, and salt",
        "In a large bowl, beat butter and sugars until fluffy",
        "Add eggs and vanilla to butter mixture",
        "Gradually blend in flour mixture",
        "Stir in chocolate chips",
        "Drop rounded tablespoons onto ungreased cookie sheets",
        "Bake for 9-11 minutes or until golden brown",
        "Cool on baking sheet for 2 minutes before removing"
    ]',
    15,
    10,
    48,
    '["dessert", "cookies", "baking", "family-favorite"]',
    'American',
    '["vegetarian"]',
    '{"calories": 80, "protein": 1, "carbs": 10, "fat": 4, "fiber": 0, "sugar": 6, "sodium": 55}',
    'public',
    'original'
),
(
    'b2c3d4e5-f6a7-8901-bcde-f23456789012',
    'Quick Veggie Stir-Fry',
    'A healthy and colorful vegetable stir-fry ready in minutes',
    '[
        {"name": "broccoli florets", "amount": 2, "unit": "cups"},
        {"name": "bell pepper", "amount": 1, "unit": "large", "notes": "sliced"},
        {"name": "carrots", "amount": 2, "unit": "medium", "notes": "sliced"},
        {"name": "snap peas", "amount": 1, "unit": "cup"},
        {"name": "garlic", "amount": 3, "unit": "cloves", "notes": "minced"},
        {"name": "ginger", "amount": 1, "unit": "tbsp", "notes": "minced"},
        {"name": "soy sauce", "amount": 3, "unit": "tbsp"},
        {"name": "sesame oil", "amount": 2, "unit": "tsp"},
        {"name": "vegetable oil", "amount": 2, "unit": "tbsp"},
        {"name": "cornstarch", "amount": 1, "unit": "tsp"},
        {"name": "vegetable broth", "amount": 0.25, "unit": "cup"}
    ]',
    '[
        "Mix soy sauce, sesame oil, cornstarch, and broth in a small bowl",
        "Heat vegetable oil in a large wok or skillet over high heat",
        "Add garlic and ginger, stir-fry for 30 seconds",
        "Add broccoli and carrots, stir-fry for 3 minutes",
        "Add bell pepper and snap peas, stir-fry for 2 minutes",
        "Pour sauce over vegetables and toss to coat",
        "Cook for 1-2 minutes until sauce thickens",
        "Serve immediately over rice or noodles"
    ]',
    10,
    10,
    4,
    '["dinner", "healthy", "quick", "vegetarian", "asian"]',
    'Asian',
    '["vegetarian", "vegan", "gluten-free"]',
    '{"calories": 120, "protein": 4, "carbs": 18, "fat": 5, "fiber": 4, "sugar": 6, "sodium": 580}',
    'public',
    'original'
),
(
    'c3d4e5f6-a7b8-9012-cdef-345678901234',
    'Grandma''s Chicken Soup',
    'A comforting bowl of homemade chicken soup that heals the soul',
    '[
        {"name": "whole chicken", "amount": 1, "unit": "3-4 lbs"},
        {"name": "water", "amount": 12, "unit": "cups"},
        {"name": "onion", "amount": 1, "unit": "large", "notes": "quartered"},
        {"name": "carrots", "amount": 4, "unit": "medium", "notes": "sliced"},
        {"name": "celery stalks", "amount": 4, "unit": "stalks", "notes": "sliced"},
        {"name": "garlic", "amount": 4, "unit": "cloves"},
        {"name": "bay leaves", "amount": 2, "unit": "leaves"},
        {"name": "fresh thyme", "amount": 1, "unit": "tbsp"},
        {"name": "egg noodles", "amount": 2, "unit": "cups"},
        {"name": "salt", "amount": 2, "unit": "tsp"},
        {"name": "black pepper", "amount": 1, "unit": "tsp"},
        {"name": "fresh parsley", "amount": 0.25, "unit": "cup", "notes": "chopped"}
    ]',
    '[
        "Place chicken in a large pot with water, onion, 2 carrots, 2 celery stalks, garlic, and bay leaves",
        "Bring to a boil, then reduce heat and simmer for 1.5 hours",
        "Remove chicken and strain broth, discarding vegetables",
        "Return broth to pot and bring to a simmer",
        "Shred chicken meat and discard bones and skin",
        "Add remaining carrots and celery to broth, simmer 10 minutes",
        "Add egg noodles and cook 8-10 minutes until tender",
        "Add shredded chicken, thyme, salt, and pepper",
        "Simmer 5 more minutes",
        "Garnish with fresh parsley before serving"
    ]',
    30,
    120,
    8,
    '["soup", "comfort-food", "family-recipe", "healing"]',
    'American',
    '[]',
    '{"calories": 220, "protein": 18, "carbs": 20, "fat": 6, "fiber": 2, "sugar": 3, "sodium": 650}',
    'public',
    'original'
),
(
    'd4e5f6a7-b8c9-0123-defa-456789012345',
    'Mediterranean Quinoa Bowl',
    'A nutritious and flavorful grain bowl with Mediterranean flavors',
    '[
        {"name": "quinoa", "amount": 1, "unit": "cup", "notes": "uncooked"},
        {"name": "vegetable broth", "amount": 2, "unit": "cups"},
        {"name": "cucumber", "amount": 1, "unit": "medium", "notes": "diced"},
        {"name": "cherry tomatoes", "amount": 1, "unit": "cup", "notes": "halved"},
        {"name": "red onion", "amount": 0.25, "unit": "cup", "notes": "diced"},
        {"name": "kalamata olives", "amount": 0.5, "unit": "cup", "notes": "pitted"},
        {"name": "feta cheese", "amount": 0.5, "unit": "cup", "notes": "crumbled"},
        {"name": "chickpeas", "amount": 1, "unit": "can", "notes": "drained"},
        {"name": "lemon juice", "amount": 3, "unit": "tbsp"},
        {"name": "olive oil", "amount": 2, "unit": "tbsp"},
        {"name": "fresh dill", "amount": 2, "unit": "tbsp", "notes": "chopped"},
        {"name": "salt and pepper", "amount": 1, "unit": "to taste"}
    ]',
    '[
        "Rinse quinoa in cold water",
        "Bring vegetable broth to a boil in a saucepan",
        "Add quinoa, reduce heat to low, cover and simmer 15 minutes",
        "Remove from heat and let stand 5 minutes, then fluff with fork",
        "Let quinoa cool to room temperature",
        "In a large bowl, combine cooled quinoa with cucumber, tomatoes, onion, olives, and chickpeas",
        "Whisk together lemon juice, olive oil, dill, salt, and pepper",
        "Pour dressing over quinoa mixture and toss to combine",
        "Top with crumbled feta cheese",
        "Serve chilled or at room temperature"
    ]',
    15,
    20,
    4,
    '["lunch", "healthy", "mediterranean", "grain-bowl", "vegetarian"]',
    'Mediterranean',
    '["vegetarian", "gluten-free"]',
    '{"calories": 380, "protein": 14, "carbs": 48, "fat": 16, "fiber": 8, "sugar": 5, "sodium": 480}',
    'public',
    'original'
),
(
    'e5f6a7b8-c9d0-1234-efab-567890123456',
    'Spicy Black Bean Tacos',
    'Quick and delicious vegetarian tacos with a kick',
    '[
        {"name": "black beans", "amount": 2, "unit": "cans", "notes": "drained"},
        {"name": "corn tortillas", "amount": 8, "unit": "small"},
        {"name": "avocado", "amount": 2, "unit": "medium", "notes": "sliced"},
        {"name": "red cabbage", "amount": 2, "unit": "cups", "notes": "shredded"},
        {"name": "lime", "amount": 2, "unit": "whole"},
        {"name": "cilantro", "amount": 0.5, "unit": "cup", "notes": "chopped"},
        {"name": "jalapeno", "amount": 1, "unit": "pepper", "notes": "minced"},
        {"name": "cumin", "amount": 1, "unit": "tsp"},
        {"name": "chili powder", "amount": 1, "unit": "tsp"},
        {"name": "garlic powder", "amount": 0.5, "unit": "tsp"},
        {"name": "hot sauce", "amount": 2, "unit": "tbsp"},
        {"name": "sour cream", "amount": 0.5, "unit": "cup"}
    ]',
    '[
        "Heat black beans in a saucepan with cumin, chili powder, and garlic powder",
        "Mash about half the beans for a creamier texture",
        "Warm corn tortillas in a dry skillet or microwave",
        "Mix shredded cabbage with juice of 1 lime and a pinch of salt",
        "Spread beans on tortillas",
        "Top with cabbage slaw, avocado slices, and jalapeno",
        "Drizzle with hot sauce and sour cream",
        "Garnish with cilantro and serve with lime wedges"
    ]',
    10,
    10,
    4,
    '["dinner", "mexican", "vegetarian", "quick", "spicy"]',
    'Mexican',
    '["vegetarian", "gluten-free"]',
    '{"calories": 320, "protein": 12, "carbs": 42, "fat": 14, "fiber": 12, "sugar": 3, "sodium": 420}',
    'public',
    'original'
);

-- Insert sample user preferences
INSERT INTO user_preferences (user_id, dietary_restrictions, favorite_cuisines, cooking_skill_level, preferred_cooking_time) VALUES
('11111111-1111-1111-1111-111111111111', '["vegetarian"]', '["Italian", "Asian", "Mediterranean"]', 'intermediate', 45),
('22222222-2222-2222-2222-222222222222', '["gluten-free"]', '["Mexican", "American"]', 'beginner', 30);

-- Insert sample ratings
INSERT INTO recipe_ratings (id, recipe_id, user_id, rating, notes, anonymous) VALUES
('r1111111-1111-1111-1111-111111111111', 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', '11111111-1111-1111-1111-111111111111', 5, 'Best cookies ever!', false),
('r2222222-2222-2222-2222-222222222222', 'b2c3d4e5-f6a7-8901-bcde-f23456789012', '11111111-1111-1111-1111-111111111111', 4, 'Quick and healthy', false),
('r3333333-3333-3333-3333-333333333333', 'c3d4e5f6-a7b8-9012-cdef-345678901234', '22222222-2222-2222-2222-222222222222', 5, 'Just like grandma used to make', true);

-- Insert sample recipe collection
INSERT INTO recipe_collections (id, user_id, name, description, recipe_ids, visibility) VALUES
('col11111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Weeknight Dinners', 'Quick recipes for busy weeknights', 
'["b2c3d4e5-f6a7-8901-bcde-f23456789012", "e5f6a7b8-c9d0-1234-efab-567890123456"]', 'public');