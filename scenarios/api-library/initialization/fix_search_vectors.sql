-- Fix search vectors for existing APIs
UPDATE apis
SET search_vector = 
    setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(provider, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(description, '')), 'C') ||
    setweight(to_tsvector('english', coalesce(array_to_string(tags, ' '), '')), 'D');

-- Verify the update
SELECT name, description, search_vector FROM apis LIMIT 5;