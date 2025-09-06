// Scenario Surfer - Main Application Logic

class ScenarioSurfer {
    constructor() {
        this.scenarios = [];
        this.filteredScenarios = [];
        this.categories = [];
        this.currentScenario = null;
        this.currentIndex = -1;
        this.history = [];
        this.historyIndex = -1;
        this.apiPort = '27000'; // Default API port
        
        this.init();
    }
    
    async init() {
        console.log('🌊 Initializing Scenario Surfer');
        
        // Detect API port from environment or current URL
        const currentPort = window.location.port;
        if (currentPort) {
            // Assume API is on port - 1000 (e.g., UI on 34100, API on 27000)
            const estimatedApiPort = parseInt(currentPort) - 7100;
            if (estimatedApiPort > 0) {
                this.apiPort = estimatedApiPort.toString();
            }
        }
        
        this.setupEventListeners();
        this.setupKeyboardShortcuts();
        await this.loadScenarios();
        this.setupNavigation();
    }
    
    setupEventListeners() {
        // Navigation controls
        document.getElementById('back-btn').addEventListener('click', () => this.goBack());
        document.getElementById('random-btn').addEventListener('click', () => this.goToRandomScenario());
        document.getElementById('scenario-select').addEventListener('change', (e) => this.goToScenario(e.target.value));
        document.getElementById('category-select').addEventListener('change', (e) => this.filterByCategory(e.target.value));
        
        // Issue reporting
        document.getElementById('report-btn').addEventListener('click', () => this.openReportModal());
        document.getElementById('close-modal').addEventListener('click', () => this.closeReportModal());
        document.getElementById('cancel-report').addEventListener('click', () => this.closeReportModal());
        document.getElementById('report-form').addEventListener('submit', (e) => this.submitReport(e));
        
        // Retry button
        document.getElementById('retry-btn').addEventListener('click', () => this.loadScenarios());
        
        // Click outside modal to close
        document.getElementById('report-modal').addEventListener('click', (e) => {
            if (e.target.id === 'report-modal') {
                this.closeReportModal();
            }
        });
    }
    
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ignore if user is typing in an input field
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
                return;
            }
            
            switch (e.key) {
                case ' ': // Spacebar for random
                    e.preventDefault();
                    this.goToRandomScenario();
                    break;
                case 'Escape': // Escape for back
                    e.preventDefault();
                    this.goBack();
                    break;
                case 'r': // R for report
                    if (this.currentScenario) {
                        this.openReportModal();
                    }
                    break;
            }
        });
    }
    
    async loadScenarios() {
        try {
            this.showLoading();
            
            console.log(`🔍 Fetching scenarios from API on port ${this.apiPort}`);
            const response = await fetch(`http://localhost:${this.apiPort}/api/v1/scenarios/healthy`);
            
            if (!response.ok) {
                throw new Error(`API returned ${response.status}`);
            }
            
            const data = await response.json();
            this.scenarios = data.scenarios || [];
            this.categories = data.categories || [];
            
            console.log(`📱 Found ${this.scenarios.length} healthy scenarios`);
            console.log(`🏷️ Found ${this.categories.length} categories`);
            
            if (this.scenarios.length === 0) {
                this.showEmptyState();
                return;
            }
            
            this.filteredScenarios = [...this.scenarios];
            this.populateSelectors();
            
            // Go to first scenario or a random one
            if (this.scenarios.length > 0) {
                this.goToRandomScenario();
            }
            
        } catch (error) {
            console.error('❌ Failed to load scenarios:', error);
            this.showError('Failed to load scenarios. Make sure the API is running.');
            this.showEmptyState();
        }
    }
    
    populateSelectors() {
        // Populate category selector
        const categorySelect = document.getElementById('category-select');
        categorySelect.innerHTML = '<option value="all">All Scenarios</option>';
        
        // Group categories by common patterns
        const categoryGroups = this.groupCategories(this.categories);
        for (const [group, categories] of Object.entries(categoryGroups)) {
            if (categories.length > 0) {
                const option = document.createElement('option');
                option.value = group;
                option.textContent = this.formatCategoryName(group);
                categorySelect.appendChild(option);
            }
        }
        
        // Populate scenario selector
        this.updateScenarioSelector();
    }
    
    updateScenarioSelector() {
        const scenarioSelect = document.getElementById('scenario-select');
        scenarioSelect.innerHTML = '';
        
        if (this.filteredScenarios.length === 0) {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = 'No scenarios available';
            scenarioSelect.appendChild(option);
            return;
        }
        
        // Sort scenarios by name for better UX
        const sortedScenarios = [...this.filteredScenarios].sort((a, b) => 
            a.name.localeCompare(b.name)
        );
        
        for (const scenario of sortedScenarios) {
            const option = document.createElement('option');
            option.value = scenario.name;
            option.textContent = this.formatScenarioName(scenario.name);
            scenarioSelect.appendChild(option);
        }
    }
    
    groupCategories(categories) {
        const groups = {
            work: [],
            dev: [],
            fun: [],
            ai: [],
            other: []
        };
        
        for (const category of categories) {
            const lower = category.toLowerCase();
            if (lower.includes('business') || lower.includes('product') || lower.includes('manager')) {
                groups.work.push(category);
            } else if (lower.includes('dev') || lower.includes('debug') || lower.includes('api') || lower.includes('test')) {
                groups.dev.push(category);
            } else if (lower.includes('game') || lower.includes('fun') || lower.includes('entertainment')) {
                groups.fun.push(category);
            } else if (lower.includes('ai') || lower.includes('intelligence') || lower.includes('llm')) {
                groups.ai.push(category);
            } else {
                groups.other.push(category);
            }
        }
        
        return groups;
    }
    
    formatCategoryName(category) {
        const names = {
            work: 'Work & Business',
            dev: 'Development Tools',
            fun: 'Fun & Games',
            ai: 'AI & Intelligence',
            other: 'Other Tools'
        };
        return names[category] || category;
    }
    
    formatScenarioName(name) {
        return name
            .split('-')
            .map(word => word.charAt(0).toUpperCase() + word.slice(1))
            .join(' ');
    }
    
    filterByCategory(category) {
        if (category === 'all') {
            this.filteredScenarios = [...this.scenarios];
        } else {
            const categoryMap = {
                work: ['business', 'product', 'manager', 'invoice', 'finance'],
                dev: ['dev', 'debug', 'api', 'test', 'monitor', 'code'],
                fun: ['game', 'fun', 'entertainment', 'picker', 'wheel'],
                ai: ['ai', 'intelligence', 'llm', 'agent', 'reasoning'],
                other: []
            };
            
            const searchTerms = categoryMap[category] || [category];
            
            this.filteredScenarios = this.scenarios.filter(scenario => {
                const tags = scenario.tags || [];
                const name = scenario.name.toLowerCase();
                
                return searchTerms.some(term => 
                    tags.some(tag => tag.toLowerCase().includes(term)) ||
                    name.includes(term)
                );
            });
        }
        
        this.updateScenarioSelector();
        
        // If current scenario is no longer in filtered list, go to random one
        if (this.currentScenario && !this.filteredScenarios.find(s => s.name === this.currentScenario.name)) {
            this.goToRandomScenario();
        }
    }
    
    goToScenario(scenarioName) {
        if (!scenarioName) return;
        
        const scenario = this.filteredScenarios.find(s => s.name === scenarioName);
        if (!scenario) {
            this.showError('Scenario not found');
            return;
        }
        
        // Add to history
        if (this.currentScenario && this.currentScenario.name !== scenarioName) {
            this.addToHistory(this.currentScenario);
        }
        
        this.currentScenario = scenario;
        this.currentIndex = this.filteredScenarios.indexOf(scenario);
        
        this.updateUI();
        this.loadScenarioInIframe(scenario);
    }
    
    goToRandomScenario() {
        if (this.filteredScenarios.length === 0) {
            this.showError('No scenarios available');
            return;
        }
        
        // Pick a random scenario different from current one
        let randomScenario;
        let attempts = 0;
        do {
            const randomIndex = Math.floor(Math.random() * this.filteredScenarios.length);
            randomScenario = this.filteredScenarios[randomIndex];
            attempts++;
        } while (
            attempts < 10 && 
            this.currentScenario && 
            randomScenario.name === this.currentScenario.name &&
            this.filteredScenarios.length > 1
        );
        
        this.goToScenario(randomScenario.name);
    }
    
    goBack() {
        if (this.history.length === 0) {
            document.getElementById('back-btn').disabled = true;
            return;
        }
        
        const previousScenario = this.history.pop();
        this.currentScenario = previousScenario;
        this.currentIndex = this.filteredScenarios.indexOf(previousScenario);
        
        this.updateUI();
        this.loadScenarioInIframe(previousScenario);
        
        // Update back button state
        document.getElementById('back-btn').disabled = this.history.length === 0;
    }
    
    addToHistory(scenario) {
        // Avoid duplicates
        if (this.history.length === 0 || this.history[this.history.length - 1].name !== scenario.name) {
            this.history.push(scenario);
            
            // Limit history size
            if (this.history.length > 10) {
                this.history.shift();
            }
        }
        
        document.getElementById('back-btn').disabled = false;
    }
    
    updateUI() {
        if (!this.currentScenario) return;
        
        // Update current scenario display
        document.getElementById('current-scenario').textContent = 
            this.formatScenarioName(this.currentScenario.name);
        
        // Update health indicator
        const healthDiv = document.getElementById('scenario-health');
        healthDiv.innerHTML = `
            <div class="health-indicator ${this.currentScenario.health}"></div>
            <span>${this.currentScenario.health}</span>
        `;
        
        // Update scenario selector
        document.getElementById('scenario-select').value = this.currentScenario.name;
        
        // Update navigation button states
        document.getElementById('random-btn').disabled = this.filteredScenarios.length <= 1;
        document.getElementById('report-btn').disabled = false;
    }
    
    loadScenarioInIframe(scenario) {
        console.log(`📱 Loading scenario: ${scenario.name} at ${scenario.url}`);
        
        const iframe = document.getElementById('scenario-iframe');
        const container = document.getElementById('iframe-container');
        const loading = document.getElementById('loading-screen');
        
        // Show loading while iframe loads
        container.style.display = 'none';
        loading.style.display = 'flex';
        loading.querySelector('h2').textContent = `Loading ${this.formatScenarioName(scenario.name)}`;
        loading.querySelector('p').textContent = 'Please wait...';
        
        // Set iframe source
        iframe.src = scenario.url;
        
        // Handle iframe load
        const handleLoad = () => {
            loading.style.display = 'none';
            container.style.display = 'block';
            iframe.removeEventListener('load', handleLoad);
            iframe.removeEventListener('error', handleError);
        };
        
        const handleError = () => {
            this.showError(`Failed to load ${scenario.name}. The scenario may not support iframe embedding.`);
            iframe.removeEventListener('load', handleLoad);
            iframe.removeEventListener('error', handleError);
        };
        
        iframe.addEventListener('load', handleLoad);
        iframe.addEventListener('error', handleError);
        
        // Timeout after 10 seconds
        setTimeout(() => {
            if (loading.style.display !== 'none') {
                handleLoad();
                this.showWarning(`${scenario.name} is taking longer than usual to load.`);
            }
        }, 10000);
    }
    
    // Issue Reporting
    openReportModal() {
        if (!this.currentScenario) return;
        
        const modal = document.getElementById('report-modal');
        const form = document.getElementById('report-form');
        
        // Pre-fill scenario name in title
        document.getElementById('issue-title').value = `Issue with ${this.formatScenarioName(this.currentScenario.name)}`;
        
        modal.style.display = 'flex';
        document.getElementById('issue-title').focus();
    }
    
    closeReportModal() {
        const modal = document.getElementById('report-modal');
        const form = document.getElementById('report-form');
        
        modal.style.display = 'none';
        form.reset();
    }
    
    async submitReport(e) {
        e.preventDefault();
        
        if (!this.currentScenario) return;
        
        const title = document.getElementById('issue-title').value.trim();
        const description = document.getElementById('issue-description').value.trim();
        const includeScreenshot = document.getElementById('include-screenshot').checked;
        
        if (!title || !description) {
            this.showError('Please fill in all required fields');
            return;
        }
        
        try {
            const submitBtn = e.target.querySelector('button[type="submit"]');
            const originalText = submitBtn.textContent;
            submitBtn.textContent = 'Submitting...';
            submitBtn.disabled = true;
            
            const reportData = {
                scenario: this.currentScenario.name,
                title,
                description,
                screenshot: includeScreenshot ? this.currentScenario.url : undefined
            };
            
            const response = await fetch(`http://localhost:${this.apiPort}/api/v1/issues/report`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(reportData)
            });
            
            if (!response.ok) {
                throw new Error(`Failed to submit report: ${response.status}`);
            }
            
            this.showSuccess('Issue reported successfully! Thank you for helping improve Vrooli.');
            this.closeReportModal();
            
        } catch (error) {
            console.error('❌ Failed to submit report:', error);
            this.showError('Failed to submit report. Please try again.');
        } finally {
            const submitBtn = document.querySelector('#report-form button[type="submit"]');
            if (submitBtn) {
                submitBtn.textContent = 'Submit Report';
                submitBtn.disabled = false;
            }
        }
    }
    
    // UI State Management
    showLoading() {
        document.getElementById('loading-screen').style.display = 'flex';
        document.getElementById('iframe-container').style.display = 'none';
        document.getElementById('empty-state').style.display = 'none';
    }
    
    showEmptyState() {
        document.getElementById('loading-screen').style.display = 'none';
        document.getElementById('iframe-container').style.display = 'none';
        document.getElementById('empty-state').style.display = 'flex';
    }
    
    setupNavigation() {
        // Auto-hide navigation on inactivity
        let inactivityTimer;
        const resetInactivityTimer = () => {
            clearTimeout(inactivityTimer);
            document.body.classList.remove('nav-hidden');
            
            inactivityTimer = setTimeout(() => {
                if (!document.getElementById('report-modal').style.display || 
                    document.getElementById('report-modal').style.display === 'none') {
                    document.body.classList.add('nav-hidden');
                }
            }, 3000);
        };
        
        document.addEventListener('mousemove', resetInactivityTimer);
        document.addEventListener('keydown', resetInactivityTimer);
        
        resetInactivityTimer();
    }
    
    // Toast Notifications
    showSuccess(message) {
        this.showToast(message, 'success');
    }
    
    showError(message) {
        this.showToast(message, 'error');
    }
    
    showWarning(message) {
        this.showToast(message, 'warning');
    }
    
    showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        
        container.appendChild(toast);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (toast.parentElement) {
                toast.parentElement.removeChild(toast);
            }
        }, 5000);
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.scenarioSurfer = new ScenarioSurfer();
});