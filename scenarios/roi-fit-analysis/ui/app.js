import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

// ROI Fit Analysis Frontend Application
const API_URL = window.location.hostname === 'localhost' ? 'http://localhost:3000' : '/api';
const BRIDGE_STATE_KEY = '__roiFitBridgeInitialized';

const initializeIframeBridge = () => {
    if (!(typeof window !== 'undefined' && window.parent !== window)) {
        return;
    }

    if (window[BRIDGE_STATE_KEY]) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[ROIFitAnalysis] Unable to parse parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ appId: 'roi-fit-analysis', parentOrigin });
    window[BRIDGE_STATE_KEY] = true;
};

if (document.readyState === 'complete' || document.readyState === 'interactive') {
    initializeIframeBridge();
} else {
    document.addEventListener('DOMContentLoaded', initializeIframeBridge, { once: true });
}

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    loadOpportunities();
    setupFormHandler();
    updateMetrics();
});

// Handle form submission
function setupFormHandler() {
    const form = document.getElementById('analysisForm');
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const formData = {
            idea: document.getElementById('idea').value,
            budget: parseFloat(document.getElementById('budget').value),
            timeline: document.getElementById('timeline').value,
            skills: document.getElementById('skills').value.split(',').map(s => s.trim())
        };
        
        await analyzeROI(formData);
    });
}

// Analyze ROI
async function analyzeROI(data) {
    const loading = document.querySelector('.loading');
    const resultsPanel = document.getElementById('resultsPanel');
    
    loading.style.display = 'block';
    resultsPanel.style.display = 'none';
    
    try {
        const response = await fetch(`${API_URL}/analyze`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        // Display results
        setTimeout(() => {
            loading.style.display = 'none';
            displayResults(result.analysis);
        }, 1500);
        
    } catch (error) {
        console.error('Error analyzing ROI:', error);
        loading.style.display = 'none';
        alert('Error analyzing ROI. Please try again.');
    }
}

// Display analysis results
function displayResults(analysis) {
    const resultsPanel = document.getElementById('resultsPanel');
    resultsPanel.style.display = 'block';
    
    document.getElementById('roiScore').textContent = analysis.roi_score.toFixed(1);
    document.getElementById('paybackPeriod').textContent = `${analysis.payback_months} mo`;
    document.getElementById('riskLevel').textContent = analysis.risk_level;
    document.getElementById('marketSize').textContent = analysis.market_size;
    
    // Animate the results
    resultsPanel.style.opacity = '0';
    resultsPanel.style.transform = 'translateY(20px)';
    setTimeout(() => {
        resultsPanel.style.transition = 'all 0.5s ease';
        resultsPanel.style.opacity = '1';
        resultsPanel.style.transform = 'translateY(0)';
    }, 100);
}

// Load opportunities
async function loadOpportunities() {
    try {
        const response = await fetch(`${API_URL}/opportunities`);
        const opportunities = await response.json();
        
        const container = document.getElementById('opportunitiesList');
        container.innerHTML = '';
        
        opportunities.forEach(opp => {
            const card = createOpportunityCard(opp);
            container.appendChild(card);
        });
        
    } catch (error) {
        console.error('Error loading opportunities:', error);
    }
}

// Create opportunity card
function createOpportunityCard(opportunity) {
    const card = document.createElement('div');
    card.className = 'opportunity-card';
    card.innerHTML = `
        <div class="opp-name">${opportunity.name}</div>
        <div class="opp-stats">
            <span class="roi-badge">${opportunity.roi_score.toFixed(1)} ROI</span>
            <span>${opportunity.payback_months} mo payback</span>
        </div>
    `;
    
    card.addEventListener('click', () => {
        showOpportunityDetails(opportunity);
    });
    
    return card;
}

// Show opportunity details
function showOpportunityDetails(opportunity) {
    // Fill the form with opportunity data
    document.getElementById('idea').value = opportunity.name;
    document.getElementById('budget').value = opportunity.investment;
    
    // Scroll to form
    document.querySelector('.analysis-form').scrollIntoView({ 
        behavior: 'smooth',
        block: 'start'
    });
}

// Update dashboard metrics
function updateMetrics() {
    // Simulate real-time updates
    setInterval(() => {
        const metrics = document.querySelectorAll('.metric-value');
        metrics.forEach((metric, index) => {
            if (index === 2) { // Active analyses counter
                const current = parseInt(metric.textContent);
                const change = Math.random() > 0.5 ? 1 : 0;
                metric.textContent = current + change;
            }
        });
    }, 10000);
}

// Add keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.key === 'n') {
        e.preventDefault();
        document.getElementById('idea').focus();
    }
    
    if (e.ctrlKey && e.key === 'Enter') {
        e.preventDefault();
        document.getElementById('analysisForm').dispatchEvent(new Event('submit'));
    }
});

// Add smooth animations
const observerOptions = {
    threshold: 0.1,
    rootMargin: '0px 0px -100px 0px'
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.style.opacity = '1';
            entry.target.style.transform = 'translateY(0)';
        }
    });
}, observerOptions);

// Observe all cards
document.querySelectorAll('.metric-card, .opportunity-card').forEach(card => {
    card.style.opacity = '0';
    card.style.transform = 'translateY(20px)';
    card.style.transition = 'all 0.5s ease';
    observer.observe(card);
});
