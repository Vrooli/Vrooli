import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__dateNightPlannerBridgeInitialized';

function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window || window[BRIDGE_FLAG]) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[DateNightPlanner] Unable to parse parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'date-night-planner' });
    window[BRIDGE_FLAG] = true;
}

bootstrapIframeBridge();

// Date Night Planner - Main Application JavaScript

let config = {
    apiUrl: 'http://localhost:8080',
    version: '1.0.0'
};

// Load configuration
async function loadConfig() {
    try {
        const response = await fetch('/config');
        if (response.ok) {
            config = await response.json();
        }
    } catch (error) {
        console.error('Failed to load config:', error);
    }
}

// Tab navigation
function initTabs() {
    const tabs = document.querySelectorAll('.nav-tab');
    const contents = document.querySelectorAll('.tab-content');
    
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const targetTab = tab.dataset.tab;
            
            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            
            // Show corresponding content
            contents.forEach(content => {
                if (content.id === `${targetTab}-tab`) {
                    content.classList.add('active');
                } else {
                    content.classList.remove('active');
                }
            });
        });
    });
}

// Status monitoring
async function updateStatus() {
    const indicators = {
        api: document.getElementById('api-status'),
        db: document.getElementById('db-status'),
        wf: document.getElementById('wf-status')
    };
    
    try {
        // Check API health
        const apiResponse = await fetch(`${config.apiUrl}/health`);
        indicators.api.className = apiResponse.ok ? 'status-indicator online' : 'status-indicator offline';
        
        // Check database health
        const dbResponse = await fetch(`${config.apiUrl}/health/database`);
        indicators.db.className = dbResponse.ok ? 'status-indicator online' : 'status-indicator degraded';
        
        // Check workflow health
        const wfResponse = await fetch(`${config.apiUrl}/health/workflows`);
        indicators.wf.className = wfResponse.ok ? 'status-indicator online' : 'status-indicator degraded';
    } catch (error) {
        // If API is completely unreachable
        indicators.api.className = 'status-indicator offline';
        indicators.db.className = 'status-indicator offline';
        indicators.wf.className = 'status-indicator offline';
    }
}

// Budget slider update
function initBudgetSlider() {
    const budgetInput = document.getElementById('budget');
    const budgetDisplay = document.getElementById('budget-value');
    
    if (budgetInput && budgetDisplay) {
        budgetInput.addEventListener('input', () => {
            budgetDisplay.textContent = budgetInput.value;
        });
    }
}

// Get date suggestions
async function getSuggestions(event) {
    event.preventDefault();
    
    const coupleId = document.getElementById('couple-id').value;
    const dateType = document.getElementById('date-type').value;
    const budget = document.getElementById('budget').value;
    const preferredDate = document.getElementById('preferred-date').value;
    const weatherPref = document.querySelector('input[name="weather"]:checked')?.value || 'flexible';
    
    if (!coupleId) {
        alert('Please enter a couple ID');
        return;
    }
    
    const requestBody = {
        couple_id: coupleId,
        ...(dateType && { date_type: dateType }),
        ...(budget && { budget_max: parseFloat(budget) }),
        ...(preferredDate && { preferred_date: preferredDate }),
        weather_preference: weatherPref
    };
    
    try {
        const response = await fetch(`${config.apiUrl}/api/v1/dates/suggest`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });
        
        if (!response.ok) {
            throw new Error('Failed to get suggestions');
        }
        
        const data = await response.json();
        displaySuggestions(data.suggestions || []);
    } catch (error) {
        console.error('Error getting suggestions:', error);
        alert('Failed to get date suggestions. Please check if the API is running.');
    }
}

// Display suggestions
function displaySuggestions(suggestions) {
    const resultsContainer = document.getElementById('suggestions-results');
    const suggestionsList = document.getElementById('suggestions-list');
    
    if (suggestions.length === 0) {
        suggestionsList.innerHTML = '<p class="empty-state">No suggestions found. Try adjusting your criteria.</p>';
    } else {
        suggestionsList.innerHTML = suggestions.map((suggestion, index) => `
            <div class="suggestion-card" data-suggestion-id="${index}">
                <h3 class="suggestion-title">${suggestion.title || 'Date Suggestion'}</h3>
                <p class="suggestion-description">${suggestion.description || 'A wonderful date experience'}</p>
                <div class="suggestion-meta">
                    <span class="suggestion-cost">💰 $${suggestion.estimated_cost || 0}</span>
                    <span class="suggestion-duration">⏱️ ${suggestion.estimated_duration || '2 hours'}</span>
                </div>
                ${suggestion.confidence_score ? `
                    <div style="margin-top: 12px;">
                        <div style="background: #E6E6FA; border-radius: 8px; height: 8px; overflow: hidden;">
                            <div style="background: linear-gradient(90deg, #FF69B4, #9370DB); height: 100%; width: ${suggestion.confidence_score * 100}%;"></div>
                        </div>
                        <small style="color: #7A7A7A;">Confidence: ${Math.round(suggestion.confidence_score * 100)}%</small>
                    </div>
                ` : ''}
                <button class="btn btn-primary" style="margin-top: 16px; width: 100%;" onclick="planDate('${index}')">
                    Plan This Date 💝
                </button>
            </div>
        `).join('');
    }
    
    resultsContainer.style.display = 'block';
}

// Plan a date
async function planDate(suggestionIndex) {
    const coupleId = document.getElementById('couple-id').value;
    const preferredDate = document.getElementById('preferred-date').value || new Date().toISOString().split('T')[0];
    
    // Get the suggestion data (in a real app, we'd store this properly)
    const suggestionCard = document.querySelector(`[data-suggestion-id="${suggestionIndex}"]`);
    const title = suggestionCard.querySelector('.suggestion-title').textContent;
    const description = suggestionCard.querySelector('.suggestion-description').textContent;
    
    const requestBody = {
        couple_id: coupleId,
        selected_suggestion: {
            title: title,
            description: description,
            activities: [],
            estimated_cost: 100,
            estimated_duration: "2 hours"
        },
        planned_date: `${preferredDate}T19:00:00Z`
    };
    
    try {
        const response = await fetch(`${config.apiUrl}/api/v1/dates/plan`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });
        
        if (!response.ok) {
            throw new Error('Failed to create date plan');
        }
        
        const data = await response.json();
        alert(`✅ Date plan created successfully!\n\nDate: ${preferredDate}\nTitle: ${title}`);
        
        // Switch to plans tab
        document.querySelector('[data-tab="plan"]').click();
        loadPlans();
    } catch (error) {
        console.error('Error creating date plan:', error);
        alert('Failed to create date plan. Please try again.');
    }
}

// Load existing plans
async function loadPlans() {
    const plansList = document.getElementById('plans-list');
    
    // In a real implementation, we'd fetch plans from the API
    // For now, show a placeholder
    plansList.innerHTML = `
        <div class="suggestion-card">
            <h3 class="suggestion-title">Your Next Date</h3>
            <p class="suggestion-description">Check back here to see your planned dates!</p>
            <div class="suggestion-meta">
                <span>📅 Coming Soon</span>
            </div>
        </div>
    `;
}

// Initialize the application
document.addEventListener('DOMContentLoaded', async () => {
    await loadConfig();
    initTabs();
    initBudgetSlider();
    
    // Set up form submission
    const suggestionForm = document.getElementById('suggestion-form');
    if (suggestionForm) {
        suggestionForm.addEventListener('submit', getSuggestions);
    }
    
    // Set default date to tomorrow
    const dateInput = document.getElementById('preferred-date');
    if (dateInput) {
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        dateInput.value = tomorrow.toISOString().split('T')[0];
    }
    
    // Start status monitoring
    updateStatus();
    setInterval(updateStatus, 30000); // Update every 30 seconds
});