// Time Tools Duration Calculator JavaScript

// Configuration - Standards-compliant API discovery
let API_BASE = null;

// Initialize API discovery
async function initializeAPI() {
    try {
        // Use the shared API discovery module
        if (window.timeToolsAPI) {
            API_BASE = await window.timeToolsAPI.discoverAPI();
            console.log('Time Tools API discovered at:', API_BASE);
            updateAPIStatus('API Connected', 'online');
        } else {
            throw new Error('API discovery module not loaded');
        }
    } catch (error) {
        console.error('Failed to discover Time Tools API:', error);
        API_BASE = null;
        updateAPIStatus('API Not Found', 'offline');
        // Show error to user
        document.body.insertAdjacentHTML('afterbegin', 
            '<div style="background: #ff6b6b; color: white; padding: 10px; text-align: center;">' +
            'Time-tools API not found. Please ensure the scenario is running: vrooli scenario run time-tools' +
            '</div>'
        );
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

// Initialize calculator
document.addEventListener('DOMContentLoaded', async function() {
    // Initialize API discovery first
    await initializeAPI();

    setupDurationForm();
    setupQuickCalculations();
    
    // Set default times (current time and 1 hour later)
    const now = new Date();
    const later = new Date(now.getTime() + 60 * 60 * 1000); // 1 hour later
    
    const startTimeInput = document.getElementById('start-time');
    const endTimeInput = document.getElementById('end-time');
    
    if (startTimeInput) startTimeInput.value = formatDateTimeLocal(now);
    if (endTimeInput) endTimeInput.value = formatDateTimeLocal(later);
    
    // Set default timezone to user's timezone
    const userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const timezoneSelect = document.getElementById('timezone');
    if (timezoneSelect && [...timezoneSelect.options].some(opt => opt.value === userTimezone)) {
        timezoneSelect.value = userTimezone;
    }
});

function setupDurationForm() {
    const form = document.getElementById('duration-form');
    if (!form) return;
    
    form.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        const startTime = document.getElementById('start-time').value;
        const endTime = document.getElementById('end-time').value;
        const timezone = document.getElementById('timezone').value;
        const excludeWeekends = document.getElementById('exclude-weekends').checked;
        const excludeHolidays = document.getElementById('exclude-holidays').checked;
        const businessHoursOnly = document.getElementById('business-hours-only').checked;
        
        if (!startTime || !endTime) {
            alert('Please fill in both start and end times');
            return;
        }
        
        if (!API_BASE) {
            alert('API not available');
            return;
        }
        
        const startISO = new Date(startTime).toISOString();
        const endISO = new Date(endTime).toISOString();
        
        try {
            const response = await fetch(`${API_BASE}/api/v1/time/duration`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    start_time: startISO,
                    end_time: endISO,
                    timezone: timezone,
                    exclude_weekends: excludeWeekends,
                    exclude_holidays: excludeHolidays,
                    business_hours_only: businessHoursOnly
                })
            });
            
            if (response.ok) {
                const data = await response.json();
                displayDurationResult(data);
            } else {
                const error = await response.json();
                alert(`Calculation failed: ${error.error}`);
            }
        } catch (error) {
            alert('Failed to connect to API');
            console.error(error);
        }
    });
}

function displayDurationResult(data) {
    const resultContainer = document.getElementById('duration-result');
    if (!resultContainer) return;
    
    resultContainer.style.display = 'block';
    
    // Format total duration as human readable
    const totalHours = Math.floor(data.total_hours);
    const totalMinutes = Math.floor((data.total_hours % 1) * 60);
    const durationText = `${totalHours}h ${totalMinutes}m`;
    
    const totalDurationEl = document.getElementById('total-duration');
    const totalHoursEl = document.getElementById('total-hours');
    const totalDaysEl = document.getElementById('total-days');
    
    if (totalDurationEl) totalDurationEl.textContent = durationText;
    if (totalHoursEl) totalHoursEl.textContent = `${data.total_hours.toFixed(2)} hours`;
    if (totalDaysEl) totalDaysEl.textContent = `${data.total_days.toFixed(2)} days`;
    
    // Show business time results if applicable
    if (data.business_hours !== undefined) {
        const businessHoursResult = document.getElementById('business-hours-result');
        const businessHoursEl = document.getElementById('business-hours');
        if (businessHoursResult) businessHoursResult.style.display = 'block';
        if (businessHoursEl) businessHoursEl.textContent = `${data.business_hours.toFixed(2)} hours`;
    }
    
    if (data.business_days !== undefined) {
        const businessDaysResult = document.getElementById('business-days-result');
        const businessDaysEl = document.getElementById('business-days');
        if (businessDaysResult) businessDaysResult.style.display = 'block';
        if (businessDaysEl) businessDaysEl.textContent = `${data.business_days} days`;
    }
}

function setupQuickCalculations() {
    const quickButtons = document.querySelectorAll('.quick-calc-btn');
    quickButtons.forEach(button => {
        button.addEventListener('click', function() {
            const duration = this.dataset.duration;
            const startTimeInput = document.getElementById('start-time');
            const endTimeInput = document.getElementById('end-time');
            
            if (!startTimeInput || !endTimeInput) return;
            
            if (!startTimeInput.value) {
                startTimeInput.value = formatDateTimeLocal(new Date());
            }
            
            const startTime = new Date(startTimeInput.value);
            let endTime = new Date(startTime);
            
            // Parse duration
            const match = duration.match(/^(\d+)([hdwm])$/);
            if (match) {
                const amount = parseInt(match[1]);
                const unit = match[2];
                
                switch (unit) {
                    case 'h':
                        endTime.setHours(endTime.getHours() + amount);
                        break;
                    case 'd':
                        endTime.setDate(endTime.getDate() + amount);
                        break;
                    case 'w':
                        endTime.setDate(endTime.getDate() + amount * 7);
                        break;
                    case 'm':
                        endTime.setMonth(endTime.getMonth() + amount);
                        break;
                }
                
                endTimeInput.value = formatDateTimeLocal(endTime);
            }
        });
    });
}

function formatDateTimeLocal(date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}`;
}