#!/usr/bin/env node

const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();

// Get port from environment variable (set by lifecycle system)
const port = process.env.UI_PORT || 3251;

// Serve static files from current directory
app.use(express.static(__dirname));

// Serve main dashboard
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// API configuration endpoint for frontend
app.get('/config', (req, res) => {
    const apiPort = process.env.API_PORT; // Provided by lifecycle system
    res.json({
        apiUrl: `http://localhost:${apiPort}`,
        apiBase: `http://localhost:${apiPort}/api/v1`,
        uiPort: port,
        service: 'maintenance-orchestrator'
    });
});

// Legacy endpoint for backward compatibility
app.get('/api/config', (req, res) => {
    const apiPort = process.env.API_PORT;
    res.json({
        api_base: `http://localhost:${apiPort}/api/v1`,
        ui_port: port,
        service: 'maintenance-orchestrator'
    });
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'maintenance-orchestrator-ui',
        port: port,
        timestamp: new Date().toISOString() 
    });
});

// 404 handler
app.use((req, res) => {
    res.status(404).json({ error: 'Not found' });
});

// Error handler
app.use((err, req, res, next) => {
    console.error('Server error:', err);
    res.status(500).json({ error: 'Internal server error' });
});

// Start server
app.listen(port, () => {
    console.log(`🎛️ Maintenance Orchestrator Dashboard running on http://localhost:${port}`);
    console.log(`📊 Mission Control Interface ready`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('🛑 Shutting down Maintenance Orchestrator Dashboard...');
    process.exit(0);
});

process.on('SIGINT', () => {
    console.log('🛑 Shutting down Maintenance Orchestrator Dashboard...');
    process.exit(0);
});