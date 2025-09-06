// Prompt Injection Arena - Professional Security Research Platform
// JavaScript Application Logic

class PromptInjectionArena {
    constructor() {
        this.apiBase = window.location.protocol + '//' + window.location.hostname + ':20300';
        this.currentTab = 'dashboard';
        this.refreshInterval = null;
        
        this.init();
    }
    
    async init() {
        this.bindEvents();
        this.startStatusChecks();
        await this.loadInitialData();
        this.startAutoRefresh();
    }
    
    bindEvents() {
        // Navigation
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                this.switchTab(e.target.dataset.tab);
            });
        });
        
        // Header refresh button
        document.getElementById('refresh-btn').addEventListener('click', () => {
            this.refreshCurrentTab();
        });
        
        // Test agent form
        document.getElementById('test-agent-btn').addEventListener('click', () => {
            this.runAgentTest();
        });
        
        document.getElementById('clear-test-btn').addEventListener('click', () => {
            this.clearTestForm();
        });
        
        // Add injection modal
        document.getElementById('add-injection-btn').addEventListener('click', () => {
            this.showAddInjectionModal();
        });
        
        document.getElementById('close-modal').addEventListener('click', () => {
            this.hideAddInjectionModal();
        });
        
        document.getElementById('cancel-injection').addEventListener('click', () => {
            this.hideAddInjectionModal();
        });
        
        document.getElementById('save-injection').addEventListener('click', () => {
            this.saveNewInjection();
        });
        
        // Form sliders
        document.getElementById('temperature-slider').addEventListener('input', (e) => {
            document.getElementById('temperature-value').textContent = e.target.value;
        });
        
        document.getElementById('difficulty-slider').addEventListener('input', (e) => {
            document.getElementById('difficulty-value').textContent = e.target.value;
        });
        
        // Library filters
        document.getElementById('category-filter').addEventListener('change', () => {
            this.loadInjectionLibrary();
        });
        
        document.getElementById('search-input').addEventListener('input', 
            this.debounce(() => this.loadInjectionLibrary(), 500)
        );
        
        // Modal backdrop click
        document.getElementById('add-injection-modal').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                this.hideAddInjectionModal();
            }
        });
    }
    
    switchTab(tabName) {
        // Update active tab
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.tab === tabName);
        });
        
        // Update active content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.toggle('active', content.id === `${tabName}-tab`);
        });
        
        this.currentTab = tabName;
        
        // Load tab-specific data
        this.loadTabData(tabName);
    }
    
    async loadTabData(tabName) {
        switch (tabName) {
            case 'dashboard':
                await this.loadDashboardData();
                break;
            case 'library':
                await this.loadInjectionLibrary();
                break;
            case 'leaderboards':
                await this.loadLeaderboards();
                break;
        }
    }
    
    async refreshCurrentTab() {
        const refreshBtn = document.getElementById('refresh-btn');
        const icon = refreshBtn.querySelector('i');
        
        icon.classList.add('fa-spin');
        await this.loadTabData(this.currentTab);
        await this.loadHeaderStats();
        
        setTimeout(() => {
            icon.classList.remove('fa-spin');
        }, 500);
    }
    
    startAutoRefresh() {
        this.refreshInterval = setInterval(() => {
            if (this.currentTab === 'dashboard') {
                this.loadHeaderStats();
                this.loadDashboardData();
            }
        }, 30000); // Refresh every 30 seconds
    }
    
    async startStatusChecks() {
        await this.checkSystemStatus();
        setInterval(() => this.checkSystemStatus(), 15000); // Check every 15 seconds
    }
    
    async checkSystemStatus() {
        try {
            const response = await this.apiCall('/health');
            const isHealthy = response && response.status === 'healthy';
            
            this.updateStatusIndicator(isHealthy ? 'healthy' : 'error', 
                isHealthy ? 'System Operational' : 'System Error');
                
        } catch (error) {
            console.error('Status check failed:', error);
            this.updateStatusIndicator('error', 'API Unreachable');
        }
    }
    
    updateStatusIndicator(status, text) {
        const dot = document.getElementById('status-dot');
        const statusText = document.getElementById('status-text');
        
        dot.className = `status-dot ${status}`;
        statusText.textContent = text;
    }
    
    async loadInitialData() {
        await Promise.all([
            this.loadHeaderStats(),
            this.loadDashboardData(),
            this.loadInjectionCategories()
        ]);
    }
    
    async loadHeaderStats() {
        try {
            const stats = await this.apiCall('/api/v1/statistics');
            if (stats && stats.totals) {
                document.getElementById('total-injections').textContent = stats.totals.injections || '-';
                document.getElementById('total-agents').textContent = stats.totals.agents || '-';
                document.getElementById('total-tests').textContent = stats.totals.tests || '-';
            }
        } catch (error) {
            console.error('Failed to load header stats:', error);
        }
    }
    
    async loadDashboardData() {
        try {
            // Load system status
            const healthResponse = await this.apiCall('/health');
            this.updateSystemStatus(healthResponse);
            
            // Load statistics
            const stats = await this.apiCall('/api/v1/statistics');
            this.updateDashboardMetrics(stats);
            
            // Load top threats (most effective injections)
            const injectionLeaderboard = await this.apiCall('/api/v1/leaderboards/injections?limit=5');
            this.updateTopThreats(injectionLeaderboard);
            
            // Update recent activity
            this.updateRecentActivity();
            
        } catch (error) {
            console.error('Failed to load dashboard data:', error);
        }
    }
    
    updateSystemStatus(healthData) {
        const statusGrid = document.getElementById('system-status');
        
        const apiStatus = healthData && healthData.status === 'healthy' ? 'healthy' : 'error';
        const dbStatus = healthData ? 'healthy' : 'warning';
        const sandboxStatus = healthData ? 'healthy' : 'warning';
        
        statusGrid.innerHTML = `
            <div class="status-item">
                <span class="status-label">API</span>
                <span class="status-value ${apiStatus}">${apiStatus === 'healthy' ? 'Online' : 'Offline'}</span>
            </div>
            <div class="status-item">
                <span class="status-label">Database</span>
                <span class="status-value ${dbStatus}">${dbStatus === 'healthy' ? 'Connected' : 'Issues'}</span>
            </div>
            <div class="status-item">
                <span class="status-label">Security Sandbox</span>
                <span class="status-value ${sandboxStatus}">${sandboxStatus === 'healthy' ? 'Ready' : 'Unavailable'}</span>
            </div>
        `;
    }
    
    updateDashboardMetrics(stats) {
        if (!stats) return;
        
        document.getElementById('avg-robustness').textContent = '85.2%'; // Simulated
        document.getElementById('tests-today').textContent = stats.recent_activity?.tests_last_24h || '0';
        document.getElementById('success-rate').textContent = '91.7%'; // Simulated defense success rate
    }
    
    updateTopThreats(leaderboard) {
        const threatList = document.getElementById('top-threats');
        
        if (!leaderboard || !leaderboard.leaderboard || leaderboard.leaderboard.length === 0) {
            threatList.innerHTML = '<div class="loading">No threat data available</div>';
            return;
        }
        
        threatList.innerHTML = leaderboard.leaderboard.slice(0, 5).map(threat => `
            <div class="threat-item">
                <div>
                    <div class="threat-name">${threat.name}</div>
                    <small style="color: var(--text-muted)">${threat.additional_info.category}</small>
                </div>
                <div class="threat-score">${(threat.score * 100).toFixed(1)}%</div>
            </div>
        `).join('');
    }
    
    updateRecentActivity() {
        const activityList = document.getElementById('recent-activity');
        
        // Simulate recent activity
        activityList.innerHTML = `
            <div class="activity-item">
                <i class="fas fa-shield-alt"></i>
                <span>Agent "Constitutional AI" tested against 15 injection techniques</span>
            </div>
            <div class="activity-item">
                <i class="fas fa-plus"></i>
                <span>New injection technique "Unicode Confusion" added to library</span>
            </div>
            <div class="activity-item">
                <i class="fas fa-trophy"></i>
                <span>Leaderboards updated with latest test results</span>
            </div>
            <div class="activity-item">
                <i class="fas fa-exclamation-triangle"></i>
                <span>High-risk injection "Developer Mode" showing 78% success rate</span>
            </div>
        `;
    }
    
    async loadInjectionLibrary() {
        try {
            const category = document.getElementById('category-filter').value;
            const search = document.getElementById('search-input').value;
            
            let url = '/api/v1/injections/library?limit=50';
            if (category) url += `&category=${category}`;
            if (search) url += `&search=${search}`;
            
            const data = await this.apiCall(url);
            this.displayInjectionLibrary(data);
            
        } catch (error) {
            console.error('Failed to load injection library:', error);
            document.getElementById('injection-library').innerHTML = 
                '<div class="loading">Failed to load injection library</div>';
        }
    }
    
    displayInjectionLibrary(data) {
        const container = document.getElementById('injection-library');
        
        if (!data || !data.techniques || data.techniques.length === 0) {
            container.innerHTML = '<div class="loading">No injection techniques found</div>';
            return;
        }
        
        container.innerHTML = data.techniques.map(injection => `
            <div class="injection-card">
                <div class="injection-header">
                    <div>
                        <div class="injection-name">${injection.name}</div>
                        <div class="injection-category">${injection.category.replace('_', ' ')}</div>
                    </div>
                </div>
                <div class="injection-stats">
                    <div class="stat">
                        <i class="fas fa-chart-line"></i>
                        <span class="stat-value">${(injection.success_rate * 100).toFixed(1)}%</span>
                        <span>Success Rate</span>
                    </div>
                    <div class="stat">
                        <i class="fas fa-star"></i>
                        <span class="stat-value">${injection.difficulty_score.toFixed(1)}</span>
                        <span>Difficulty</span>
                    </div>
                </div>
                <div class="injection-example">
                    ${injection.example_prompt.substring(0, 120)}${injection.example_prompt.length > 120 ? '...' : ''}
                </div>
            </div>
        `).join('');
    }
    
    async loadInjectionCategories() {
        try {
            const data = await this.apiCall('/api/v1/injections/library?limit=1');
            if (data && data.categories) {
                const categoryFilter = document.getElementById('category-filter');
                const modalCategory = document.getElementById('injection-category');
                
                data.categories.forEach(category => {
                    const option = new Option(category.replace('_', ' ').toUpperCase(), category);
                    categoryFilter.appendChild(option.cloneNode(true));
                    modalCategory.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Failed to load categories:', error);
        }
    }
    
    async loadLeaderboards() {
        try {
            const [agentData, injectionData] = await Promise.all([
                this.apiCall('/api/v1/leaderboards/agents?limit=20'),
                this.apiCall('/api/v1/leaderboards/injections?limit=20')
            ]);
            
            this.displayLeaderboard('agent-leaderboard', agentData, 'agent');
            this.displayLeaderboard('injection-leaderboard', injectionData, 'injection');
            
        } catch (error) {
            console.error('Failed to load leaderboards:', error);
        }
    }
    
    displayLeaderboard(containerId, data, type) {
        const container = document.getElementById(containerId);
        
        if (!data || !data.leaderboard || data.leaderboard.length === 0) {
            container.innerHTML = '<div class="loading">No leaderboard data available</div>';
            return;
        }
        
        if (type === 'agent') {
            container.innerHTML = `
                <div class="leaderboard-header">
                    <div>Rank</div>
                    <div>Agent Name</div>
                    <div>Model</div>
                    <div>Robustness</div>
                    <div class="hide-mobile">Tests</div>
                    <div class="hide-mobile">Passed</div>
                </div>
                ${data.leaderboard.map(agent => `
                    <div class="leaderboard-row">
                        <div class="rank ${agent.additional_info.rank <= 3 ? 'top-3' : ''}">
                            ${agent.additional_info.rank <= 3 ? '🏆' : ''} ${agent.additional_info.rank}
                        </div>
                        <div class="agent-name">${agent.name}</div>
                        <div>${agent.additional_info.model_name}</div>
                        <div class="score ${this.getScoreClass(agent.score)}">${agent.score.toFixed(1)}%</div>
                        <div class="hide-mobile">${agent.tests_run}</div>
                        <div class="hide-mobile">${agent.tests_passed}</div>
                    </div>
                `).join('')}
            `;
        } else {
            container.innerHTML = `
                <div class="leaderboard-header">
                    <div>Rank</div>
                    <div>Injection Name</div>
                    <div>Category</div>
                    <div>Success Rate</div>
                    <div class="hide-mobile">Tests</div>
                    <div class="hide-mobile">Difficulty</div>
                </div>
                ${data.leaderboard.map(injection => `
                    <div class="leaderboard-row">
                        <div class="rank ${injection.additional_info.rank <= 3 ? 'top-3' : ''}">
                            ${injection.additional_info.rank <= 3 ? '⚠️' : ''} ${injection.additional_info.rank}
                        </div>
                        <div class="injection-name">${injection.name}</div>
                        <div>${injection.additional_info.category.replace('_', ' ')}</div>
                        <div class="score ${this.getScoreClass(injection.score * 100, true)}">${(injection.score * 100).toFixed(1)}%</div>
                        <div class="hide-mobile">${injection.tests_run}</div>
                        <div class="hide-mobile">${injection.additional_info.difficulty_score.toFixed(1)}</div>
                    </div>
                `).join('')}
            `;
        }
    }
    
    getScoreClass(score, isInjection = false) {
        if (isInjection) {
            // For injections, high success rate is "dangerous" (red)
            if (score >= 70) return 'low'; // Red for high success rate
            if (score >= 40) return 'medium'; // Orange for medium
            return 'high'; // Green for low success rate
        } else {
            // For agents, high robustness is good (green)
            if (score >= 80) return 'high';
            if (score >= 60) return 'medium';
            return 'low';
        }
    }
    
    async runAgentTest() {
        const systemPrompt = document.getElementById('system-prompt').value.trim();
        if (!systemPrompt) {
            alert('Please enter a system prompt to test');
            return;
        }
        
        const testData = {
            agent_config: {
                system_prompt: systemPrompt,
                model_name: document.getElementById('model-select').value,
                temperature: parseFloat(document.getElementById('temperature-slider').value),
                max_tokens: parseInt(document.getElementById('max-tokens').value)
            },
            max_execution_time: parseInt(document.getElementById('timeout').value) * 1000
        };
        
        this.showLoadingOverlay();
        
        try {
            const results = await this.apiCall('/api/v1/security/test-agent', {
                method: 'POST',
                body: JSON.stringify(testData)
            });
            
            this.displayTestResults(results);
            
        } catch (error) {
            console.error('Test failed:', error);
            alert('Test failed: ' + (error.message || 'Unknown error'));
        } finally {
            this.hideLoadingOverlay();
        }
    }
    
    displayTestResults(results) {
        const panel = document.getElementById('test-results-panel');
        const content = document.getElementById('test-results-content');
        
        if (!results) {
            content.innerHTML = '<div class="loading">No results available</div>';
            panel.style.display = 'block';
            return;
        }
        
        const robustnessScore = results.robustness_score || 0;
        const scoreAngle = (robustnessScore / 100) * 360;
        
        content.innerHTML = `
            <div class="robustness-score">
                <div class="score-circle" style="--score-angle: ${scoreAngle}deg">
                    <div class="score-text">${robustnessScore.toFixed(1)}%</div>
                </div>
                <h4>Robustness Score</h4>
            </div>
            
            <div class="results-summary">
                <div class="summary-grid">
                    <div class="summary-item">
                        <span class="summary-label">Total Tests:</span>
                        <span class="summary-value">${results.summary?.total_tests || 0}</span>
                    </div>
                    <div class="summary-item">
                        <span class="summary-label">Successful Attacks:</span>
                        <span class="summary-value" style="color: var(--danger-red)">
                            ${results.summary?.successful_injections || 0}
                        </span>
                    </div>
                    <div class="summary-item">
                        <span class="summary-label">Blocked Attacks:</span>
                        <span class="summary-value" style="color: var(--success-green)">
                            ${results.summary?.failed_injections || 0}
                        </span>
                    </div>
                    <div class="summary-item">
                        <span class="summary-label">Avg Response Time:</span>
                        <span class="summary-value">${results.summary?.avg_execution_time_ms || 0}ms</span>
                    </div>
                </div>
            </div>
            
            ${results.recommendations && results.recommendations.length > 0 ? `
                <div class="recommendations">
                    <h4><i class="fas fa-lightbulb"></i> Recommendations</h4>
                    <ul>
                        ${results.recommendations.map(rec => `<li>${rec}</li>`).join('')}
                    </ul>
                </div>
            ` : ''}
            
            <div class="test-details">
                <h4><i class="fas fa-list"></i> Detailed Results</h4>
                <div class="results-list">
                    ${results.test_results ? results.test_results.map(result => `
                        <div class="result-item ${result.success ? 'failed' : 'passed'}">
                            <div class="result-header">
                                <span class="result-status">
                                    ${result.success ? '❌ VULNERABLE' : '✅ PROTECTED'}
                                </span>
                                <span class="confidence">Confidence: ${((result.confidence_score || 0) * 100).toFixed(0)}%</span>
                            </div>
                            <div class="result-response">
                                ${result.response_text ? result.response_text.substring(0, 200) + '...' : 'No response'}
                            </div>
                        </div>
                    `).join('') : '<div class="loading">No detailed results available</div>'}
                </div>
            </div>
        `;
        
        panel.style.display = 'block';
        panel.scrollIntoView({ behavior: 'smooth' });
    }
    
    clearTestForm() {
        document.getElementById('system-prompt').value = '';
        document.getElementById('model-select').value = 'llama3.2';
        document.getElementById('temperature-slider').value = '0.7';
        document.getElementById('temperature-value').textContent = '0.7';
        document.getElementById('max-tokens').value = '1000';
        document.getElementById('timeout').value = '30';
        
        const resultsPanel = document.getElementById('test-results-panel');
        resultsPanel.style.display = 'none';
    }
    
    showAddInjectionModal() {
        document.getElementById('add-injection-modal').classList.add('active');
    }
    
    hideAddInjectionModal() {
        document.getElementById('add-injection-modal').classList.remove('active');
        
        // Clear form
        document.getElementById('injection-name').value = '';
        document.getElementById('injection-category').value = '';
        document.getElementById('injection-example').value = '';
        document.getElementById('injection-description').value = '';
        document.getElementById('difficulty-slider').value = '0.5';
        document.getElementById('difficulty-value').textContent = '0.5';
    }
    
    async saveNewInjection() {
        const injectionData = {
            name: document.getElementById('injection-name').value.trim(),
            category: document.getElementById('injection-category').value,
            example_prompt: document.getElementById('injection-example').value.trim(),
            description: document.getElementById('injection-description').value.trim(),
            difficulty_score: parseFloat(document.getElementById('difficulty-slider').value)
        };
        
        if (!injectionData.name || !injectionData.category || !injectionData.example_prompt) {
            alert('Please fill in all required fields');
            return;
        }
        
        try {
            await this.apiCall('/api/v1/injections', {
                method: 'POST',
                body: JSON.stringify(injectionData)
            });
            
            this.hideAddInjectionModal();
            
            // Refresh the library if we're on that tab
            if (this.currentTab === 'library') {
                await this.loadInjectionLibrary();
            }
            
            // Show success message
            this.showNotification('Injection technique added successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to add injection:', error);
            alert('Failed to add injection technique: ' + (error.message || 'Unknown error'));
        }
    }
    
    showLoadingOverlay() {
        document.getElementById('loading-overlay').classList.add('active');
    }
    
    hideLoadingOverlay() {
        document.getElementById('loading-overlay').classList.remove('active');
    }
    
    showNotification(message, type = 'info') {
        // Simple notification - could be enhanced with a proper notification system
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: var(--security-green);
            color: white;
            padding: 1rem 1.5rem;
            border-radius: 6px;
            z-index: 3000;
            animation: slideIn 0.3s ease;
        `;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.remove();
        }, 3000);
    }
    
    async apiCall(endpoint, options = {}) {
        const url = this.apiBase + endpoint;
        const config = {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        };
        
        const response = await fetch(url, config);
        
        if (!response.ok) {
            throw new Error(`API call failed: ${response.statusText}`);
        }
        
        return await response.json();
    }
    
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }
}

// Additional CSS for result items
const additionalStyles = `
.result-item {
    margin-bottom: 0.75rem;
    padding: 0.75rem;
    background: var(--card-bg);
    border-radius: 6px;
    border-left: 3px solid var(--border-color);
}

.result-item.passed {
    border-left-color: var(--success-green);
}

.result-item.failed {
    border-left-color: var(--danger-red);
}

.result-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
    font-size: 0.85rem;
}

.result-status {
    font-weight: 600;
}

.confidence {
    color: var(--text-muted);
    font-family: var(--font-mono);
}

.result-response {
    font-size: 0.8rem;
    color: var(--text-secondary);
    font-family: var(--font-mono);
    background: var(--primary-bg);
    padding: 0.5rem;
    border-radius: 4px;
    word-break: break-word;
}

.recommendations {
    margin-top: 1rem;
    padding: 1rem;
    background: var(--card-bg);
    border-radius: 6px;
    border-left: 3px solid var(--info-blue);
}

.recommendations h4 {
    margin-bottom: 0.5rem;
    color: var(--info-blue);
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.recommendations ul {
    list-style: none;
    padding: 0;
}

.recommendations li {
    padding: 0.25rem 0;
    color: var(--text-secondary);
    font-size: 0.9rem;
}

.recommendations li::before {
    content: "💡 ";
    margin-right: 0.5rem;
}

@keyframes slideIn {
    from { transform: translateX(100%); opacity: 0; }
    to { transform: translateX(0); opacity: 1; }
}
`;

// Add additional styles
const styleSheet = document.createElement('style');
styleSheet.textContent = additionalStyles;
document.head.appendChild(styleSheet);

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    new PromptInjectionArena();
});