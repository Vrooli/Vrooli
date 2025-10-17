-- News Aggregator & Bias Analysis Database Schema

-- Create database if it doesn't exist
CREATE DATABASE IF NOT EXISTS news_aggregator;

-- Create feeds table for RSS sources
CREATE TABLE IF NOT EXISTS feeds (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL UNIQUE,
    category VARCHAR(100),
    bias_rating VARCHAR(50),
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create articles table
CREATE TABLE IF NOT EXISTS articles (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    url VARCHAR(500) NOT NULL,
    source VARCHAR(255),
    category VARCHAR(100),
    published_at TIMESTAMP,
    summary TEXT,
    content TEXT,
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    bias_score INTEGER DEFAULT 50,
    bias_direction VARCHAR(50) DEFAULT 'unknown',
    bias_analysis TEXT,
    perspective_count INTEGER DEFAULT 1,
    analyzed_at TIMESTAMP,
    UNIQUE(url)
);

-- Create user_preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) UNIQUE,
    preferred_sources TEXT[],
    blocked_sources TEXT[],
    preferred_categories TEXT[],
    bias_balance_preference VARCHAR(50) DEFAULT 'balanced',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create article_interactions table for tracking user engagement
CREATE TABLE IF NOT EXISTS article_interactions (
    id SERIAL PRIMARY KEY,
    article_id VARCHAR(255) REFERENCES articles(id),
    user_id VARCHAR(255),
    interaction_type VARCHAR(50), -- 'view', 'share', 'save', 'report'
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create topics table for tracking trending topics
CREATE TABLE IF NOT EXISTS topics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    article_count INTEGER DEFAULT 0,
    last_analyzed TIMESTAMP,
    perspectives_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create fact_checks table for storing fact-checking results
CREATE TABLE IF NOT EXISTS fact_checks (
    id SERIAL PRIMARY KEY,
    article_id VARCHAR(255) REFERENCES articles(id),
    claim TEXT NOT NULL,
    verdict VARCHAR(50), -- 'true', 'mostly-true', 'mixed', 'mostly-false', 'false', 'unverifiable'
    confidence INTEGER CHECK (confidence >= 0 AND confidence <= 100),
    supporting_sources TEXT[],
    contradicting_sources TEXT[],
    explanation TEXT,
    context TEXT,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create fact_check_summaries table for overall article credibility
CREATE TABLE IF NOT EXISTS fact_check_summaries (
    id SERIAL PRIMARY KEY,
    article_id VARCHAR(255) REFERENCES articles(id) UNIQUE,
    overall_credibility_score INTEGER CHECK (overall_credibility_score >= 0 AND overall_credibility_score <= 100),
    credibility_rating VARCHAR(50), -- 'highly-credible', 'mostly-credible', 'mixed-credibility', 'low-credibility', 'not-credible'
    average_confidence INTEGER,
    verified_claims_count INTEGER,
    total_claims_checked INTEGER,
    supporting_sources TEXT[],
    contradicting_sources TEXT[],
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_articles_source ON articles(source);
CREATE INDEX IF NOT EXISTS idx_articles_bias_score ON articles(bias_score);
CREATE INDEX IF NOT EXISTS idx_articles_category ON articles(category);
CREATE INDEX IF NOT EXISTS idx_article_interactions_user ON article_interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_topics_name ON topics(name);
CREATE INDEX IF NOT EXISTS idx_fact_checks_article ON fact_checks(article_id);
CREATE INDEX IF NOT EXISTS idx_fact_checks_verdict ON fact_checks(verdict);
CREATE INDEX IF NOT EXISTS idx_fact_check_summaries_article ON fact_check_summaries(article_id);
CREATE INDEX IF NOT EXISTS idx_fact_check_summaries_rating ON fact_check_summaries(credibility_rating);

-- Insert default RSS feeds
INSERT INTO feeds (name, url, category, bias_rating) VALUES
    ('BBC News', 'http://feeds.bbci.co.uk/news/rss.xml', 'general', 'center'),
    ('CNN', 'http://rss.cnn.com/rss/cnn_topstories.rss', 'general', 'center-left'),
    ('Fox News', 'https://moxie.foxnews.com/google-publisher/latest.xml', 'general', 'right'),
    ('Reuters', 'https://feeds.reuters.com/reuters/topNews', 'general', 'center'),
    ('The Guardian', 'https://www.theguardian.com/world/rss', 'world', 'left'),
    ('NPR News', 'https://feeds.npr.org/1001/rss.xml', 'general', 'center-left'),
    ('The Wall Street Journal', 'https://feeds.a.dj.com/rss/RSSWorldNews.xml', 'world', 'center-right'),
    ('Associated Press', 'https://feeds.apnews.com/rss/apf-topnews', 'general', 'center'),
    ('Al Jazeera', 'https://www.aljazeera.com/xml/rss/all.xml', 'world', 'center-left'),
    ('The Economist', 'https://www.economist.com/sections/economics/rss.xml', 'economics', 'center-right')
ON CONFLICT (url) DO NOTHING;