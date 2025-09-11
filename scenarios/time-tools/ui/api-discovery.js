/**
 * Time Tools API Discovery Module
 * 
 * Provides centralized, standards-compliant API discovery for all UI components.
 * Follows Vrooli standards by attempting multiple discovery methods without hardcoded fallbacks.
 */

class TimeToolsAPIDiscovery {
    constructor() {
        this.apiBaseUrl = null;
        this.isDiscovering = false;
        this.discoveryPromise = null;
    }

    /**
     * Discover the Time Tools API URL using Vrooli-compliant methods
     * @returns {Promise<string>} The base URL of the API
     * @throws {Error} When API cannot be discovered
     */
    async discoverAPI() {
        // Return cached result if already discovered
        if (this.apiBaseUrl) {
            return this.apiBaseUrl;
        }

        // Return ongoing discovery promise if in progress
        if (this.isDiscovering && this.discoveryPromise) {
            return this.discoveryPromise;
        }

        // Start new discovery process
        this.isDiscovering = true;
        this.discoveryPromise = this._performDiscovery();

        try {
            this.apiBaseUrl = await this.discoveryPromise;
            return this.apiBaseUrl;
        } finally {
            this.isDiscovering = false;
            this.discoveryPromise = null;
        }
    }

    /**
     * Internal discovery method that tries multiple approaches
     * @private
     */
    async _performDiscovery() {
        // Method 1: Check for explicitly set API URL (for development/testing)
        if (window.TIME_TOOLS_API_URL) {
            if (await this._validateAPIUrl(window.TIME_TOOLS_API_URL)) {
                return window.TIME_TOOLS_API_URL;
            }
        }

        // Method 2: Try to get port from Vrooli scenario system via parent window
        try {
            const port = await this._getPortFromVrooli();
            if (port) {
                const url = `http://localhost:${port}`;
                if (await this._validateAPIUrl(url)) {
                    return url;
                }
            }
        } catch (error) {
            console.debug('Vrooli port discovery failed:', error.message);
        }

        // Method 3: Check environment-style variables
        const envPort = this._getPortFromEnvironment();
        if (envPort) {
            const url = `http://localhost:${envPort}`;
            if (await this._validateAPIUrl(url)) {
                return url;
            }
        }

        // Method 4: Discovery via service registry (if available)
        try {
            const registryUrl = await this._discoverViaServiceRegistry();
            if (registryUrl && await this._validateAPIUrl(registryUrl)) {
                return registryUrl;
            }
        } catch (error) {
            console.debug('Service registry discovery failed:', error.message);
        }

        // All methods failed
        throw new Error(
            'Time Tools API not found. Please ensure the scenario is running via: vrooli scenario run time-tools'
        );
    }

    /**
     * Attempt to get port from Vrooli scenario system
     * @private
     */
    async _getPortFromVrooli() {
        // This would ideally communicate with a Vrooli service discovery endpoint
        // For now, we'll check if there's a way to call vrooli commands from the browser
        
        // Check if there's a Vrooli bridge available
        if (window.vrooli && window.vrooli.scenario && window.vrooli.scenario.getPort) {
            try {
                return await window.vrooli.scenario.getPort('time-tools', 'TIME_TOOLS_PORT');
            } catch (error) {
                throw new Error('Vrooli bridge port discovery failed');
            }
        }

        // Check if there's a parent frame that can provide port info
        if (window.parent !== window && window.parent.vrooli) {
            try {
                return await window.parent.vrooli.scenario.getPort('time-tools', 'TIME_TOOLS_PORT');
            } catch (error) {
                throw new Error('Parent frame port discovery failed');
            }
        }

        throw new Error('No Vrooli bridge available');
    }

    /**
     * Get port from environment-style variables
     * @private
     */
    _getPortFromEnvironment() {
        // Check various environment variable patterns
        if (typeof process !== 'undefined' && process.env) {
            return process.env.TIME_TOOLS_PORT || process.env.API_PORT;
        }

        // Check window-level environment
        if (window.env) {
            return window.env.TIME_TOOLS_PORT || window.env.API_PORT;
        }

        // Check data attributes on the document
        const portAttr = document.documentElement.getAttribute('data-api-port');
        if (portAttr) {
            return portAttr;
        }

        return null;
    }

    /**
     * Try to discover via service registry
     * @private
     */
    async _discoverViaServiceRegistry() {
        // Check if there's a local service registry endpoint
        const registryEndpoints = [
            'http://localhost:8500/v1/catalog/service/time-tools', // Consul
            'http://localhost:2379/v2/keys/services/time-tools'    // etcd
        ];

        for (const endpoint of registryEndpoints) {
            try {
                const response = await fetch(endpoint, {
                    method: 'GET',
                    signal: AbortSignal.timeout(1000)
                });

                if (response.ok) {
                    const data = await response.json();
                    // Parse registry response (format varies by registry type)
                    const serviceUrl = this._parseRegistryResponse(data);
                    if (serviceUrl) {
                        return serviceUrl;
                    }
                }
            } catch (error) {
                // Registry not available, continue
                continue;
            }
        }

        throw new Error('No service registry available');
    }

    /**
     * Parse service registry response to extract service URL
     * @private
     */
    _parseRegistryResponse(data) {
        // This is a placeholder - actual implementation would depend on registry format
        if (Array.isArray(data) && data.length > 0) {
            const service = data[0];
            if (service.ServiceAddress && service.ServicePort) {
                return `http://${service.ServiceAddress}:${service.ServicePort}`;
            }
        }
        return null;
    }

    /**
     * Validate that a URL actually hosts the Time Tools API
     * @private
     */
    async _validateAPIUrl(url) {
        try {
            const response = await fetch(`${url}/api/v1/health`, {
                method: 'GET',
                signal: AbortSignal.timeout(2000),
                headers: {
                    'Accept': 'application/json'
                }
            });

            if (!response.ok) {
                return false;
            }

            const data = await response.json();
            
            // Verify this is actually the Time Tools API
            return data.service === 'time-tools' || data.service === 'time_tools';
            
        } catch (error) {
            return false;
        }
    }

    /**
     * Clear cached discovery result (useful for retrying after failures)
     */
    clearCache() {
        this.apiBaseUrl = null;
    }

    /**
     * Get the current API base URL if already discovered
     * @returns {string|null} The base URL or null if not discovered yet
     */
    getCurrentAPIUrl() {
        return this.apiBaseUrl;
    }

    /**
     * Check if API is currently available
     * @returns {Promise<boolean>}
     */
    async isAPIAvailable() {
        try {
            const url = await this.discoverAPI();
            return await this._validateAPIUrl(url);
        } catch (error) {
            return false;
        }
    }
}

// Create singleton instance
const timeToolsAPI = new TimeToolsAPIDiscovery();

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = timeToolsAPI;
} else {
    window.timeToolsAPI = timeToolsAPI;
}