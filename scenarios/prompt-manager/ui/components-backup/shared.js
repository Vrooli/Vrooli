// Shared utilities for Prompt Manager UI
// API configuration and HTTP utilities

// API URL Discovery
let API_URL = '';

// Fetch configuration on load
fetch('/api/config')
    .then(res => res.json())
    .then(config => {
        API_URL = config.apiUrl;
        console.log('API configured:', API_URL);
    })
    .catch(err => {
        console.error('Failed to fetch config:', err);
        // Fallback to environment variable port
        const port = window.location.port || '15000';
        API_URL = `http://localhost:${port}`;
    });

// API utilities
const api = {
    get: async (endpoint) => {
        const response = await fetch(`${API_URL}${endpoint}`);
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        return response.json();
    },
    post: async (endpoint, data) => {
        const response = await fetch(`${API_URL}${endpoint}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        return response.json();
    },
    put: async (endpoint, data) => {
        const response = await fetch(`${API_URL}${endpoint}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        return response.json();
    },
    delete: async (endpoint) => {
        const response = await fetch(`${API_URL}${endpoint}`, {
            method: 'DELETE'
        });
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
        return response.status === 204 ? {} : response.json();
    }
};

// Make API available globally for components
window.PromptManagerAPI = api;