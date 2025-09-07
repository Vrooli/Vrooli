// Pregnancy Tracker - Main Application JavaScript

// Configuration
const API_BASE = `http://localhost:${window.location.port.replace('37', '17')}`;
const STORAGE_KEY = 'pregnancy-tracker-data';

// Application State
let state = {
    userId: null,
    pregnancy: null,
    currentWeek: 0,
    dueDate: null,
    logs: [],
    appointments: [],
    emergencyInfo: {}
};

// Initialize Application
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
    setupEventListeners();
    loadStoredData();
});

function initializeApp() {
    // Check if user has existing data
    const storedData = localStorage.getItem(STORAGE_KEY);
    if (storedData) {
        state = JSON.parse(storedData);
        hideSetupModal();
        loadDashboard();
    } else {
        showSetupModal();
    }
}

// Event Listeners
function setupEventListeners() {
    // Navigation
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            switchTab(e.target.dataset.tab);
        });
    });

    // Search
    document.getElementById('searchToggle').addEventListener('click', toggleSearch);
    document.getElementById('searchInput').addEventListener('input', performSearch);

    // Emergency Card
    document.getElementById('emergencyCard').addEventListener('click', showEmergencyModal);

    // Setup Form
    document.getElementById('setupForm').addEventListener('submit', handleSetup);

    // Daily Log Form
    document.getElementById('dailyLogForm').addEventListener('submit', handleDailyLog);

    // Mood/Energy Sliders
    document.getElementById('mood').addEventListener('input', (e) => {
        document.getElementById('moodValue').textContent = e.target.value;
    });
    document.getElementById('energy').addEventListener('input', (e) => {
        document.getElementById('energyValue').textContent = e.target.value;
    });

    // Quick Actions
    document.getElementById('logSymptom').addEventListener('click', () => switchTab('daily-log'));
    document.getElementById('kickCounter').addEventListener('click', startKickCounter);
    document.getElementById('contractionTimer').addEventListener('click', startContractionTimer);
    document.getElementById('photoJournal').addEventListener('click', openPhotoJournal);

    // Tools
    document.getElementById('kickCounterTool').querySelector('.tool-btn').addEventListener('click', startKickCounter);
    document.getElementById('contractionTimerTool').querySelector('.tool-btn').addEventListener('click', startContractionTimer);
    document.getElementById('dueDateCalculator').querySelector('.tool-btn').addEventListener('click', showDueDateCalculator);
    document.getElementById('exportData').querySelectorAll('.tool-btn').forEach((btn, index) => {
        btn.addEventListener('click', () => exportData(index === 0 ? 'pdf' : 'json'));
    });
    document.getElementById('partnerAccess').querySelector('.tool-btn').addEventListener('click', managePartnerAccess);
    document.getElementById('askAI').querySelector('.tool-btn').addEventListener('click', showAIAssistant);

    // Appointments
    document.getElementById('addAppointment').addEventListener('click', showAddAppointment);

    // View Citations
    document.getElementById('viewCitations').addEventListener('click', showCitations);

    // Modal Close
    document.querySelectorAll('.close-btn').forEach(btn => {
        btn.addEventListener('click', closeModal);
    });
    document.getElementById('modalOverlay').addEventListener('click', closeModal);
}

// Tab Switching
function switchTab(tabName) {
    // Update nav buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === tabName);
    });

    // Update content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === tabName);
    });

    // Load tab-specific data
    if (tabName === 'timeline') loadTimeline();
    if (tabName === 'appointments') loadAppointments();
}

// Search Functionality
function toggleSearch() {
    const searchBar = document.getElementById('searchBar');
    searchBar.classList.toggle('hidden');
    if (!searchBar.classList.contains('hidden')) {
        document.getElementById('searchInput').focus();
    }
}

async function performSearch() {
    const query = document.getElementById('searchInput').value;
    if (query.length < 2) {
        document.getElementById('searchResults').innerHTML = '';
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/v1/search?q=${encodeURIComponent(query)}`);
        const results = await response.json();
        displaySearchResults(results);
    } catch (error) {
        console.error('Search error:', error);
    }
}

function displaySearchResults(results) {
    const container = document.getElementById('searchResults');
    if (!results || results.length === 0) {
        container.innerHTML = '<p>No results found</p>';
        return;
    }

    container.innerHTML = results.map(result => `
        <div class="search-result">
            <h4>${result.title}</h4>
            <p>${result.snippet}</p>
        </div>
    `).join('');
}

// Setup Modal
function showSetupModal() {
    document.getElementById('setupModal').classList.remove('hidden');
    document.getElementById('modalOverlay').classList.remove('hidden');
}

function hideSetupModal() {
    document.getElementById('setupModal').classList.add('hidden');
    document.getElementById('modalOverlay').classList.add('hidden');
}

async function handleSetup(e) {
    e.preventDefault();
    
    const userId = document.getElementById('userId').value;
    const dateType = document.querySelector('input[name="dateType"]:checked').value;
    const dateInput = document.getElementById('dateInput').value;
    const importData = document.getElementById('importPeriodData').checked;

    // Calculate dates based on input type
    let lmpDate, dueDate;
    const inputDate = new Date(dateInput);
    
    if (dateType === 'lmp') {
        lmpDate = inputDate;
        dueDate = new Date(inputDate);
        dueDate.setDate(dueDate.getDate() + 280); // 40 weeks
    } else if (dateType === 'conception') {
        lmpDate = new Date(inputDate);
        lmpDate.setDate(lmpDate.getDate() - 14); // Approximate LMP
        dueDate = new Date(inputDate);
        dueDate.setDate(dueDate.getDate() + 266); // 38 weeks from conception
    } else {
        dueDate = inputDate;
        lmpDate = new Date(inputDate);
        lmpDate.setDate(lmpDate.getDate() - 280);
    }

    // Update state
    state.userId = userId;
    state.pregnancy = {
        lmpDate: lmpDate.toISOString(),
        dueDate: dueDate.toISOString(),
        startDate: new Date().toISOString()
    };
    
    // Calculate current week
    const today = new Date();
    const weeksDiff = Math.floor((today - lmpDate) / (7 * 24 * 60 * 60 * 1000));
    state.currentWeek = Math.max(0, Math.min(42, weeksDiff));

    // Import period tracker data if requested
    if (importData) {
        await importPeriodTrackerData();
    }

    // Save to API
    try {
        const response = await fetch(`${API_BASE}/api/v1/pregnancy/start`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-User-ID': state.userId
            },
            body: JSON.stringify({
                lmp_date: state.pregnancy.lmpDate,
                due_date: state.pregnancy.dueDate
            })
        });

        if (response.ok) {
            const data = await response.json();
            state.pregnancy.id = data.id;
        }
    } catch (error) {
        console.error('Error saving pregnancy data:', error);
    }

    // Save to local storage
    saveState();
    
    // Hide modal and load dashboard
    hideSetupModal();
    loadDashboard();
}

async function importPeriodTrackerData() {
    try {
        const response = await fetch(`http://localhost:${window.location.port.replace('37', '36')}/api/v1/cycles`, {
            headers: {
                'X-User-ID': state.userId
            }
        });
        
        if (response.ok) {
            const cycles = await response.json();
            // Use the last cycle for more accurate dating
            if (cycles && cycles.length > 0) {
                const lastCycle = cycles[cycles.length - 1];
                console.log('Imported period data:', lastCycle);
            }
        }
    } catch (error) {
        console.log('Period tracker not available or no data found');
    }
}

// Dashboard
async function loadDashboard() {
    if (!state.pregnancy) return;

    // Calculate pregnancy progress
    const today = new Date();
    const lmpDate = new Date(state.pregnancy.lmpDate);
    const dueDate = new Date(state.pregnancy.dueDate);
    
    const totalDays = Math.floor((dueDate - lmpDate) / (24 * 60 * 60 * 1000));
    const daysPassed = Math.floor((today - lmpDate) / (24 * 60 * 60 * 1000));
    const daysRemaining = Math.floor((dueDate - today) / (24 * 60 * 60 * 1000));
    
    const weekNumber = Math.floor(daysPassed / 7);
    const dayNumber = daysPassed % 7;
    const progressPercent = Math.min(100, Math.round((daysPassed / totalDays) * 100));

    // Update UI
    document.getElementById('weekNumber').textContent = `Week ${weekNumber}`;
    document.getElementById('daysRemaining').textContent = 
        daysRemaining > 0 ? `${daysRemaining} days until due date` : 
        daysRemaining === 0 ? 'Due today!' : 
        `${Math.abs(daysRemaining)} days overdue`;
    
    document.getElementById('progressPercent').textContent = `${progressPercent}%`;
    
    // Update progress circle
    const circumference = 2 * Math.PI * 90;
    const offset = circumference - (progressPercent / 100) * circumference;
    document.getElementById('progressCircle').style.strokeDashoffset = offset;

    // Load week information
    await loadWeekInfo(weekNumber);
    
    // Load daily tip
    loadDailyTip();
}

async function loadWeekInfo(week) {
    try {
        const response = await fetch(`${API_BASE}/api/v1/content/week/${week}`);
        if (response.ok) {
            const info = await response.json();
            document.getElementById('weekInfo').innerHTML = `
                <h4>${info.title}</h4>
                <p><strong>Baby's Size:</strong> ${info.size}</p>
                <p><strong>Development:</strong> ${info.development}</p>
                <p><strong>Your Body:</strong> ${info.bodyChanges}</p>
                <p><strong>Tips:</strong> ${info.tips}</p>
            `;
        }
    } catch (error) {
        console.error('Error loading week info:', error);
        document.getElementById('weekInfo').innerHTML = 
            '<p>Week information will be available once the server is running.</p>';
    }
}

function loadDailyTip() {
    const tips = [
        "Stay hydrated! Aim for 8-10 glasses of water daily during pregnancy.",
        "Take your prenatal vitamins daily, especially folic acid.",
        "Gentle exercise like walking can help with pregnancy symptoms.",
        "Get plenty of rest - your body is working hard!",
        "Eat small, frequent meals to help with nausea.",
        "Practice good posture to reduce back pain.",
        "Stay connected with your healthcare provider.",
        "Document your journey - you'll treasure these memories!",
        "Listen to your body and rest when needed.",
        "Connect with other expecting parents for support."
    ];
    
    const todayIndex = new Date().getDate() % tips.length;
    document.getElementById('dailyTip').textContent = tips[todayIndex];
}

// Daily Log
async function handleDailyLog(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const symptoms = Array.from(document.querySelectorAll('input[name="symptoms"]:checked'))
        .map(cb => cb.value);
    
    const logData = {
        date: new Date().toISOString(),
        symptoms: symptoms,
        weight: formData.get('weight'),
        blood_pressure: formData.get('bloodPressure'),
        mood: parseInt(formData.get('mood')),
        energy: parseInt(formData.get('energy')),
        notes: formData.get('notes')
    };

    // Save to API
    try {
        const response = await fetch(`${API_BASE}/api/v1/logs/daily`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-User-ID': state.userId
            },
            body: JSON.stringify(logData)
        });

        if (response.ok) {
            showNotification('Daily log saved successfully!');
            e.target.reset();
            document.getElementById('moodValue').textContent = '5';
            document.getElementById('energyValue').textContent = '5';
        }
    } catch (error) {
        console.error('Error saving daily log:', error);
        // Save locally as backup
        state.logs.push(logData);
        saveState();
        showNotification('Daily log saved locally');
    }
}

// Timeline
function loadTimeline() {
    if (!state.pregnancy) return;
    
    const today = new Date();
    const lmpDate = new Date(state.pregnancy.lmpDate);
    const dueDate = new Date(state.pregnancy.dueDate);
    
    const totalDays = Math.floor((dueDate - lmpDate) / (24 * 60 * 60 * 1000));
    const daysPassed = Math.floor((today - lmpDate) / (24 * 60 * 60 * 1000));
    const progressPercent = Math.min(100, (daysPassed / totalDays) * 100);
    
    // Update timeline progress
    document.getElementById('timelineProgress').style.width = `${progressPercent}%`;
    document.getElementById('timelineMarker').style.left = `${progressPercent}%`;
    
    // Load milestones
    const milestones = [
        { week: 4, title: "Missed Period", description: "Pregnancy can be detected" },
        { week: 6, title: "Heartbeat Detectable", description: "Baby's heart starts beating" },
        { week: 8, title: "First Prenatal Visit", description: "Initial checkup and ultrasound" },
        { week: 12, title: "End of First Trimester", description: "Risk of miscarriage decreases" },
        { week: 16, title: "Gender Can Be Determined", description: "If you choose to know" },
        { week: 18, title: "Anatomy Scan", description: "Detailed ultrasound examination" },
        { week: 20, title: "Halfway Point", description: "You're halfway there!" },
        { week: 24, title: "Viability", description: "Baby could survive if born prematurely" },
        { week: 28, title: "Third Trimester Begins", description: "Final stretch begins" },
        { week: 32, title: "Baby Shower Time", description: "Common time for celebrations" },
        { week: 36, title: "Full Term Soon", description: "Baby is almost ready" },
        { week: 37, title: "Full Term", description: "Baby can be born any time now" },
        { week: 40, title: "Due Date", description: "Your estimated due date" }
    ];
    
    const currentWeek = Math.floor(daysPassed / 7);
    const milestoneList = document.getElementById('milestoneList');
    
    milestoneList.innerHTML = milestones.map(milestone => {
        const isPassed = currentWeek >= milestone.week;
        const isCurrent = Math.abs(currentWeek - milestone.week) <= 1;
        
        return `
            <div class="milestone ${isPassed ? 'passed' : ''} ${isCurrent ? 'current' : ''}">
                <div class="milestone-week">Week ${milestone.week}</div>
                <div class="milestone-content">
                    <h4>${milestone.title}</h4>
                    <p>${milestone.description}</p>
                </div>
            </div>
        `;
    }).join('');
}

// Appointments
async function loadAppointments() {
    try {
        const response = await fetch(`${API_BASE}/api/v1/appointments/upcoming`, {
            headers: {
                'X-User-ID': state.userId
            }
        });
        
        if (response.ok) {
            const appointments = await response.json();
            displayAppointments(appointments);
        }
    } catch (error) {
        console.error('Error loading appointments:', error);
        displayAppointments(state.appointments);
    }
}

function displayAppointments(appointments) {
    const container = document.getElementById('appointmentsList');
    
    if (!appointments || appointments.length === 0) {
        container.innerHTML = '<p>No upcoming appointments. Click "Add Appointment" to schedule one.</p>';
        return;
    }
    
    container.innerHTML = appointments.map(apt => `
        <div class="appointment-card">
            <h4>${apt.type}</h4>
            <p><strong>Date:</strong> ${new Date(apt.date).toLocaleDateString()}</p>
            <p><strong>Time:</strong> ${new Date(apt.date).toLocaleTimeString()}</p>
            <p><strong>Provider:</strong> ${apt.provider || 'Not specified'}</p>
            <p><strong>Location:</strong> ${apt.location || 'Not specified'}</p>
            ${apt.notes ? `<p><strong>Notes:</strong> ${apt.notes}</p>` : ''}
        </div>
    `).join('');
}

function showAddAppointment() {
    // This would open a modal to add a new appointment
    alert('Add appointment modal would open here');
}

// Tools
function startKickCounter() {
    alert('Kick counter would start here - tracking baby movements');
}

function startContractionTimer() {
    alert('Contraction timer would start here');
}

function openPhotoJournal() {
    alert('Photo journal would open here - document your bump progression');
}

function showDueDateCalculator() {
    alert('Due date calculator would open here');
}

async function exportData(format) {
    try {
        const endpoint = format === 'pdf' ? '/api/v1/export/pdf' : '/api/v1/export/json';
        const response = await fetch(`${API_BASE}${endpoint}`, {
            headers: {
                'X-User-ID': state.userId
            }
        });
        
        if (response.ok) {
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `pregnancy-data.${format}`;
            a.click();
            window.URL.revokeObjectURL(url);
            showNotification(`Data exported as ${format.toUpperCase()}`);
        }
    } catch (error) {
        console.error('Export error:', error);
        // Fallback to local export
        if (format === 'json') {
            const dataStr = JSON.stringify(state, null, 2);
            const blob = new Blob([dataStr], { type: 'application/json' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'pregnancy-data.json';
            a.click();
            window.URL.revokeObjectURL(url);
        }
    }
}

function managePartnerAccess() {
    alert('Partner access management would open here - share limited data with your partner');
}

function showAIAssistant() {
    const question = prompt('Ask a pregnancy-related question (not medical advice):');
    if (question) {
        alert('AI would process: ' + question + '\n\nNote: This is not medical advice. Always consult your healthcare provider.');
    }
}

function showCitations() {
    alert('Scientific citations and sources for pregnancy information would be displayed here');
}

// Emergency Modal
function showEmergencyModal() {
    document.getElementById('emergencyModal').classList.remove('hidden');
    document.getElementById('modalOverlay').classList.remove('hidden');
    
    // Load emergency info
    document.getElementById('bloodType').textContent = state.emergencyInfo.bloodType || 'Not set';
    document.getElementById('allergies').textContent = 
        state.emergencyInfo.allergies?.join(', ') || 'None';
    document.getElementById('medications').textContent = 
        state.emergencyInfo.medications?.join(', ') || 'None';
    document.getElementById('obContact').textContent = 
        state.emergencyInfo.obContact || 'Not set';
    document.getElementById('hospital').textContent = 
        state.emergencyInfo.hospital || 'Not set';
    document.getElementById('emergencyContact').textContent = 
        state.emergencyInfo.emergencyContact || 'Not set';
}

function closeModal() {
    document.querySelectorAll('.modal').forEach(modal => {
        modal.classList.add('hidden');
    });
    document.getElementById('modalOverlay').classList.add('hidden');
}

// Utility Functions
function saveState() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

function loadStoredData() {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
        state = JSON.parse(stored);
    }
}

function showNotification(message) {
    // Create a simple notification
    const notification = document.createElement('div');
    notification.className = 'notification';
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        bottom: 20px;
        right: 20px;
        background: var(--success);
        color: white;
        padding: 16px 24px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.2);
        z-index: 2000;
        animation: slideIn 0.3s;
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Add animation styles
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); }
        to { transform: translateX(0); }
    }
    @keyframes slideOut {
        from { transform: translateX(0); }
        to { transform: translateX(100%); }
    }
    
    .milestone {
        padding: 16px;
        border-left: 4px solid var(--border);
        margin-left: 20px;
        position: relative;
    }
    
    .milestone::before {
        content: '';
        position: absolute;
        left: -12px;
        top: 20px;
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background: var(--bg-secondary);
        border: 4px solid var(--border);
    }
    
    .milestone.passed {
        border-left-color: var(--primary);
    }
    
    .milestone.passed::before {
        background: var(--primary);
        border-color: var(--primary);
    }
    
    .milestone.current {
        background: var(--bg-tertiary);
        border-left-color: var(--primary-dark);
    }
    
    .milestone-week {
        font-size: 0.9rem;
        color: var(--text-secondary);
        margin-bottom: 8px;
    }
    
    .milestone-content h4 {
        color: var(--primary-dark);
        margin-bottom: 4px;
    }
    
    .appointment-card {
        background: var(--bg-tertiary);
        padding: 16px;
        border-radius: 8px;
        margin-bottom: 16px;
    }
    
    .appointment-card h4 {
        color: var(--primary-dark);
        margin-bottom: 8px;
    }
    
    .search-result {
        padding: 12px;
        border-bottom: 1px solid var(--border);
        cursor: pointer;
    }
    
    .search-result:hover {
        background: var(--bg-tertiary);
    }
    
    .search-result h4 {
        color: var(--primary-dark);
        margin-bottom: 4px;
    }
`;
document.head.appendChild(style);