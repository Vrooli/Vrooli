// Time Tools Dashboard JavaScript

// Configuration - no hardcoded fallbacks
let API_BASE = null;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', async function() {
    // Initialize API discovery first
    await initializeAPI();
    
    // Start time updates immediately (don't require API)
    updateCurrentTime();
    setInterval(updateCurrentTime, 1000);
    
    // Check API status and start monitoring
    checkAPIStatus();
    setInterval(checkAPIStatus, 30000);
    
    // Load data that requires API
    if (API_BASE) {
        loadUpcomingEvents();
        loadAnalytics();
        setupQuickConvert();
    }
});

// Initialize API discovery
async function initializeAPI() {
    try {
        // Use the shared API discovery module
        if (window.timeToolsAPI) {
            API_BASE = await window.timeToolsAPI.discoverAPI();
            console.log('Time Tools API discovered at:', API_BASE);
        } else {
            throw new Error('API discovery module not loaded');
        }
    } catch (error) {
        console.error('Failed to discover Time Tools API:', error);
        API_BASE = null;
        updateAPIStatus('API Not Found', 'offline');
    }
}

// Update current time display
function updateCurrentTime() {
    const now = new Date();
    const timeElement = document.getElementById('local-time');
    const dateElement = document.getElementById('local-date');
    const timezoneElement = document.getElementById('timezone-name');
    
    if (timeElement) {
        timeElement.textContent = now.toLocaleTimeString('en-US', {
            hour12: false,
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    }
    
    if (dateElement) {
        dateElement.textContent = now.toLocaleDateString('en-US', {
            weekday: 'long',
            year: 'numeric',
            month: 'long',
            day: 'numeric'
        });
    }
    
    if (timezoneElement) {
        timezoneElement.textContent = Intl.DateTimeFormat().resolvedOptions().timeZone;
    }
}

// Update API status display
function updateAPIStatus(message, className) {
    const statusElement = document.getElementById('api-status');
    if (statusElement) {
        statusElement.textContent = message;
        statusElement.className = className;
    }
}

// Check API status
async function checkAPIStatus() {
    if (!API_BASE) {
        updateAPIStatus('API Not Available', 'offline');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/v1/health`);
        if (response.ok) {
            const data = await response.json();
            updateAPIStatus('API Online', 'online');
        } else {
            updateAPIStatus('API Offline', 'offline');
        }
    } catch (error) {
        updateAPIStatus('API Offline', 'offline');
    }
}

// Load upcoming events
async function loadUpcomingEvents() {
    const eventsContainer = document.getElementById('events-list');
    
    if (!API_BASE) {
        eventsContainer.innerHTML = '<div class="error">API not available</div>';
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/api/v1/events?status=scheduled`);
        if (response.ok) {
            const data = await response.json();
            const events = data.events || [];
            
            if (events.length === 0) {
                eventsContainer.innerHTML = '<div class="no-events">No upcoming events</div>';
            } else {
                eventsContainer.innerHTML = events.slice(0, 5).map(event => `
                    <div class="event-item">
                        <div class="event-time">${formatEventTime(event.start_time)}</div>
                        <div class="event-title">${event.title}</div>
                        ${event.location ? `<div class="event-location">📍 ${event.location}</div>` : ''}
                    </div>
                `).join('');
            }
        } else {
            eventsContainer.innerHTML = '<div class="error">Failed to load events</div>';
        }
    } catch (error) {
        eventsContainer.innerHTML = '<div class="error">Failed to connect to API</div>';
    }
}

// Load analytics
async function loadAnalytics() {
    // This would normally fetch from an analytics endpoint
    // For now, we'll use placeholder data
    const stats = {
        totalEvents: 12,
        hoursScheduled: 24,
        conflictsDetected: 2,
        timezonesUsed: 5
    };
    
    document.getElementById('total-events').textContent = stats.totalEvents;
    document.getElementById('hours-scheduled').textContent = stats.hoursScheduled;
    document.getElementById('conflicts-detected').textContent = stats.conflictsDetected;
    document.getElementById('timezones-used').textContent = stats.timezonesUsed;
}

// Setup quick timezone converter
function setupQuickConvert() {
    const timezoneSelect = document.getElementById('quick-timezone');
    const convertedDisplay = document.getElementById('converted-time');
    
    if (!timezoneSelect || !convertedDisplay) return;
    
    async function updateConversion() {
        const targetTimezone = timezoneSelect.value;
        const now = new Date();
        
        // If API is not available, fall back to client-side conversion
        if (!API_BASE) {
            try {
                const timeDisplay = now.toLocaleTimeString('en-US', {
                    timeZone: targetTimezone,
                    hour: '2-digit',
                    minute: '2-digit',
                    hour12: true
                });
                const dateDisplay = now.toLocaleDateString('en-US', {
                    timeZone: targetTimezone,
                    month: 'short',
                    day: 'numeric'
                });
                
                convertedDisplay.innerHTML = `
                    <span class="time">${timeDisplay}</span>
                    <span class="date">${dateDisplay}</span>
                `;
            } catch (error) {
                convertedDisplay.innerHTML = '<span class="error">Conversion failed</span>';
            }
            return;
        }
        
        try {
            const response = await fetch(`${API_BASE}/api/v1/time/convert`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    time: now.toISOString(),
                    from_timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                    to_timezone: targetTimezone
                })
            });
            
            if (response.ok) {
                const data = await response.json();
                const convertedTime = new Date(data.converted_time);
                
                convertedDisplay.innerHTML = `
                    <span class="time">${convertedTime.toLocaleTimeString('en-US', {
                        hour: '2-digit',
                        minute: '2-digit',
                        hour12: true
                    })}</span>
                    <span class="date">${convertedTime.toLocaleDateString('en-US', {
                        month: 'short',
                        day: 'numeric'
                    })}</span>
                `;
            }
        } catch (error) {
            convertedDisplay.innerHTML = '<span class="error">Conversion failed</span>';
        }
    }
    
    timezoneSelect.addEventListener('change', updateConversion);
    updateConversion(); // Initial conversion
}

// Format event time
function formatEventTime(isoTime) {
    const date = new Date(isoTime);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);
    
    let dayLabel = '';
    if (date.toDateString() === today.toDateString()) {
        dayLabel = 'Today';
    } else if (date.toDateString() === tomorrow.toDateString()) {
        dayLabel = 'Tomorrow';
    } else {
        dayLabel = date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    }
    
    const time = date.toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        hour12: true
    });
    
    return `${dayLabel} at ${time}`;
}