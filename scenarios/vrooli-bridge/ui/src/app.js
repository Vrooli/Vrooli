// Vrooli Bridge UI Application
class VrooliBridgeApp {
    constructor() {
        this.apiBase = window.location.origin.replace(/:\d+/, '') + ':' + (this.getApiPort() || '8080');
        this.projects = [];
        this.filteredProjects = [];
        
        this.init();
    }
    
    getApiPort() {
        // Try to detect API port from current URL or environment
        const currentPort = window.location.port;
        if (currentPort) {
            // Assume API is on current port + 1000 (UI on 3000, API on 8000, etc.)
            return parseInt(currentPort) + 5000;
        }
        return '8080'; // fallback
    }
    
    init() {
        this.bindEvents();
        this.loadProjects();
        this.startAutoRefresh();
    }
    
    bindEvents() {
        // Header actions
        document.getElementById('scan-btn').addEventListener('click', () => this.showScanModal());
        document.getElementById('refresh-btn').addEventListener('click', () => this.loadProjects());
        
        // Filters
        document.getElementById('type-filter').addEventListener('change', () => this.applyFilters());
        document.getElementById('status-filter').addEventListener('change', () => this.applyFilters());
        document.getElementById('search-input').addEventListener('input', () => this.applyFilters());
        
        // Modal actions
        document.getElementById('modal-close').addEventListener('click', () => this.hideScanModal());
        document.getElementById('cancel-scan').addEventListener('click', () => this.hideScanModal());
        document.getElementById('start-scan').addEventListener('click', () => this.startScan());
        
        // Close modal on backdrop click
        document.getElementById('scan-modal').addEventListener('click', (e) => {
            if (e.target.id === 'scan-modal') {
                this.hideScanModal();
            }
        });
        
        // Set default scan path to user home
        document.getElementById('scan-path').value = this.getDefaultScanPath();
    }
    
    getDefaultScanPath() {
        // Try to guess user's home directory
        return process?.env?.HOME || process?.env?.USERPROFILE || '/home/user';
    }
    
    async makeApiCall(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.apiBase}/api/v1${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });
            
            if (!response.ok) {
                throw new Error(`API call failed: ${response.status} ${response.statusText}`);
            }
            
            return await response.json();
        } catch (error) {
            console.error('API call failed:', error);
            this.showError(`Failed to communicate with API: ${error.message}`);
            throw error;
        }
    }
    
    async loadProjects() {
        this.showLoading(true);
        this.hideError();
        
        try {
            const response = await this.makeApiCall('/projects');
            this.projects = response.projects || [];
            this.updateStats();
            this.applyFilters();
        } catch (error) {
            this.projects = [];
            this.updateStats();
            this.renderProjects([]);
        } finally {
            this.showLoading(false);
        }
    }
    
    applyFilters() {
        const typeFilter = document.getElementById('type-filter').value;
        const statusFilter = document.getElementById('status-filter').value;
        const searchTerm = document.getElementById('search-input').value.toLowerCase();
        
        this.filteredProjects = this.projects.filter(project => {
            const matchesType = !typeFilter || project.type === typeFilter;
            const matchesStatus = !statusFilter || project.integration_status === statusFilter;
            const matchesSearch = !searchTerm || 
                project.name.toLowerCase().includes(searchTerm) ||
                project.path.toLowerCase().includes(searchTerm);
            
            return matchesType && matchesStatus && matchesSearch;
        });
        
        this.renderProjects(this.filteredProjects);
    }
    
    updateStats() {
        const total = this.projects.length;
        const integrated = this.projects.filter(p => p.integration_status === 'active').length;
        const outdated = this.projects.filter(p => p.integration_status === 'outdated').length;
        const errors = this.projects.filter(p => p.integration_status === 'error').length;
        
        document.getElementById('total-projects').textContent = total;
        document.getElementById('integrated-projects').textContent = integrated;
        document.getElementById('outdated-projects').textContent = outdated;
        document.getElementById('error-projects').textContent = errors;
    }
    
    renderProjects(projects) {
        const container = document.getElementById('projects-list');
        
        if (projects.length === 0) {
            container.innerHTML = `
                <div class="no-projects">
                    <p>No projects found. Try scanning for projects or adjusting your filters.</p>
                </div>
            `;
            return;
        }
        
        container.innerHTML = projects.map(project => this.renderProjectCard(project)).join('');
        
        // Bind action events
        projects.forEach(project => {
            this.bindProjectActions(project);
        });
    }
    
    renderProjectCard(project) {
        const lastUpdated = project.last_updated ? 
            new Date(project.last_updated).toLocaleDateString() : 'Never';
        
        const statusClass = `status-${project.integration_status}`;
        const statusText = project.integration_status.charAt(0).toUpperCase() + 
                          project.integration_status.slice(1);
        
        return `
            <div class="project-card" data-project-id="${project.id}">
                <div class="project-header">
                    <div class="project-info">
                        <h3>${this.escapeHtml(project.name)}</h3>
                        <div class="project-path">${this.escapeHtml(project.path)}</div>
                    </div>
                    <div class="project-actions">
                        ${this.renderProjectActions(project)}
                    </div>
                </div>
                
                <div class="project-details">
                    <div class="detail-group">
                        <div class="detail-label">Status</div>
                        <div class="detail-value">
                            <span class="status ${statusClass}">${statusText}</span>
                        </div>
                    </div>
                    
                    <div class="detail-group">
                        <div class="detail-label">Type</div>
                        <div class="detail-value">
                            <span class="type-badge">${this.escapeHtml(project.type)}</span>
                        </div>
                    </div>
                    
                    <div class="detail-group">
                        <div class="detail-label">Last Updated</div>
                        <div class="detail-value">${lastUpdated}</div>
                    </div>
                    
                    <div class="detail-group">
                        <div class="detail-label">Vrooli Version</div>
                        <div class="detail-value">${project.vrooli_version || 'Unknown'}</div>
                    </div>
                </div>
            </div>
        `;
    }
    
    renderProjectActions(project) {
        const actions = [];
        
        if (project.integration_status === 'missing') {
            actions.push(`
                <button class="btn btn-success btn-small integrate-btn" 
                        data-project-id="${project.id}">
                    <span class="btn-icon">✓</span>
                    Integrate
                </button>
            `);
        }
        
        if (project.integration_status === 'active' || project.integration_status === 'outdated') {
            actions.push(`
                <button class="btn btn-warning btn-small update-btn" 
                        data-project-id="${project.id}">
                    <span class="btn-icon">↻</span>
                    Update
                </button>
            `);
        }
        
        if (project.integration_status !== 'missing') {
            actions.push(`
                <button class="btn btn-danger btn-small remove-btn" 
                        data-project-id="${project.id}">
                    <span class="btn-icon">✗</span>
                    Remove
                </button>
            `);
        }
        
        return actions.join('');
    }
    
    bindProjectActions(project) {
        const card = document.querySelector(`[data-project-id="${project.id}"]`);
        if (!card) return;
        
        const integrateBtn = card.querySelector('.integrate-btn');
        const updateBtn = card.querySelector('.update-btn');
        const removeBtn = card.querySelector('.remove-btn');
        
        if (integrateBtn) {
            integrateBtn.addEventListener('click', () => this.integrateProject(project));
        }
        
        if (updateBtn) {
            updateBtn.addEventListener('click', () => this.updateProject(project));
        }
        
        if (removeBtn) {
            removeBtn.addEventListener('click', () => this.removeIntegration(project));
        }
    }
    
    async integrateProject(project) {
        try {
            await this.makeApiCall(`/projects/${project.id}/integrate`, {
                method: 'POST'
            });
            
            this.showSuccess(`Successfully integrated ${project.name}`);
            this.loadProjects();
        } catch (error) {
            this.showError(`Failed to integrate ${project.name}: ${error.message}`);
        }
    }
    
    async updateProject(project) {
        try {
            await this.makeApiCall(`/projects/${project.id}/integrate`, {
                method: 'POST',
                body: JSON.stringify({ force: true })
            });
            
            this.showSuccess(`Successfully updated ${project.name}`);
            this.loadProjects();
        } catch (error) {
            this.showError(`Failed to update ${project.name}: ${error.message}`);
        }
    }
    
    async removeIntegration(project) {
        if (!confirm(`Remove Vrooli integration from ${project.name}?`)) {
            return;
        }
        
        try {
            await this.makeApiCall(`/projects/${project.id}`, {
                method: 'DELETE'
            });
            
            this.showSuccess(`Successfully removed integration from ${project.name}`);
            this.loadProjects();
        } catch (error) {
            this.showError(`Failed to remove integration from ${project.name}: ${error.message}`);
        }
    }
    
    showScanModal() {
        document.getElementById('scan-modal').classList.remove('hidden');
        document.getElementById('scan-progress').classList.add('hidden');
    }
    
    hideScanModal() {
        document.getElementById('scan-modal').classList.add('hidden');
    }
    
    async startScan() {
        const path = document.getElementById('scan-path').value.trim();
        const depth = parseInt(document.getElementById('scan-depth').value) || 3;
        
        if (!path) {
            this.showError('Please enter a directory path to scan');
            return;
        }
        
        document.getElementById('scan-progress').classList.remove('hidden');
        document.getElementById('scan-status').textContent = 'Scanning...';
        
        try {
            const response = await this.makeApiCall('/projects/scan', {
                method: 'POST',
                body: JSON.stringify({
                    directories: [path],
                    depth: depth
                })
            });
            
            document.getElementById('scan-status').textContent = 
                `Found ${response.found} projects, ${response.new} new`;
            
            setTimeout(() => {
                this.hideScanModal();
                this.loadProjects();
            }, 2000);
            
        } catch (error) {
            document.getElementById('scan-status').textContent = `Scan failed: ${error.message}`;
        }
    }
    
    showLoading(show) {
        document.getElementById('loading').classList.toggle('hidden', !show);
    }
    
    showError(message) {
        const errorEl = document.getElementById('error-message');
        errorEl.textContent = message;
        errorEl.classList.remove('hidden');
        
        // Auto-hide after 5 seconds
        setTimeout(() => this.hideError(), 5000);
    }
    
    hideError() {
        document.getElementById('error-message').classList.add('hidden');
    }
    
    showSuccess(message) {
        // Create a temporary success message
        const successEl = document.createElement('div');
        successEl.className = 'error-message'; // Reuse error styling but with different color
        successEl.style.background = 'rgba(16, 185, 129, 0.1)';
        successEl.style.color = '#10b981';
        successEl.style.borderColor = 'rgba(16, 185, 129, 0.3)';
        successEl.textContent = message;
        
        const container = document.querySelector('.container');
        container.insertBefore(successEl, container.firstChild);
        
        setTimeout(() => {
            successEl.remove();
        }, 3000);
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    startAutoRefresh() {
        // Refresh every 30 seconds
        setInterval(() => {
            this.loadProjects();
        }, 30000);
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new VrooliBridgeApp();
});