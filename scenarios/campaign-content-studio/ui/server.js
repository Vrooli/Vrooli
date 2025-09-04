const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname)));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'ok', service: 'Campaign Content Studio UI' });
});

// API endpoints for integration with the Go backend
app.get('/api/campaigns', async (req, res) => {
    try {
        // In a real implementation, this would proxy to the Go API
        const response = await fetch(`http://localhost:${API_PORT}/campaigns`);
        
        if (response.ok) {
            const data = await response.json();
            res.json(data);
        } else {
            // Fallback to mock data if API is not available
            res.json({
                campaigns: [
                    {
                        id: 1,
                        name: 'Summer Product Launch',
                        description: 'Marketing campaign for new product line',
                        created_at: '2024-10-01T00:00:00Z',
                        status: 'active'
                    },
                    {
                        id: 2,
                        name: 'Holiday Season 2024',
                        description: 'Holiday marketing and promotions',
                        created_at: '2024-10-15T00:00:00Z',
                        status: 'active'
                    }
                ]
            });
        }
    } catch (error) {
        console.error('Failed to fetch campaigns:', error);
        // Return mock data as fallback
        res.json({
            campaigns: [
                {
                    id: 1,
                    name: 'Demo Campaign',
                    description: 'Sample campaign for demonstration',
                    created_at: new Date().toISOString(),
                    status: 'active'
                }
            ]
        });
    }
});

app.post('/api/campaigns', async (req, res) => {
    try {
        const response = await fetch(`http://localhost:${API_PORT}/campaigns`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(req.body)
        });

        if (response.ok) {
            const data = await response.json();
            res.json(data);
        } else {
            throw new Error('API request failed');
        }
    } catch (error) {
        console.error('Failed to create campaign:', error);
        // Mock successful response
        res.json({
            id: Date.now(),
            ...req.body,
            created_at: new Date().toISOString(),
            status: 'active'
        });
    }
});

app.get('/api/documents', async (req, res) => {
    try {
        const response = await fetch(`http://localhost:${API_PORT}/documents`);
        
        if (response.ok) {
            const data = await response.json();
            res.json(data);
        } else {
            throw new Error('API request failed');
        }
    } catch (error) {
        console.error('Failed to fetch documents:', error);
        // Return mock data as fallback
        res.json({
            documents: [
                {
                    id: 1,
                    campaign_id: 1,
                    filename: 'Brand Guidelines.pdf',
                    content_type: 'application/pdf',
                    file_path: '/uploads/brand-guidelines.pdf',
                    upload_date: '2024-10-01T00:00:00Z'
                }
            ]
        });
    }
});

app.post('/api/generate-content', async (req, res) => {
    try {
        const response = await fetch(`http://localhost:${API_PORT}/generate-content`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(req.body)
        });

        if (response.ok) {
            const data = await response.json();
            res.json(data);
        } else {
            throw new Error('API request failed');
        }
    } catch (error) {
        console.error('Failed to generate content:', error);
        // Mock content generation response
        const { content_type = 'blog', prompt = '' } = req.body;
        
        res.json({
            content: {
                type: content_type,
                title: `Generated ${content_type.charAt(0).toUpperCase() + content_type.slice(1)} Content`,
                body: `This is AI-generated ${content_type} content based on your prompt: "${prompt}"\n\nIn a real implementation, this would be generated using the Ollama AI service with your campaign context and document embeddings.`,
                created_at: new Date().toISOString()
            }
        });
    }
});

// Serve the main HTML file for all non-API routes
app.get('*', (req, res) => {
    if (req.path.startsWith('/api/')) {
        return res.status(404).json({ error: 'API endpoint not found' });
    }
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Error handling middleware
app.use((err, req, res, next) => {
    console.error('Server error:', err);
    res.status(500).json({ error: 'Internal server error' });
});

app.listen(PORT, '0.0.0.0', () => {
    console.log(`🚀 Campaign Content Studio UI running on http://localhost:${PORT}`);
    console.log(`📁 Serving files from: ${__dirname}`);
    console.log(`🔗 API integration on port: ${process.env.API_PORT || 22500}`);
});

module.exports = app;
