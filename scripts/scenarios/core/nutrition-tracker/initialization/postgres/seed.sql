-- Seed data for Nutrition Tracker

-- Insert sample foods (common items)
INSERT INTO foods (name, serving_size, serving_unit, calories, protein, carbs, fat, fiber, sugar, sodium, food_category, tags, source) VALUES
('Apple', '182', 'g', 95, 0.5, 25, 0.3, 4.4, 19, 2, 'Fruits', '["fruit", "snack", "fiber", "vegan"]', 'usda'),
('Banana', '118', 'g', 105, 1.3, 27, 0.4, 3.1, 14, 1, 'Fruits', '["fruit", "potassium", "energy", "vegan"]', 'usda'),
('Chicken Breast (Grilled)', '100', 'g', 165, 31, 0, 3.6, 0, 0, 74, 'Protein', '["protein", "lean", "keto"]', 'usda'),
('Brown Rice (Cooked)', '195', 'g', 218, 5, 46, 2, 4, 0, 10, 'Grains', '["grain", "fiber", "vegan", "gluten-free"]', 'usda'),
('Broccoli (Steamed)', '156', 'g', 55, 3.7, 11, 0.6, 5, 2, 64, 'Vegetables', '["vegetable", "fiber", "vitamin-c", "vegan"]', 'usda'),
('Greek Yogurt (Plain)', '150', 'g', 100, 17, 6, 0.7, 0, 4, 70, 'Dairy', '["dairy", "protein", "probiotics", "vegetarian"]', 'usda'),
('Almonds', '28', 'g', 164, 6, 6, 14, 3.5, 1, 1, 'Nuts', '["nuts", "healthy-fats", "protein", "vegan"]', 'usda'),
('Salmon (Baked)', '100', 'g', 206, 22, 0, 13, 0, 0, 59, 'Protein', '["protein", "omega-3", "keto"]', 'usda'),
('Egg (Large)', '50', 'g', 72, 6, 0.4, 5, 0, 0.2, 71, 'Protein', '["protein", "breakfast", "keto", "vegetarian"]', 'usda'),
('Spinach (Raw)', '30', 'g', 7, 0.9, 1.1, 0.1, 0.7, 0.1, 24, 'Vegetables', '["vegetable", "iron", "low-calorie", "vegan"]', 'usda'),
('Oatmeal (Cooked)', '234', 'g', 158, 6, 27, 3, 4, 1, 115, 'Grains', '["grain", "breakfast", "fiber", "vegan"]', 'usda'),
('Avocado', '150', 'g', 240, 3, 13, 22, 10, 1, 11, 'Fruits', '["fruit", "healthy-fats", "keto", "vegan"]', 'usda'),
('Sweet Potato (Baked)', '200', 'g', 180, 4, 41, 0.3, 6.6, 13, 71, 'Vegetables', '["vegetable", "vitamin-a", "fiber", "vegan"]', 'usda'),
('Black Beans (Cooked)', '172', 'g', 227, 15, 41, 0.9, 15, 0.6, 2, 'Legumes', '["legume", "protein", "fiber", "vegan"]', 'usda'),
('Quinoa (Cooked)', '185', 'g', 222, 8, 39, 4, 5, 2, 13, 'Grains', '["grain", "protein", "gluten-free", "vegan"]', 'usda'),
('Blueberries', '148', 'g', 84, 1, 21, 0.5, 4, 15, 1, 'Fruits', '["fruit", "antioxidants", "snack", "vegan"]', 'usda'),
('Cottage Cheese (2%)', '226', 'g', 206, 27, 8, 5, 0, 8, 918, 'Dairy', '["dairy", "protein", "vegetarian"]', 'usda'),
('Olive Oil', '14', 'g', 119, 0, 0, 14, 0, 0, 0, 'Fats', '["fat", "healthy-fats", "cooking", "vegan", "keto"]', 'usda'),
('Whole Wheat Bread', '43', 'g', 109, 5, 18, 2, 3, 2, 170, 'Grains', '["grain", "fiber", "sandwich", "vegan"]', 'usda'),
('Milk (2%)', '244', 'ml', 122, 8, 12, 5, 0, 12, 100, 'Dairy', '["dairy", "calcium", "vegetarian"]', 'usda');

-- Insert sample user
INSERT INTO users (email, name) VALUES
('demo@nutrition-tracker.com', 'Demo User');

-- Get the user ID for reference
WITH demo_user AS (
    SELECT id FROM users WHERE email = 'demo@nutrition-tracker.com' LIMIT 1
)
-- Insert user goals
INSERT INTO user_goals (user_id, daily_calories, protein_grams, carbs_grams, fat_grams, fiber_grams, sugar_limit, sodium_limit, dietary_preferences, activity_level, weight_goal)
SELECT 
    id,
    2000,
    75,
    225,
    65,
    25,
    50,
    2300,
    '["balanced"]',
    'moderately_active',
    'maintain'
FROM demo_user;

-- Sample recipes
WITH demo_user AS (
    SELECT id FROM users WHERE email = 'demo@nutrition-tracker.com' LIMIT 1
)
INSERT INTO recipes (user_id, name, description, instructions, prep_time, cook_time, servings, cuisine, meal_type, tags, is_public, total_calories, total_protein, total_carbs, total_fat)
SELECT 
    id,
    'Overnight Oats',
    'Healthy and delicious breakfast prepared the night before',
    '1. Mix oats with milk in a jar\n2. Add chia seeds and honey\n3. Top with berries\n4. Refrigerate overnight\n5. Enjoy cold in the morning',
    5,
    0,
    1,
    'American',
    'breakfast',
    '["breakfast", "healthy", "meal-prep", "vegetarian"]',
    true,
    320,
    12,
    48,
    8
FROM demo_user;

WITH demo_user AS (
    SELECT id FROM users WHERE email = 'demo@nutrition-tracker.com' LIMIT 1
)
INSERT INTO recipes (user_id, name, description, instructions, prep_time, cook_time, servings, cuisine, meal_type, tags, is_public, total_calories, total_protein, total_carbs, total_fat)
SELECT 
    id,
    'Grilled Chicken Salad',
    'Light and protein-packed lunch option',
    '1. Season and grill chicken breast\n2. Mix greens, tomatoes, and cucumber\n3. Slice grilled chicken\n4. Add chicken to salad\n5. Drizzle with olive oil and lemon',
    10,
    15,
    1,
    'Mediterranean',
    'lunch',
    '["lunch", "salad", "protein", "low-carb", "keto-friendly"]',
    true,
    285,
    35,
    12,
    10
FROM demo_user;