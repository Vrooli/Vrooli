-- Recipe Book Database Schema

-- Create database if not exists
CREATE DATABASE IF NOT EXISTS recipe_book;

-- Create recipes table
CREATE TABLE IF NOT EXISTS recipes (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    ingredients JSONB NOT NULL DEFAULT '[]',
    instructions JSONB NOT NULL DEFAULT '[]',
    prep_time INTEGER DEFAULT 0,
    cook_time INTEGER DEFAULT 0,
    servings INTEGER DEFAULT 4,
    tags JSONB DEFAULT '[]',
    cuisine VARCHAR(100),
    dietary_info JSONB DEFAULT '[]',
    nutrition JSONB DEFAULT '{}',
    photo_url TEXT,
    created_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    visibility VARCHAR(20) DEFAULT 'private' CHECK (visibility IN ('public', 'private', 'shared')),
    shared_with JSONB DEFAULT '[]',
    source VARCHAR(50) DEFAULT 'original' CHECK (source IN ('original', 'ai_generated', 'modified', 'imported')),
    parent_recipe_id UUID REFERENCES recipes(id) ON DELETE SET NULL
);

-- Create indexes for better query performance
CREATE INDEX idx_recipes_created_by ON recipes(created_by);
CREATE INDEX idx_recipes_visibility ON recipes(visibility);
CREATE INDEX idx_recipes_cuisine ON recipes(cuisine);
CREATE INDEX idx_recipes_created_at ON recipes(created_at DESC);
CREATE INDEX idx_recipes_tags ON recipes USING GIN(tags);
CREATE INDEX idx_recipes_dietary ON recipes USING GIN(dietary_info);
CREATE INDEX idx_recipes_shared_with ON recipes USING GIN(shared_with);

-- Create recipe ratings table
CREATE TABLE IF NOT EXISTS recipe_ratings (
    id UUID PRIMARY KEY,
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    notes TEXT,
    cooked_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    anonymous BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for ratings
CREATE INDEX idx_ratings_recipe ON recipe_ratings(recipe_id);
CREATE INDEX idx_ratings_user ON recipe_ratings(user_id);
CREATE INDEX idx_ratings_cooked_date ON recipe_ratings(cooked_date DESC);

-- Create user preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id UUID PRIMARY KEY,
    dietary_restrictions JSONB DEFAULT '[]',
    favorite_cuisines JSONB DEFAULT '[]',
    disliked_ingredients JSONB DEFAULT '[]',
    cooking_skill_level VARCHAR(20) DEFAULT 'intermediate',
    preferred_cooking_time INTEGER DEFAULT 60,
    household_size INTEGER DEFAULT 2,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create meal plans table
CREATE TABLE IF NOT EXISTS meal_plans (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    recipes JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for meal plans
CREATE INDEX idx_meal_plans_user ON meal_plans(user_id);
CREATE INDEX idx_meal_plans_dates ON meal_plans(start_date, end_date);

-- Create shopping lists table
CREATE TABLE IF NOT EXISTS shopping_lists (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255),
    items JSONB NOT NULL DEFAULT '[]',
    recipe_ids JSONB DEFAULT '[]',
    completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for shopping lists
CREATE INDEX idx_shopping_lists_user ON shopping_lists(user_id);
CREATE INDEX idx_shopping_lists_completed ON shopping_lists(completed);

-- Create recipe collections table (for organizing recipes)
CREATE TABLE IF NOT EXISTS recipe_collections (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    recipe_ids JSONB DEFAULT '[]',
    visibility VARCHAR(20) DEFAULT 'private' CHECK (visibility IN ('public', 'private', 'shared')),
    shared_with JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for collections
CREATE INDEX idx_collections_user ON recipe_collections(user_id);
CREATE INDEX idx_collections_visibility ON recipe_collections(visibility);

-- Create cooking history table
CREATE TABLE IF NOT EXISTS cooking_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    cooked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    servings_made INTEGER,
    notes TEXT,
    modifications JSONB DEFAULT '[]'
);

-- Create indexes for cooking history
CREATE INDEX idx_cooking_history_user ON cooking_history(user_id);
CREATE INDEX idx_cooking_history_recipe ON cooking_history(recipe_id);
CREATE INDEX idx_cooking_history_date ON cooking_history(cooked_at DESC);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_recipes_updated_at BEFORE UPDATE ON recipes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON user_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_meal_plans_updated_at BEFORE UPDATE ON meal_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_shopping_lists_updated_at BEFORE UPDATE ON shopping_lists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_recipe_collections_updated_at BEFORE UPDATE ON recipe_collections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create view for recipe statistics
CREATE OR REPLACE VIEW recipe_stats AS
SELECT 
    r.id,
    r.title,
    COUNT(DISTINCT rr.id) as total_ratings,
    AVG(rr.rating) as average_rating,
    COUNT(DISTINCT ch.id) as times_cooked,
    MAX(ch.cooked_at) as last_cooked
FROM recipes r
LEFT JOIN recipe_ratings rr ON r.id = rr.recipe_id
LEFT JOIN cooking_history ch ON r.id = ch.recipe_id
GROUP BY r.id, r.title;

-- Create view for popular recipes
CREATE OR REPLACE VIEW popular_recipes AS
SELECT 
    r.*,
    rs.average_rating,
    rs.total_ratings,
    rs.times_cooked
FROM recipes r
JOIN recipe_stats rs ON r.id = rs.id
WHERE r.visibility = 'public'
    AND rs.average_rating >= 4.0
    AND rs.total_ratings >= 3
ORDER BY rs.average_rating DESC, rs.times_cooked DESC;