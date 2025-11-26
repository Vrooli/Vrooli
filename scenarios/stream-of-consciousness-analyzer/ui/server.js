const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_URL = process.env.API_URL || 'http://localhost:8080';

// Middleware
app.use(cors());
app.use(express.json({ limit: '50mb' }));
app.use(express.static(path.join(__dirname)));

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'stream-of-consciousness-analyzer',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});


// API endpoints
app.post('/api/process-stream', async (req, res) => {
    try {
        const { text, campaign } = req.body;
        
        // Call API for processing stream of consciousness
        const response = await fetch(`${API_URL}/api/process-stream`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ text, campaign })
        });
        
        const result = await response.json();
        res.json(result);
    } catch (error) {
        console.error('Error processing stream:', error);
        res.status(500).json({ error: 'Failed to process stream' });
    }
});

app.post('/api/organize-thoughts', async (req, res) => {
    try {
        const { thoughts, campaign } = req.body;
        
        // Call API for organizing thoughts
        const response = await fetch(`${API_URL}/api/organize-thoughts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ thoughts, campaign })
        });
        
        const result = await response.json();
        res.json(result);
    } catch (error) {
        console.error('Error organizing thoughts:', error);
        res.status(500).json({ error: 'Failed to organize thoughts' });
    }
});

app.get('/api/campaigns', async (req, res) => {
    try {
        // Return list of campaigns (in real app, would fetch from database)
        const campaigns = [
            { id: 'general', name: 'General Thoughts', noteCount: 42 },
            { id: 'daily', name: 'Daily Reflections', noteCount: 28 },
            { id: 'work', name: 'Work Ideas', noteCount: 35 },
            { id: 'personal', name: 'Personal Growth', noteCount: 19 }
        ];
        res.json(campaigns);
    } catch (error) {
        console.error('Error fetching campaigns:', error);
        res.status(500).json({ error: 'Failed to fetch campaigns' });
    }
});

app.get('/api/notes', async (req, res) => {
    try {
        const { campaign, search } = req.query;
        
        // In real app, would fetch from database with filters
        const notes = [
            {
                id: '1',
                timestamp: new Date().toISOString(),
                title: 'Product Roadmap Thoughts',
                content: 'Considering the feedback from users, we should prioritize the mobile experience.',
                tags: ['product', 'priority-high', 'mobile'],
                campaign: 'work'
            },
            {
                id: '2',
                timestamp: new Date(Date.now() - 86400000).toISOString(),
                title: 'Morning Reflection',
                content: 'Feeling energized about the new direction. The team seems aligned and motivated.',
                tags: ['reflection', 'team', 'positive'],
                campaign: 'daily'
            }
        ];
        
        res.json(notes);
    } catch (error) {
        console.error('Error fetching notes:', error);
        res.status(500).json({ error: 'Failed to fetch notes' });
    }
});

app.post('/api/extract-insights', async (req, res) => {
    try {
        const { noteIds } = req.body;
        
        // Call API for extracting insights
        const response = await fetch(`${API_URL}/api/extract-insights`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ noteIds })
        });
        
        const result = await response.json();
        res.json(result);
    } catch (error) {
        console.error('Error extracting insights:', error);
        res.status(500).json({ error: 'Failed to extract insights' });
    }
});

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(PORT, () => {
    console.log(`Stream of Consciousness Analyzer running on http://localhost:${PORT}`);
    console.log(`API target: ${API_URL}`);
});
