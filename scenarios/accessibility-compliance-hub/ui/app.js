// Accessibility Compliance Hub - Dashboard Application

// API Configuration
const API_BASE = window.location.hostname === 'localhost' 
    ? 'http://localhost:3400/api/v1' 
    : '/api/v1';

// Application State
let scenarios = [];
let currentFilter = '';
let highContrastEnabled = false;
let animationsEnabled = true;

// Initialize dashboard on load
document.addEventListener('DOMContentLoaded', () => {
    initializeDashboard();
    setupEventListeners();
    loadDashboardData();
    startPolling();
});

// Initialize dashboard
function initializeDashboard() {
    // Check user preferences
    highContrastEnabled = localStorage.getItem('highContrast') === 'true';
    animationsEnabled = localStorage.getItem('animations') !== 'false';
    
    // Apply preferences
    if (highContrastEnabled) {
        document.body.classList.add('high-contrast');
    }
    
    if (!animationsEnabled) {
        document.body.classList.add('no-animations');
    }
    
    // Set up ARIA live region announcements
    const notifications = document.getElementById('notifications');
    window.announce = (message) => {
        notifications.textContent = message;
        setTimeout(() => notifications.textContent = '', 3000);
    };
}

// Set up event listeners
function setupEventListeners() {
    // Search functionality
    const searchInput = document.getElementById('scenario-search');
    searchInput?.addEventListener('input', (e) => {
        currentFilter = e.target.value.toLowerCase();
        filterScenarios();
    });
    
    // Keyboard navigation for table
    const table = document.querySelector('.compliance-table');
    table?.addEventListener('keydown', handleTableKeyboard);
}

// Load dashboard data
async function loadDashboardData() {
    try {
        // Fetch overall metrics
        const metricsResponse = await fetch(`${API_BASE}/accessibility/dashboard`);
        const metrics = await metricsResponse.json();
        
        updateMetrics(metrics);
        updateScenariosTable(metrics.scenarios);
        
        announce('Dashboard data loaded successfully');
    } catch (error) {
        console.error('Failed to load dashboard data:', error);
        announce('Error loading dashboard data');
        showError('Failed to load dashboard data. Please refresh the page.');
    }
}

// Update metrics display
function updateMetrics(metrics) {
    // Update overall score
    const overallScore = document.getElementById('overall-score');
    if (overallScore) {
        overallScore.textContent = metrics.overall_compliance || '--';
        overallScore.parentElement.classList.toggle('score-high', metrics.overall_compliance >= 90);
        overallScore.parentElement.classList.toggle('score-medium', metrics.overall_compliance >= 70 && metrics.overall_compliance < 90);
        overallScore.parentElement.classList.toggle('score-low', metrics.overall_compliance < 70);
    }
    
    // Update other metrics
    document.getElementById('active-scenarios').textContent = metrics.scenarios?.length || 0;
    document.getElementById('issues-fixed').textContent = metrics.issues_fixed_today || 0;
    document.getElementById('critical-issues').textContent = metrics.critical_issues || 0;
}

// Update scenarios table
function updateScenariosTable(scenarioData) {
    scenarios = scenarioData || [];
    const tbody = document.getElementById('scenarios-table');
    if (!tbody) return;
    
    tbody.innerHTML = '';
    
    const filteredScenarios = scenarios.filter(scenario => 
        !currentFilter || scenario.name.toLowerCase().includes(currentFilter)
    );
    
    filteredScenarios.forEach(scenario => {
        const row = createScenarioRow(scenario);
        tbody.appendChild(row);
    });
    
    if (filteredScenarios.length === 0) {
        const row = document.createElement('tr');
        row.innerHTML = `<td colspan="6" style="text-align: center; padding: 2rem;">No scenarios found</td>`;
        tbody.appendChild(row);
    }
}

// Create scenario table row
function createScenarioRow(scenario) {
    const row = document.createElement('tr');
    row.setAttribute('data-scenario-id', scenario.id);
    
    // Determine score class
    let scoreClass = 'score-low';
    if (scenario.score >= 90) scoreClass = 'score-high';
    else if (scenario.score >= 70) scoreClass = 'score-medium';
    
    // Format last audit date
    const lastAudit = scenario.last_audit 
        ? new Date(scenario.last_audit).toLocaleDateString() 
        : 'Never';
    
    row.innerHTML = `
        <td>
            <strong>${escapeHtml(scenario.name)}</strong>
            <br>
            <span style="font-size: 0.875rem; color: var(--color-text-secondary);">
                ${escapeHtml(scenario.id)}
            </span>
        </td>
        <td>
            <span class="score ${scoreClass}" role="status" aria-label="Score: ${scenario.score}%">
                ${scenario.score}%
            </span>
        </td>
        <td>${scenario.wcag_level || 'AA'}</td>
        <td>
            <span style="color: var(--color-danger);">${scenario.critical_issues || 0}</span> /
            <span style="color: var(--color-warning);">${scenario.major_issues || 0}</span> /
            <span style="color: var(--color-text-secondary);">${scenario.minor_issues || 0}</span>
        </td>
        <td>${lastAudit}</td>
        <td>
            <button 
                type="button" 
                class="btn btn-primary" 
                onclick="runAudit('${scenario.id}')"
                aria-label="Run audit for ${escapeHtml(scenario.name)}"
            >
                Audit
            </button>
            <button 
                type="button" 
                class="btn btn-secondary" 
                onclick="viewDetails('${scenario.id}')"
                aria-label="View details for ${escapeHtml(scenario.name)}"
            >
                Details
            </button>
        </td>
    `;
    
    return row;
}

// Run audit for scenario
async function runAudit(scenarioId) {
    try {
        announce(`Starting audit for scenario ${scenarioId}`);
        
        const response = await fetch(`${API_BASE}/accessibility/audit`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                scenario_id: scenarioId,
                wcag_level: 'AA',
                auto_fix: true,
                include_suggestions: true
            })
        });
        
        if (!response.ok) throw new Error('Audit failed');
        
        const result = await response.json();
        announce(`Audit completed. Score: ${result.score}%. Issues found: ${result.issues_found}`);
        
        // Reload dashboard data
        setTimeout(() => loadDashboardData(), 2000);
    } catch (error) {
        console.error('Audit failed:', error);
        announce('Audit failed. Please try again.');
    }
}

// Run all audits
async function runAllAudits() {
    if (!confirm('Run audits for all scenarios? This may take several minutes.')) return;
    
    announce('Starting audits for all scenarios');
    
    for (const scenario of scenarios) {
        await runAudit(scenario.id);
        // Add delay between audits
        await new Promise(resolve => setTimeout(resolve, 1000));
    }
    
    announce('All audits completed');
}

// View scenario details
async function viewDetails(scenarioId) {
    try {
        const response = await fetch(`${API_BASE}/accessibility/scenarios/${scenarioId}/issues`);
        const issues = await response.json();
        
        displayIssueDetails(scenarioId, issues);
        
        // Show issues section
        const issuesSection = document.querySelector('.issues-section');
        if (issuesSection) {
            issuesSection.hidden = false;
            issuesSection.scrollIntoView({ behavior: 'smooth' });
        }
    } catch (error) {
        console.error('Failed to load details:', error);
        announce('Failed to load scenario details');
    }
}

// Display issue details
function displayIssueDetails(scenarioId, issues) {
    const detailsContainer = document.getElementById('issue-details');
    if (!detailsContainer) return;
    
    const scenario = scenarios.find(s => s.id === scenarioId);
    
    detailsContainer.innerHTML = `
        <h3>Issues for ${escapeHtml(scenario?.name || scenarioId)}</h3>
        <div class="issues-list">
            ${issues.map(issue => `
                <div class="issue-card ${issue.auto_fixable ? 'auto-fixable' : ''}">
                    <div class="issue-header">
                        <span class="issue-type">${escapeHtml(issue.type)}</span>
                        <span class="issue-severity severity-${issue.severity}">${issue.severity}</span>
                    </div>
                    <p class="issue-description">${escapeHtml(issue.description)}</p>
                    <div class="issue-actions">
                        ${issue.auto_fixable ? `
                            <button 
                                type="button" 
                                class="btn btn-primary" 
                                onclick="applyFix('${issue.id}')"
                            >
                                Auto-Fix
                            </button>
                        ` : ''}
                        <a href="${issue.help_url}" target="_blank" rel="noopener">
                            Learn More
                        </a>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
}

// Apply fix for issue
async function applyFix(issueId) {
    try {
        announce('Applying fix...');
        
        const response = await fetch(`${API_BASE}/accessibility/fix`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                issue_ids: [issueId],
                review_before_apply: false
            })
        });
        
        if (!response.ok) throw new Error('Fix failed');
        
        announce('Fix applied successfully');
        loadDashboardData();
    } catch (error) {
        console.error('Failed to apply fix:', error);
        announce('Failed to apply fix');
    }
}

// Export compliance report
async function exportReport() {
    try {
        announce('Generating report...');
        
        const response = await fetch(`${API_BASE}/accessibility/report`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                format: 'pdf',
                include_all_scenarios: true
            })
        });
        
        if (!response.ok) throw new Error('Report generation failed');
        
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `compliance-report-${new Date().toISOString().split('T')[0]}.pdf`;
        a.click();
        
        announce('Report downloaded');
    } catch (error) {
        console.error('Failed to export report:', error);
        announce('Failed to generate report');
    }
}

// Filter scenarios
function filterScenarios() {
    updateScenariosTable(scenarios);
}

// Handle table keyboard navigation
function handleTableKeyboard(e) {
    if (e.key === 'Enter' || e.key === ' ') {
        const target = e.target;
        if (target.tagName === 'BUTTON') {
            e.preventDefault();
            target.click();
        }
    }
}

// Toggle high contrast mode
function toggleHighContrast() {
    highContrastEnabled = !highContrastEnabled;
    document.body.classList.toggle('high-contrast', highContrastEnabled);
    localStorage.setItem('highContrast', highContrastEnabled);
    announce(`High contrast ${highContrastEnabled ? 'enabled' : 'disabled'}`);
}

// Toggle animations
function toggleAnimations() {
    animationsEnabled = !animationsEnabled;
    document.body.classList.toggle('no-animations', !animationsEnabled);
    localStorage.setItem('animations', animationsEnabled);
    announce(`Animations ${animationsEnabled ? 'enabled' : 'disabled'}`);
}

// Show error message
function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-message';
    errorDiv.setAttribute('role', 'alert');
    errorDiv.textContent = message;
    document.body.appendChild(errorDiv);
    
    setTimeout(() => errorDiv.remove(), 5000);
}

// Escape HTML for security
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Start polling for updates
function startPolling() {
    // Poll every 30 seconds for updates
    setInterval(() => {
        loadDashboardData();
    }, 30000);
}

// Make functions globally available
window.runAudit = runAudit;
window.runAllAudits = runAllAudits;
window.viewDetails = viewDetails;
window.applyFix = applyFix;
window.exportReport = exportReport;
window.toggleHighContrast = toggleHighContrast;
window.toggleAnimations = toggleAnimations;