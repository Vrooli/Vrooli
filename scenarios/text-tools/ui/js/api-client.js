// API Client Module
class APIClient {
    constructor(baseURL) {
        if (!window.API_PORT) {
            throw new Error('API_PORT not configured. Please ensure the API is running.');
        }
        this.baseURL = baseURL || `http://localhost:${window.API_PORT}/api/v1/text`;
    }

    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${endpoint}`;
        const config = {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
        };

        try {
            const response = await fetch(url, config);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.message || `HTTP error! status: ${response.status}`);
            }

            return data;
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    }

    // Diff API
    async diff(text1, text2, options = {}) {
        return this.request('/diff', {
            method: 'POST',
            body: JSON.stringify({ text1, text2, options }),
        });
    }

    // Search API
    async search(text, pattern, options = {}) {
        return this.request('/search', {
            method: 'POST',
            body: JSON.stringify({ text, pattern, options }),
        });
    }

    async searchText(text, pattern, options = {}) {
        return this.search(text, pattern, options);
    }

    // Transform API
    async transform(text, transformations) {
        return this.request('/transform', {
            method: 'POST',
            body: JSON.stringify({ text, transformations }),
        });
    }

    async transformText(text, transformations) {
        return this.transform(text, transformations);
    }

    // Extract API
    async extract(source, format, options = {}) {
        return this.request('/extract', {
            method: 'POST',
            body: JSON.stringify({ source, format, options }),
        });
    }

    async extractText(source, options = {}) {
        return this.extract(source, options.format, options);
    }

    // Analyze API
    async analyze(text, analyses, options = {}) {
        return this.request('/analyze', {
            method: 'POST',
            body: JSON.stringify({ text, analyses, options }),
        });
    }

    async analyzeText(text, analyses, options = {}) {
        return this.analyze(text, analyses, options);
    }

    // Pipeline API (v2 endpoint)
    async processPipeline(input, steps) {
        // Use v2 endpoint for pipeline processing
        const url = `http://localhost:${window.API_PORT}/api/v2/text/pipeline`;
        
        try {
            const response = await fetch(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ input, steps }),
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || `HTTP error! status: ${response.status}`);
            }

            return data;
        } catch (error) {
            console.error('Pipeline API request failed:', error);
            throw error;
        }
    }

    // Health check
    async checkHealth() {
        if (!window.API_PORT) {
            throw new Error('API_PORT not configured');
        }
        const response = await fetch(`http://localhost:${window.API_PORT}/health`);
        return response.json();
    }
}

// Export for use in other modules
window.APIClient = APIClient;