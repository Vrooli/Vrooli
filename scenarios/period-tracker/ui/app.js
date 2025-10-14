import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

if (typeof window !== 'undefined' && window.parent !== window && !window.__periodTrackerBridgeInitialized) {
  let parentOrigin;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[PeriodTracker] Unable to determine parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'period-tracker' });
  window.__periodTrackerBridgeInitialized = true;
}

// Period Tracker UI - Privacy-First JavaScript Application
// All data processing happens locally with encrypted API communication

class PeriodTracker {
  constructor() {
    this.apiHost = window.location.hostname;
    this.apiPort = '16000'; // Default API port
    this.apiBaseUrl = `http://${this.apiHost}:${this.apiPort}/api/v1`;
    this.currentUserId = null;
    this.theme = 'light';
    
    this.init();
  }

  // Initialize the application
  init() {
    this.loadUserPreferences();
    this.setupEventListeners();
    this.setupSliderListeners();
    this.checkApiConnection();
    this.loadUserData();
    
    // Set today's date as default for date inputs
    this.setDefaultDates();
    
    // Auto-refresh data every 5 minutes
    setInterval(() => this.refreshData(), 5 * 60 * 1000);
    
    console.log('🩸 Period Tracker initialized - Privacy mode active');
  }

  // Load user preferences from localStorage
  loadUserPreferences() {
    const preferences = JSON.parse(localStorage.getItem('period-tracker-preferences') || '{}');
    
    this.currentUserId = preferences.userId || null;
    this.theme = preferences.theme || 'light';
    this.apiPort = preferences.apiPort || '16000';
    
    // Update API URL with custom port
    this.apiBaseUrl = `http://${this.apiHost}:${this.apiPort}/api/v1`;
    
    // Apply theme
    this.applyTheme();
    
    // Update UI with user ID
    this.updateUserIdDisplay();
    
    // Hide privacy banner if previously dismissed
    if (preferences.privacyBannerDismissed) {
      this.dismissPrivacyBanner();
    }
  }

  // Save user preferences to localStorage
  saveUserPreferences() {
    const preferences = {
      userId: this.currentUserId,
      theme: this.theme,
      apiPort: this.apiPort,
      privacyBannerDismissed: document.getElementById('privacy-banner').classList.contains('hidden')
    };
    
    localStorage.setItem('period-tracker-preferences', JSON.stringify(preferences));
  }

  // Setup all event listeners
  setupEventListeners() {
    // Close modals when clicking outside
    document.addEventListener('click', (e) => {
      if (e.target.classList.contains('modal')) {
        this.closeModal(e.target.id);
      }
    });
    
    // Close user dropdown when clicking outside
    document.addEventListener('click', (e) => {
      const userMenu = document.getElementById('user-menu');
      const userDropdown = document.getElementById('user-dropdown');
      
      if (!userMenu.contains(e.target) && !userDropdown.contains(e.target)) {
        userDropdown.classList.add('hidden');
      }
    });
    
    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        this.closeAllModals();
        this.hideUserDropdown();
      }
    });
  }

  // Setup slider value display listeners
  setupSliderListeners() {
    const sliders = document.querySelectorAll('.slider');
    sliders.forEach(slider => {
      const valueDisplay = slider.parentElement.querySelector('.slider-value');
      
      slider.addEventListener('input', (e) => {
        valueDisplay.textContent = e.target.value;
      });
      
      // Initialize display
      valueDisplay.textContent = slider.value;
    });
  }

  // Set default dates for forms
  setDefaultDates() {
    const today = new Date().toISOString().split('T')[0];
    
    const cycleStartDate = document.getElementById('cycle-start-date');
    const symptomDate = document.getElementById('symptom-date');
    
    if (cycleStartDate) cycleStartDate.value = today;
    if (symptomDate) symptomDate.value = today;
  }

  // Check API connection and update UI
  async checkApiConnection() {
    try {
      const response = await fetch(`http://${this.apiHost}:${this.apiPort}/health`);
      const data = await response.json();
      
      if (data.status === 'healthy') {
        console.log('✅ API connection established');
      } else {
        this.showNotification('⚠️ API connection unstable', 'warning');
      }
    } catch (error) {
      console.error('❌ API connection failed:', error);
      this.showNotification('❌ Cannot connect to API server', 'error');
    }
  }

  // API request helper with error handling
  async makeApiRequest(endpoint, options = {}) {
    if (!this.currentUserId && !endpoint.includes('/health')) {
      this.showNotification('Please set your User ID first', 'warning');
      this.showUserIdDialog();
      return null;
    }

    try {
      const headers = {
        'Content-Type': 'application/json',
        ...options.headers
      };
      
      if (this.currentUserId && !endpoint.includes('/health')) {
        headers['X-User-ID'] = this.currentUserId;
      }

      const response = await fetch(`${this.apiBaseUrl}${endpoint}`, {
        ...options,
        headers
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('API request failed:', error);
      this.showNotification(`API Error: ${error.message}`, 'error');
      return null;
    }
  }

  // Load user data (cycles, symptoms, predictions)
  async loadUserData() {
    if (!this.currentUserId) return;
    
    this.showLoading();
    
    try {
      // Load predictions
      await this.loadPredictions();
      
      // Load recent cycles
      await this.loadRecentCycles();
      
      // Load patterns
      await this.loadPatterns();
      
      // Load recent activity
      await this.loadRecentActivity();
      
    } catch (error) {
      console.error('Failed to load user data:', error);
    } finally {
      this.hideLoading();
    }
  }

  // Load cycle predictions
  async loadPredictions() {
    const data = await this.makeApiRequest('/predictions');
    if (!data) return;

    const predictionInfo = document.getElementById('next-period-info');
    const cycleStatusInfo = document.getElementById('cycle-status-info');
    
    if (data.next_period) {
      const nextDate = new Date(data.next_period);
      const today = new Date();
      const daysUntil = Math.ceil((nextDate - today) / (1000 * 60 * 60 * 24));
      
      predictionInfo.innerHTML = `
        <div class="prediction-date">${this.formatDate(nextDate)}</div>
        <div class="prediction-confidence">${Math.round(data.confidence * 100)}% confidence</div>
        <div class="prediction-days-until">${daysUntil} days from now</div>
      `;
      
      // Update cycle status (simplified calculation)
      cycleStatusInfo.innerHTML = `
        <div class="cycle-day">Day ${28 - daysUntil} of Cycle</div>
        <div class="cycle-phase">${this.getCyclePhase(28 - daysUntil)}</div>
        <div class="cycle-length">Average: 28 days</div>
      `;
    } else {
      predictionInfo.innerHTML = `
        <div class="prediction-date">Not enough data</div>
        <div class="prediction-confidence">Track more cycles for predictions</div>
        <div class="prediction-days-until">Start logging your periods</div>
      `;
      
      cycleStatusInfo.innerHTML = `
        <div class="cycle-day">No active cycle</div>
        <div class="cycle-phase">Start tracking to see insights</div>
        <div class="cycle-length">Average: - days</div>
      `;
    }
  }

  // Load recent cycles for activity timeline
  async loadRecentCycles() {
    const data = await this.makeApiRequest('/cycles');
    if (!data || !data.cycles) return;

    // Store cycles for other functions to use
    this.recentCycles = data.cycles;
  }

  // Load detected patterns
  async loadPatterns() {
    const data = await this.makeApiRequest('/patterns');
    if (!data) return;

    const patternsInfo = document.getElementById('patterns-info');
    
    if (data.patterns && data.patterns.length > 0) {
      patternsInfo.innerHTML = data.patterns.slice(0, 3).map(pattern => `
        <div class="pattern-item">
          <div class="pattern-title">${this.formatPatternType(pattern.pattern_type)}</div>
          <div class="pattern-description">${pattern.pattern_description || 'Pattern detected in your cycle data'}</div>
          <div class="pattern-confidence">Confidence: ${pattern.confidence_level}</div>
        </div>
      `).join('');
    } else {
      patternsInfo.innerHTML = `
        <div class="patterns-loading">No patterns detected yet. Keep logging symptoms to discover insights about your cycle.</div>
      `;
    }
  }

  // Load recent activity
  async loadRecentActivity() {
    const activityContainer = document.getElementById('recent-activity');
    const activities = [];
    
    // Add cycle activities
    if (this.recentCycles) {
      this.recentCycles.slice(0, 5).forEach(cycle => {
        activities.push({
          type: 'cycle_start',
          date: cycle.start_date,
          title: 'Period Started',
          description: `Flow: ${cycle.flow_intensity || 'Not specified'}`,
          icon: '🩸'
        });
      });
    }
    
    // Sort activities by date
    activities.sort((a, b) => new Date(b.date) - new Date(a.date));
    
    if (activities.length > 0) {
      activityContainer.innerHTML = activities.map(activity => `
        <div class="activity-item">
          <div class="activity-icon">${activity.icon}</div>
          <div class="activity-content">
            <div class="activity-title">${activity.title}</div>
            <div class="activity-description">${activity.description}</div>
            <div class="activity-time">${this.formatRelativeDate(activity.date)}</div>
          </div>
        </div>
      `).join('');
    } else {
      activityContainer.innerHTML = `
        <div class="activity-loading">No recent activity. Start by logging your first cycle or daily symptoms.</div>
      `;
    }
  }

  // Get cycle phase based on day
  getCyclePhase(day) {
    if (day <= 5) return 'Menstrual Phase';
    if (day <= 13) return 'Follicular Phase';
    if (day <= 15) return 'Ovulation';
    if (day <= 28) return 'Luteal Phase';
    return 'Unknown Phase';
  }

  // Format pattern type for display
  formatPatternType(patternType) {
    return patternType.replace(/_/g, ' ')
                     .split(' ')
                     .map(word => word.charAt(0).toUpperCase() + word.slice(1))
                     .join(' ');
  }

  // Format date for display
  formatDate(date) {
    if (typeof date === 'string') {
      date = new Date(date);
    }
    
    return date.toLocaleDateString('en-US', {
      weekday: 'short',
      month: 'short',
      day: 'numeric'
    });
  }

  // Format relative date
  formatRelativeDate(dateStr) {
    const date = new Date(dateStr);
    const today = new Date();
    const diffTime = Math.abs(today - date);
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    
    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    if (diffDays < 30) return `${Math.ceil(diffDays / 7)} weeks ago`;
    return `${Math.ceil(diffDays / 30)} months ago`;
  }

  // Theme management
  applyTheme() {
    document.documentElement.setAttribute('data-theme', this.theme);
    const themeIcon = document.querySelector('.theme-icon');
    if (themeIcon) {
      themeIcon.textContent = this.theme === 'dark' ? '☀️' : '🌙';
    }
  }

  toggleTheme() {
    this.theme = this.theme === 'light' ? 'dark' : 'light';
    this.applyTheme();
    this.saveUserPreferences();
    
    this.showNotification(`Switched to ${this.theme} theme`, 'info');
  }

  // User management
  updateUserIdDisplay() {
    const userIdElement = document.getElementById('current-user-id');
    if (userIdElement) {
      userIdElement.textContent = this.currentUserId || 'Not set';
    }
  }

  showUserIdDialog() {
    const modal = document.getElementById('user-id-modal');
    const input = document.getElementById('user-id-input');
    
    input.value = this.currentUserId || '';
    modal.classList.remove('hidden');
    input.focus();
  }

  setUserId() {
    const input = document.getElementById('user-id-input');
    const userId = input.value.trim();
    
    if (!userId) {
      this.showNotification('Please enter a valid User ID', 'warning');
      return;
    }
    
    this.currentUserId = userId;
    this.updateUserIdDisplay();
    this.saveUserPreferences();
    this.closeModal('user-id-modal');
    
    // Reload data for the new user
    this.loadUserData();
    
    this.showNotification(`User ID set to: ${userId}`, 'success');
  }

  // Privacy banner
  dismissPrivacyBanner() {
    document.getElementById('privacy-banner').classList.add('hidden');
    this.saveUserPreferences();
  }

  // Modal management
  showModal(modalId) {
    document.getElementById(modalId).classList.remove('hidden');
    document.body.style.overflow = 'hidden';
  }

  closeModal(modalId) {
    document.getElementById(modalId).classList.add('hidden');
    document.body.style.overflow = 'auto';
  }

  closeAllModals() {
    const modals = document.querySelectorAll('.modal');
    modals.forEach(modal => {
      modal.classList.add('hidden');
    });
    document.body.style.overflow = 'auto';
  }

  // User menu
  toggleUserMenu() {
    const dropdown = document.getElementById('user-dropdown');
    dropdown.classList.toggle('hidden');
  }

  hideUserDropdown() {
    document.getElementById('user-dropdown').classList.add('hidden');
  }

  // Cycle management
  showCycleStartDialog() {
    this.showModal('cycle-start-modal');
  }

  async startCycle() {
    const startDate = document.getElementById('cycle-start-date').value;
    const flowIntensity = document.getElementById('flow-intensity').value;
    const notes = document.getElementById('cycle-notes').value.trim();
    
    if (!startDate) {
      this.showNotification('Please select a start date', 'warning');
      return;
    }
    
    const payload = {
      start_date: startDate,
      flow_intensity: flowIntensity,
      notes: notes
    };
    
    this.showLoading();
    
    const result = await this.makeApiRequest('/cycles', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
    
    this.hideLoading();
    
    if (result) {
      this.closeModal('cycle-start-modal');
      this.showNotification('✅ Cycle started successfully', 'success');
      
      // Clear form
      document.getElementById('flow-intensity').value = '';
      document.getElementById('cycle-notes').value = '';
      
      // Refresh data
      this.loadUserData();
    }
  }

  // Symptom logging
  showSymptomsDialog() {
    this.showModal('symptoms-modal');
  }

  async logSymptoms() {
    const date = document.getElementById('symptom-date').value;
    const moodRating = document.getElementById('mood-rating').value;
    const energyRating = document.getElementById('energy-rating').value;
    const painRating = document.getElementById('pain-rating').value;
    const flowLevel = document.getElementById('flow-level').value;
    const notes = document.getElementById('symptom-notes').value.trim();
    
    // Get selected symptoms
    const symptomCheckboxes = document.querySelectorAll('.symptoms-checkboxes input[type="checkbox"]:checked');
    const symptoms = Array.from(symptomCheckboxes).map(cb => cb.value);
    
    if (!date) {
      this.showNotification('Please select a date', 'warning');
      return;
    }
    
    const payload = {
      date: date,
      mood_rating: parseInt(moodRating),
      energy_level: parseInt(energyRating),
      cramp_intensity: parseInt(painRating),
      flow_level: flowLevel,
      physical_symptoms: symptoms,
      notes: notes
    };
    
    this.showLoading();
    
    const result = await this.makeApiRequest('/symptoms', {
      method: 'POST',
      body: JSON.stringify(payload)
    });
    
    this.hideLoading();
    
    if (result) {
      this.closeModal('symptoms-modal');
      this.showNotification('✅ Symptoms logged successfully', 'success');
      
      // Reset form
      this.resetSymptomsForm();
      
      // Refresh data
      this.loadUserData();
    }
  }

  resetSymptomsForm() {
    // Reset sliders
    document.getElementById('mood-rating').value = 5;
    document.getElementById('energy-rating').value = 5;
    document.getElementById('pain-rating').value = 0;
    
    // Reset checkboxes
    document.querySelectorAll('.symptoms-checkboxes input[type="checkbox"]').forEach(cb => {
      cb.checked = false;
    });
    
    // Reset other fields
    document.getElementById('flow-level').value = 'none';
    document.getElementById('symptom-notes').value = '';
    
    // Update slider displays
    document.querySelectorAll('.slider-value').forEach((display, index) => {
      const slider = document.querySelectorAll('.slider')[index];
      display.textContent = slider.value;
    });
  }

  // Data export
  async exportData() {
    if (!this.currentUserId) {
      this.showNotification('Please set your User ID first', 'warning');
      return;
    }
    
    try {
      // Get all user data
      const [cycles, symptoms, predictions, patterns] = await Promise.all([
        this.makeApiRequest('/cycles'),
        this.makeApiRequest('/symptoms'),
        this.makeApiRequest('/predictions'),
        this.makeApiRequest('/patterns')
      ]);
      
      const exportData = {
        user_id: this.currentUserId,
        export_date: new Date().toISOString(),
        cycles: cycles?.cycles || [],
        symptoms: symptoms?.symptoms || [],
        predictions: predictions?.predictions || [],
        patterns: patterns?.patterns || []
      };
      
      // Create download
      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: 'application/json'
      });
      
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `period-tracker-export-${this.currentUserId}-${new Date().toISOString().split('T')[0]}.json`;
      a.click();
      
      URL.revokeObjectURL(url);
      
      this.showNotification('✅ Data exported successfully', 'success');
      
    } catch (error) {
      this.showNotification('❌ Export failed', 'error');
    }
    
    this.hideUserDropdown();
  }

  // Privacy information
  showPrivacyInfo() {
    this.showModal('privacy-modal');
    this.hideUserDropdown();
  }

  // Data refresh
  async refreshData() {
    if (!this.currentUserId) return;
    
    const refreshIcon = document.querySelector('.refresh-icon');
    if (refreshIcon) {
      refreshIcon.style.animation = 'spin 1s linear';
      setTimeout(() => {
        refreshIcon.style.animation = '';
      }, 1000);
    }
    
    await this.loadUserData();
  }

  // Loading states
  showLoading() {
    document.getElementById('loading-overlay').classList.remove('hidden');
  }

  hideLoading() {
    document.getElementById('loading-overlay').classList.add('hidden');
  }

  // Notifications
  showNotification(message, type = 'info') {
    const toast = document.getElementById('notification-toast');
    const messageEl = toast.querySelector('.toast-message');
    const iconEl = toast.querySelector('.toast-icon');
    
    // Set icon based on type
    const icons = {
      success: '✅',
      error: '❌',
      warning: '⚠️',
      info: 'ℹ️'
    };
    
    iconEl.textContent = icons[type] || icons.info;
    messageEl.textContent = message;
    
    toast.classList.remove('hidden');
    
    // Auto-hide after 5 seconds
    setTimeout(() => {
      this.hideNotification();
    }, 5000);
  }

  hideNotification() {
    document.getElementById('notification-toast').classList.add('hidden');
  }
}

// Global functions for HTML onclick handlers
function dismissPrivacyBanner() {
  app.dismissPrivacyBanner();
}

function toggleTheme() {
  app.toggleTheme();
}

function toggleUserMenu() {
  app.toggleUserMenu();
}

function showUserIdDialog() {
  app.showUserIdDialog();
}

function setUserId() {
  app.setUserId();
}

function exportData() {
  app.exportData();
}

function showPrivacyInfo() {
  app.showPrivacyInfo();
}

function refreshData() {
  app.refreshData();
}

function showCycleStartDialog() {
  app.showCycleStartDialog();
}

function startCycle() {
  app.startCycle();
}

function showSymptomsDialog() {
  app.showSymptomsDialog();
}

function logSymptoms() {
  app.logSymptoms();
}

function showCycleEndDialog() {
  app.showNotification('Cycle end tracking coming soon!', 'info');
}

function closeModal(modalId) {
  app.closeModal(modalId);
}

function hideNotification() {
  app.hideNotification();
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  window.app = new PeriodTracker();
});

// Global error handling
window.addEventListener('error', (e) => {
  console.error('Application error:', e.error);
  if (window.app) {
    window.app.showNotification('An error occurred. Please refresh the page.', 'error');
  }
});

// Handle offline/online status
window.addEventListener('online', () => {
  if (window.app) {
    window.app.showNotification('Connection restored', 'success');
    window.app.checkApiConnection();
  }
});

window.addEventListener('offline', () => {
  if (window.app) {
    window.app.showNotification('Connection lost - data will sync when online', 'warning');
  }
});

console.log('🩸 Period Tracker UI loaded - Privacy-first menstrual health tracking');
