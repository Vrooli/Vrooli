#!/usr/bin/env node

/**
 * Personal Digital Twin - Chat Server
 * Chat interface for interacting with AI personas
 * Port: 8201
 */

const express = require('express');
const { Pool } = require('pg');
const { QdrantClient } = require('@qdrant/js-client-rest');
const crypto = require('crypto');

const app = express();
app.use(express.json());

// Environment configuration
const PORT = process.env.CHAT_PORT || 8201;
const POSTGRES_HOST = process.env.POSTGRES_HOST || 'localhost';
const POSTGRES_PORT = process.env.POSTGRES_PORT || 5433;
const POSTGRES_USER = process.env.POSTGRES_USER || 'postgres';
const POSTGRES_PASSWORD = process.env.POSTGRES_PASSWORD || 'postgres';
const POSTGRES_DB = process.env.POSTGRES_DB || 'digital_twin';
const QDRANT_HOST = process.env.QDRANT_HOST || 'localhost';
const QDRANT_PORT = process.env.QDRANT_PORT || 6333;
const OLLAMA_HOST = process.env.OLLAMA_HOST || 'localhost';
const OLLAMA_PORT = process.env.OLLAMA_PORT || 11434;

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
        service: 'personal-digital-twin-chat',
        timestamp: new Date().toISOString()
    });
});

// Chat with persona
app.post('/api/chat', async (req, res) => {
    try {
        const { persona_id, message, session_id } = req.body;
        
        // Get persona information
        const personaResult = await db.query(
            'SELECT * FROM personas WHERE id = $1',
            [persona_id]
        );
        
        if (personaResult.rows.length === 0) {
            return res.status(404).json({ error: 'Persona not found' });
        }
        
        const persona = personaResult.rows[0];
        const actualSessionId = session_id || crypto.randomUUID();
        
        // Retrieve relevant context from vector search (mock implementation)
        // In real implementation, this would generate embeddings and search
        const contextResults = [];
        try {
            const mockEmbedding = new Array(768).fill(0).map(() => Math.random());
            const searchResult = await qdrant.search('memories', {
                vector: mockEmbedding,
                limit: 5,
                filter: {
                    must: [
                        {
                            key: 'persona_id',
                            match: { value: persona_id }
                        }
                    ]
                }
            });
            contextResults.push(...searchResult.map(point => point.payload.text));
        } catch (searchError) {
            console.warn('Context search failed:', searchError);
        }
        
        // Generate response using Ollama (mock implementation)
        let response;
        try {
            const fetch = require('node-fetch');
            
            const systemPrompt = `You are ${persona.name}, an AI persona with the following characteristics:
Description: ${persona.description}
Personality traits: ${JSON.stringify(persona.personality_traits)}

Context from your knowledge:
${contextResults.join('\n')}

Respond as this persona would, incorporating your personality traits and knowledge.`;
            
            const ollamaResponse = await fetch(`http://${OLLAMA_HOST}:${OLLAMA_PORT}/api/generate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    model: persona.fine_tuned_model_path || persona.base_model || 'llama3.2',
                    prompt: `${systemPrompt}\n\nHuman: ${message}\nAssistant:`,
                    stream: false,
                    options: {
                        temperature: persona.personality_traits?.creativity || 0.7,
                        top_p: 0.9,
                        max_tokens: 500
                    }
                })
            });
            
            const ollamaResult = await ollamaResponse.json();
            response = ollamaResult.response || 'I apologize, but I cannot generate a response at the moment.';
        } catch (ollamaError) {
            console.warn('Ollama request failed:', ollamaError);
            response = `Hello! I'm ${persona.name}. ${persona.description || 'How can I help you today?'}`;
        }
        
        // Store conversation in database
        try {
            // Get or create conversation
            let conversationResult = await db.query(
                'SELECT id, messages FROM conversations WHERE persona_id = $1 AND session_id = $2',
                [persona_id, actualSessionId]
            );
            
            let conversationId;
            let messages = [];
            
            if (conversationResult.rows.length === 0) {
                // Create new conversation
                const newConversationResult = await db.query(
                    'INSERT INTO conversations (persona_id, session_id, messages) VALUES ($1, $2, $3) RETURNING id',
                    [persona_id, actualSessionId, JSON.stringify([])]
                );
                conversationId = newConversationResult.rows[0].id;
            } else {
                conversationId = conversationResult.rows[0].id;
                messages = conversationResult.rows[0].messages || [];
            }
            
            // Add new messages
            messages.push({
                role: 'user',
                content: message,
                timestamp: new Date().toISOString()
            });
            
            messages.push({
                role: 'assistant',
                content: response,
                timestamp: new Date().toISOString()
            });
            
            // Update conversation
            await db.query(
                'UPDATE conversations SET messages = $1, total_tokens_used = total_tokens_used + $2 WHERE id = $3',
                [JSON.stringify(messages), message.length + response.length, conversationId]
            );
            
        } catch (dbError) {
            console.warn('Failed to store conversation:', dbError);
        }
        
        // Store conversation vectors for future context
        try {
            const conversationText = `User: ${message}\nAssistant: ${response}`;
            const mockEmbedding = new Array(768).fill(0).map(() => Math.random());
            
            await qdrant.upsert('conversations', {
                wait: false,
                points: [{
                    id: crypto.randomUUID(),
                    vector: mockEmbedding,
                    payload: {
                        persona_id: persona_id,
                        conversation_id: actualSessionId,
                        role: 'conversation',
                        message: conversationText,
                        timestamp: new Date().toISOString()
                    }
                }]
            });
        } catch (vectorError) {
            console.warn('Failed to store conversation vector:', vectorError);
        }
        
        res.json({
            response: response,
            session_id: actualSessionId,
            persona_name: persona.name,
            timestamp: new Date().toISOString(),
            context_used: contextResults.length > 0
        });
        
    } catch (error) {
        console.error('Error in chat:', error);
        res.status(500).json({
            error: 'Chat failed',
            details: error.message
        });
    }
});

// Get conversation history
app.get('/api/chat/history/:session_id', async (req, res) => {
    try {
        const { session_id } = req.params;
        const { persona_id } = req.query;
        
        const result = await db.query(
            'SELECT messages, created_at FROM conversations WHERE session_id = $1 AND persona_id = $2',
            [session_id, persona_id]
        );
        
        if (result.rows.length === 0) {
            return res.json({ messages: [], session_id });
        }
        
        res.json({
            messages: result.rows[0].messages || [],
            session_id: session_id,
            created_at: result.rows[0].created_at
        });
    } catch (error) {
        console.error('Error fetching chat history:', error);
        res.status(500).json({
            error: 'Failed to fetch chat history',
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
    console.log(`Personal Digital Twin Chat server running on port ${PORT}`);
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