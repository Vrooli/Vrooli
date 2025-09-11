// Time Tools Timezone Converter JavaScript

// Configuration - Dynamic API discovery
let API_BASE = null;

// Discover API port dynamically
async function discoverApiPort() {
    // Check if API URL is provided via environment
    if (window.TIME_TOOLS_API_URL) {
        return window.TIME_TOOLS_API_URL;
    }
    
    // Try common port ranges for dynamic allocation
    const portRanges = [
        // Common dynamic port range
        { start: 8000, end: 8100 },
        { start: 3000, end: 3100 },
        { start: 5000, end: 5100 }
    ];
    
    for (const range of portRanges) {
        for (let port = range.start; port <= range.end; port++) {
            try {
                const url = `http://localhost:${port}`;
                const response = await fetch(`${url}/api/v1/health`, {
                    method: 'GET',
                    signal: AbortSignal.timeout(500) // 500ms timeout
                });
                
                if (response.ok) {
                    const data = await response.json();
                    if (data.service === 'time-tools') {
                        return url;
                    }
                }
            } catch (e) {
                // Port not available, continue
            }
        }
    }
    
    throw new Error('Time-tools API not found. Please ensure it is running.');
}

// Popular timezones for multi-display
const POPULAR_TIMEZONES = [
    { name: 'New York', timezone: 'America/New_York' },
    { name: 'Los Angeles', timezone: 'America/Los_Angeles' },
    { name: 'London', timezone: 'Europe/London' },
    { name: 'Paris', timezone: 'Europe/Paris' },
    { name: 'Tokyo', timezone: 'Asia/Tokyo' },
    { name: 'Sydney', timezone: 'Australia/Sydney' }
];

// Initialize converter
document.addEventListener('DOMContentLoaded', async function() {
    // Discover API first
    try {
        API_BASE = await discoverApiPort();
    } catch (error) {
        console.error('Failed to discover API:', error);
        // Show error to user
        document.body.insertAdjacentHTML('afterbegin', 
            '<div style="background: #ff6b6b; color: white; padding: 10px; text-align: center;">' +
            'Time-tools API not found. Please ensure it is running.' +
            '</div>'
        );
        return;
    }
    
    setupConverterForm();
    setupMultiTimezoneDisplay();
    checkAPIStatus();
    
    // Set default time to current time
    const now = new Date();
    const inputTime = document.getElementById('input-time');
    if (inputTime) {
        // Format for datetime-local input
        const year = now.getFullYear();
        const month = String(now.getMonth() + 1).padStart(2, '0');
        const day = String(now.getDate()).padStart(2, '0');
        const hours = String(now.getHours()).padStart(2, '0');
        const minutes = String(now.getMinutes()).padStart(2, '0');
        inputTime.value = `${year}-${month}-${day}T${hours}:${minutes}`;
    }
});

// Setup converter form
function setupConverterForm() {
    const form = document.getElementById('converter-form');
    if (!form) return;
    
    form.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        const inputTime = document.getElementById('input-time').value;
        const fromTimezone = document.getElementById('from-timezone').value;
        const toTimezone = document.getElementById('to-timezone').value;
        
        if (!inputTime || !fromTimezone || !toTimezone) {
            alert('Please fill in all fields');
            return;
        }
        
        // Convert datetime-local to ISO format
        const isoTime = new Date(inputTime).toISOString();
        
        try {
            if (!API_BASE) {
                alert('API not available');
                return;
            }
            
            const response = await fetch(`${API_BASE}/api/v1/time/convert`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    time: isoTime,
                    from_timezone: fromTimezone,
                    to_timezone: toTimezone
                })
            });
            
            if (response.ok) {
                const data = await response.json();
                displayConversionResult(data);
            } else {
                const error = await response.json();
                alert(`Conversion failed: ${error.error}`);
            }
        } catch (error) {
            alert('Failed to connect to API');
            console.error(error);
        }
    });
}

// Display conversion result
function displayConversionResult(data) {
    const resultContainer = document.getElementById('conversion-result');
    if (!resultContainer) return;
    
    resultContainer.style.display = 'block';
    
    // Format times for display
    const originalTime = new Date(data.original_time);
    const convertedTime = new Date(data.converted_time);
    
    document.getElementById('original-time').textContent = formatDateTime(originalTime) + ` (${data.from_timezone})`;
    document.getElementById('converted-time').textContent = formatDateTime(convertedTime) + ` (${data.to_timezone})`;
    
    // Calculate and display time difference
    const offsetHours = data.offset_minutes / 60;
    const sign = offsetHours >= 0 ? '+' : '';
    document.getElementById('time-difference').textContent = `${sign}${offsetHours} hours`;
    
    // Display DST status
    document.getElementById('dst-status').textContent = data.is_dst ? 'Active' : 'Not Active';
}

// Setup multi-timezone display
function setupMultiTimezoneDisplay() {
    const container = document.getElementById('multi-timezone-display');
    if (!container) return;
    
    updateMultiTimezoneDisplay();
    setInterval(updateMultiTimezoneDisplay, 60000); // Update every minute
}

// Update multi-timezone display
async function updateMultiTimezoneDisplay() {
    const container = document.getElementById('multi-timezone-display');
    if (!container) return;
    
    const now = new Date().toISOString();
    const currentTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    
    const timezoneCards = await Promise.all(
        POPULAR_TIMEZONES.map(async (tz) => {
            try {
                if (!API_BASE) return;
                
                const response = await fetch(`${API_BASE}/api/v1/time/convert`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        time: now,
                        from_timezone: currentTimezone,
                        to_timezone: tz.timezone
                    })
                });
                
                if (response.ok) {
                    const data = await response.json();
                    const convertedTime = new Date(data.converted_time);
                    
                    return `
                        <div class="timezone-card">
                            <div class="city">${tz.name}</div>
                            <div class="time">${convertedTime.toLocaleTimeString('en-US', {
                                hour: '2-digit',
                                minute: '2-digit',
                                hour12: false
                            })}</div>
                            <div class="date">${convertedTime.toLocaleDateString('en-US', {
                                month: 'short',
                                day: 'numeric'
                            })}</div>
                        </div>
                    `;
                }
            } catch (error) {
                console.error(`Failed to convert for ${tz.name}:`, error);
            }
            
            return `
                <div class="timezone-card">
                    <div class="city">${tz.name}</div>
                    <div class="time">--:--</div>
                    <div class="date">--</div>
                </div>
            `;
        })
    );
    
    container.innerHTML = timezoneCards.join('');
}

// Format date and time for display
function formatDateTime(date) {
    return date.toLocaleString('en-US', {
        weekday: 'short',
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: true
    });
}

// Check API status
async function checkAPIStatus() {
    const statusElement = document.getElementById('api-status');
    if (!statusElement) return;
    
    try {
        if (!API_BASE) {
            statusElement.textContent = 'API Not Discovered';
            statusElement.className = 'offline';
            return;
        }
        
        const response = await fetch(`${API_BASE}/api/v1/health`);
        if (response.ok) {
            statusElement.textContent = 'API Online';
            statusElement.className = 'online';
        } else {
            statusElement.textContent = 'API Offline';
            statusElement.className = 'offline';
        }
    } catch (error) {
        statusElement.textContent = 'API Offline';
        statusElement.className = 'offline';
    }
}