#!/usr/bin/env node

/**
 * Personal Digital Twin - API Server
 * Main API server for persona management, data ingestion, and training
 * Port: 8200
 */

const express = require('express');
const { Pool } = require('pg');
const { QdrantClient } = require('@qdrant/js-client-rest');
const crypto = require('crypto');

const app = express();
app.use(express.json());

// Environment configuration
const PORT = process.env.API_PORT || 8200;
const POSTGRES_HOST = process.env.POSTGRES_HOST || 'localhost';
const POSTGRES_PORT = process.env.POSTGRES_PORT || 5433;
const POSTGRES_USER = process.env.POSTGRES_USER || 'postgres';
const POSTGRES_PASSWORD = process.env.POSTGRES_PASSWORD || 'postgres';
const POSTGRES_DB = process.env.POSTGRES_DB || 'digital_twin';
const QDRANT_HOST = process.env.QDRANT_HOST || 'localhost';
const QDRANT_PORT = process.env.QDRANT_PORT || 6333;
const N8N_HOST = process.env.N8N_HOST || 'localhost';
const N8N_PORT = process.env.N8N_PORT || 5678;

// Database connection
const db = new Pool({
    host: POSTGRES_HOST,
    port: POSTGRES_PORT,
    user: POSTGRES_USER,
    password: POSTGRES_PASSWORD,
    database: POSTGRES_DB,
    max: 10,
    idleTimeoutMillis: 30000
});

// Qdrant connection
const qdrant = new QdrantClient({
    url: `http://${QDRANT_HOST}:${QDRANT_PORT}`
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'personal-digital-twin-api',
        timestamp: new Date().toISOString()
    });
});

// Create new persona
app.post('/api/persona/create', async (req, res) => {
    try {
        const { name, description, personality_traits } = req.body;
        
        const result = await db.query(
            'INSERT INTO personas (name, description, personality_traits) VALUES ($1, $2, $3) RETURNING id, name, created_at',
            [name, description, JSON.stringify(personality_traits || {})]
        );
        
        res.status(201).json({
            id: result.rows[0].id,
            name: result.rows[0].name,
            created_at: result.rows[0].created_at,
            message: 'Persona created successfully'
        });
    } catch (error) {
        console.error('Error creating persona:', error);
        res.status(500).json({
            error: 'Failed to create persona',
            details: error.message
        });
    }
});

// Connect data source
app.post('/api/datasource/connect', async (req, res) => {
    try {
        const { persona_id, source_type, path, config } = req.body;
        
        const result = await db.query(
            'INSERT INTO data_sources (persona_id, source_type, source_config) VALUES ($1, $2, $3) RETURNING id',
            [persona_id, source_type, JSON.stringify({ path, ...config })]
        );
        
        res.json({
            source_id: result.rows[0].id,
            message: 'Data source connected successfully'
        });
    } catch (error) {
        console.error('Error connecting data source:', error);
        res.status(500).json({
            error: 'Failed to connect data source',
            details: error.message
        });
    }
});

// Semantic search
app.post('/api/search', async (req, res) => {
    try {
        const { persona_id, query, limit = 10 } = req.body;
        
        // Generate embedding for search query (mock implementation)
        // In real implementation, this would use Ollama API
        const mockEmbedding = new Array(768).fill(0).map(() => Math.random());
        
        // Search in Qdrant
        const searchResult = await qdrant.search('memories', {
            vector: mockEmbedding,
            limit: limit,
            filter: {
                must: [
                    {
                        key: 'persona_id',
                        match: { value: persona_id }
                    }
                ]
            }
        });
        
        const results = searchResult.map(point => ({
            id: point.id,
            score: point.score,
            text: point.payload.text,
            source: point.payload.source,
            timestamp: point.payload.timestamp
        }));
        
        res.json({
            results: results,
            query: query,
            total: results.length
        });
    } catch (error) {
        console.error('Error performing search:', error);
        res.status(500).json({
            error: 'Search failed',
            details: error.message
        });
    }
});

// Start training
app.post('/api/train/start', async (req, res) => {
    try {
        const { persona_id, model, technique } = req.body;
        
        // Create training job record
        const result = await db.query(
            'INSERT INTO training_jobs (persona_id, model_name, technique, status) VALUES ($1, $2, $3, $4) RETURNING id',
            [persona_id, model, technique, 'queued']
        );
        
        // Trigger training via n8n webhook (async)
        try {
            const fetch = require('node-fetch');
            await fetch(`http://${N8N_HOST}:${N8N_PORT}/webhook/train`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ persona_id, model, technique, job_id: result.rows[0].id })
            });
        } catch (webhookError) {
            console.warn('Failed to trigger training webhook:', webhookError);
        }
        
        res.status(202).json({
            job_id: result.rows[0].id,
            status: 'queued',
            message: 'Training job started'
        });
    } catch (error) {
        console.error('Error starting training:', error);
        res.status(500).json({
            error: 'Failed to start training',
            details: error.message
        });
    }
});

// Create API token
app.post('/api/tokens/create', async (req, res) => {
    try {
        const { persona_id, name, permissions } = req.body;
        
        // Generate secure token
        const token = crypto.randomBytes(32).toString('hex');
        const tokenHash = crypto.createHash('sha256').update(token).digest('hex');
        
        const result = await db.query(
            'INSERT INTO api_tokens (persona_id, token_hash, name, permissions) VALUES ($1, $2, $3, $4) RETURNING id',
            [persona_id, tokenHash, name, JSON.stringify(permissions || ['read'])]
        );
        
        res.status(201).json({
            token: token,
            token_id: result.rows[0].id,
            name: name,
            permissions: permissions,
            message: 'API token created successfully'
        });
    } catch (error) {
        console.error('Error creating API token:', error);
        res.status(500).json({
            error: 'Failed to create API token',
            details: error.message
        });
    }
});

// Get persona details
app.get('/api/persona/:id', async (req, res) => {
    try {
        const { id } = req.params;
        
        const result = await db.query(
            'SELECT * FROM personas WHERE id = $1',
            [id]
        );
        
        if (result.rows.length === 0) {
            return res.status(404).json({ error: 'Persona not found' });
        }
        
        res.json(result.rows[0]);
    } catch (error) {
        console.error('Error fetching persona:', error);
        res.status(500).json({
            error: 'Failed to fetch persona',
            details: error.message
        });
    }
});

// Error handling middleware
app.use((error, req, res, next) => {
    console.error('Unhandled error:', error);
    res.status(500).json({
        error: 'Internal server error',
        details: error.message
    });
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
    console.log(`Personal Digital Twin API server running on port ${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down gracefully...');
    db.end();
    process.exit(0);
});

process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully...');
    db.end();
    process.exit(0);
});