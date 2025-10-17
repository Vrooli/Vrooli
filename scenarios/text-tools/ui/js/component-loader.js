// Component Loader - Dynamically loads UI components
class ComponentLoader {
    constructor() {
        this.loadedComponents = new Map();
        this.cache = new Map();
        this.loadingPromises = new Map();
    }

    /**
     * Load a component by name
     * @param {string} componentName - Name of the component to load
     * @param {string} targetSelector - CSS selector for the target container
     * @returns {Promise<HTMLElement>} - The loaded component element
     */
    async loadComponent(componentName, targetSelector = null) {
        // Check if already loading this component
        if (this.loadingPromises.has(componentName)) {
            return await this.loadingPromises.get(componentName);
        }

        // Create loading promise
        const loadingPromise = this._doLoadComponent(componentName, targetSelector);
        this.loadingPromises.set(componentName, loadingPromise);

        try {
            const result = await loadingPromise;
            this.loadingPromises.delete(componentName);
            return result;
        } catch (error) {
            this.loadingPromises.delete(componentName);
            throw error;
        }
    }

    /**
     * Internal method to perform component loading
     */
    async _doLoadComponent(componentName, targetSelector) {
        try {
            // Check cache first
            if (this.cache.has(componentName)) {
                const cachedHtml = this.cache.get(componentName);
                return this._injectComponent(cachedHtml, componentName, targetSelector);
            }

            // Load component HTML
            const componentPath = `components/${componentName}.html`;
            const response = await fetch(componentPath);
            
            if (!response.ok) {
                throw new Error(`Failed to load component ${componentName}: ${response.status} ${response.statusText}`);
            }

            const html = await response.text();
            
            // Cache the component
            this.cache.set(componentName, html);

            // Inject into DOM
            return this._injectComponent(html, componentName, targetSelector);

        } catch (error) {
            console.error(`Error loading component ${componentName}:`, error);
            throw error;
        }
    }

    /**
     * Inject component HTML into the DOM
     */
    _injectComponent(html, componentName, targetSelector) {
        const container = targetSelector ? 
            document.querySelector(targetSelector) : 
            document.getElementById('tool-container');

        if (!container) {
            throw new Error(`Target container not found: ${targetSelector || '#tool-container'}`);
        }

        // Create a temporary div to parse the HTML
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = html.trim();
        
        const componentElement = tempDiv.firstChild;
        if (!componentElement) {
            throw new Error(`No valid HTML found in component ${componentName}`);
        }

        // Add to container
        container.appendChild(componentElement);

        // Mark as loaded
        this.loadedComponents.set(componentName, componentElement);

        console.log(`Component ${componentName} loaded successfully`);
        return componentElement;
    }

    /**
     * Load multiple components
     * @param {Array<string>} componentNames - Array of component names to load
     * @param {string} targetSelector - Target container selector
     */
    async loadComponents(componentNames, targetSelector = null) {
        const promises = componentNames.map(name => 
            this.loadComponent(name, targetSelector)
        );

        try {
            return await Promise.all(promises);
        } catch (error) {
            console.error('Error loading multiple components:', error);
            throw error;
        }
    }

    /**
     * Unload a component
     * @param {string} componentName - Name of the component to unload
     */
    unloadComponent(componentName) {
        const element = this.loadedComponents.get(componentName);
        if (element && element.parentNode) {
            element.parentNode.removeChild(element);
            this.loadedComponents.delete(componentName);
            console.log(`Component ${componentName} unloaded`);
        }
    }

    /**
     * Check if a component is loaded
     * @param {string} componentName - Name of the component
     */
    isComponentLoaded(componentName) {
        return this.loadedComponents.has(componentName);
    }

    /**
     * Get a loaded component element
     * @param {string} componentName - Name of the component
     */
    getComponent(componentName) {
        return this.loadedComponents.get(componentName);
    }

    /**
     * Reload a component (useful for development)
     * @param {string} componentName - Name of the component to reload
     */
    async reloadComponent(componentName) {
        // Remove from cache
        this.cache.delete(componentName);
        
        // Unload existing
        this.unloadComponent(componentName);
        
        // Load fresh
        return await this.loadComponent(componentName);
    }

    /**
     * Clear all cached components
     */
    clearCache() {
        this.cache.clear();
        console.log('Component cache cleared');
    }

    /**
     * Get loading status
     */
    getLoadingStatus() {
        return {
            loaded: Array.from(this.loadedComponents.keys()),
            cached: Array.from(this.cache.keys()),
            loading: Array.from(this.loadingPromises.keys())
        };
    }
}

// Global component loader instance
window.componentLoader = new ComponentLoader();

// Utility functions for common component operations
window.loadToolComponent = async function(toolName) {
    try {
        const componentName = `${toolName}-panel`;
        await window.componentLoader.loadComponent(componentName);
        console.log(`Tool component ${toolName} loaded`);
    } catch (error) {
        console.error(`Failed to load tool component ${toolName}:`, error);
        
        // Create a fallback component
        const container = document.getElementById('tool-container');
        const fallback = document.createElement('div');
        fallback.className = 'tool-panel';
        fallback.id = `${toolName}-panel`;
        fallback.innerHTML = `
            <div class="tool-header">
                <h2>${toolName.charAt(0).toUpperCase() + toolName.slice(1)} Tool</h2>
            </div>
            <div class="error-message">
                <p>Failed to load ${toolName} component.</p>
                <p>Error: ${error.message}</p>
                <button onclick="window.componentLoader.reloadComponent('${componentName}')">
                    Retry Loading
                </button>
            </div>
        `;
        container.appendChild(fallback);
    }
};

// Load all tool components
window.loadAllToolComponents = async function() {
    const tools = ['diff', 'search', 'transform', 'extract', 'analyze', 'pipeline'];
    const container = document.getElementById('tool-container');
    
    if (!container) {
        console.error('Tool container not found');
        return;
    }

    // Clear existing content
    container.innerHTML = '';
    
    // Show loading indicator
    const loadingDiv = document.createElement('div');
    loadingDiv.className = 'loading-indicator';
    loadingDiv.innerHTML = `
        <div class="loading-spinner"></div>
        <p>Loading components...</p>
    `;
    container.appendChild(loadingDiv);

    try {
        // Load all components
        const promises = tools.map(tool => window.loadToolComponent(tool));
        await Promise.all(promises);
        
        // Remove loading indicator
        if (loadingDiv.parentNode) {
            loadingDiv.parentNode.removeChild(loadingDiv);
        }
        
        console.log('All tool components loaded successfully');
        
        // Trigger initialization event
        document.dispatchEvent(new CustomEvent('componentsLoaded', { 
            detail: { tools } 
        }));
        
    } catch (error) {
        console.error('Error loading tool components:', error);
        
        // Update loading indicator to show error
        loadingDiv.innerHTML = `
            <div class="error-indicator">
                <p>Failed to load some components</p>
                <button onclick="window.loadAllToolComponents()">Retry</button>
            </div>
        `;
    }
};

// Auto-load components when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Small delay to ensure other scripts are loaded
    setTimeout(() => {
        window.loadAllToolComponents();
    }, 100);
});