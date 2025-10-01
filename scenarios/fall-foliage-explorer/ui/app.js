// Fall Foliage Explorer - Interactive Application

// Configuration
const API_BASE = 'http://localhost:17175';  // Fixed API port from service.json
let map = null;
let markers = [];
let currentView = 'map';
let regionsData = [];  // Will be populated from API

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
    initializeMap();
    setupEventListeners();
    loadRegions();
    updateTimeSlider();
});

// Initialize Leaflet map
function initializeMap() {
    map = L.map('map').setView([42.3601, -71.0589], 6);
    
    // Add autumn-themed tile layer
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        opacity: 0.8
    }).addTo(map);
    
    // Add foliage overlay
    addFoliageMarkers();
}

// Add markers for foliage regions
async function addFoliageMarkers() {
    // Use real data from API if available, otherwise use empty array
    const regions = regionsData.length > 0 ? regionsData : [];

    // Fetch current foliage status for each region
    for (const region of regions) {
        try {
            const foliageResponse = await fetch(`${API_BASE}/api/foliage?region_id=${region.id}`);
            const foliageData = await foliageResponse.json();

            const status = foliageData.data?.peak_status || 'not_started';
            const intensity = foliageData.data?.color_intensity || 0;

            const color = getColorForStatus(status);
            const marker = L.circleMarker([region.latitude, region.longitude], {
                radius: 10 + intensity,
                fillColor: color,
                color: '#4a3c2a',
                weight: 2,
                opacity: 1,
                fillOpacity: 0.7
            }).addTo(map);

            marker.bindPopup(`
                <div class="popup-content">
                    <h3>${region.name}</h3>
                    <p>${region.state}</p>
                    <p><strong>Status:</strong> ${formatStatus(status)}</p>
                    <p><strong>Color Intensity:</strong> ${intensity}/10</p>
                    <button onclick="showRegionDetails(${region.id})">View Details</button>
                </div>
            `);

            // Store status and intensity on marker for later updates
            marker.regionData = { id: region.id, status, intensity };
            markers.push(marker);
        } catch (err) {
            console.error(`Failed to load foliage data for region ${region.id}:`, err);
        }
    }
}

// Get color based on foliage status
function getColorForStatus(status) {
    const colors = {
        'not_started': '#2d5a2d',
        'progressing': '#8b7355',
        'near_peak': '#ff8c00',
        'peak': '#dc143c',
        'past_peak': '#8b4513'
    };
    return colors[status] || '#8b7355';
}

// Format status for display
function formatStatus(status) {
    return status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase());
}

// Setup event listeners
function setupEventListeners() {
    // Navigation buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            switchView(e.target.dataset.view);
        });
    });
    
    // Time slider
    const slider = document.getElementById('time-slider');
    slider.addEventListener('input', updateTimeSlider);
    
    // Trip planner
    document.querySelector('.save-trip-btn')?.addEventListener('click', saveTripPlan);
}

// Switch between views
function switchView(view) {
    // Update nav buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === view);
    });
    
    // Update panels
    document.querySelectorAll('.view-panel').forEach(panel => {
        panel.classList.remove('active');
    });
    document.getElementById(`${view}-view`).classList.add('active');
    
    currentView = view;
    
    // Load view-specific content
    switch(view) {
        case 'map':
            setTimeout(() => map.invalidateSize(), 100);
            break;
        case 'timeline':
            loadTimeline();
            break;
        case 'regions':
            loadRegionsGrid();
            break;
        case 'planner':
            loadPlannerRegions();
            break;
    }
}

// Update time slider display
function updateTimeSlider() {
    const slider = document.getElementById('time-slider');
    const value = parseInt(slider.value);
    
    // Calculate date based on slider position
    const startDate = new Date(2025, 8, 1); // September 1
    const currentDate = new Date(startDate);
    currentDate.setDate(startDate.getDate() + value);
    
    const dateDisplay = document.getElementById('selected-date');
    dateDisplay.textContent = currentDate.toLocaleDateString('en-US', { 
        month: 'long', 
        day: 'numeric', 
        year: 'numeric' 
    });
    
    // Update map markers based on date
    updateMarkersForDate(currentDate);
}

// Update markers based on selected date
function updateMarkersForDate(date) {
    // Simulate changing foliage based on date
    const dayOfYear = Math.floor((date - new Date(date.getFullYear(), 0, 0)) / 86400000);

    markers.forEach((marker) => {
        if (!marker.regionData) return;

        // Use typical peak week from region data if available
        const peakDay = 270 + (marker.regionData.id * 5); // Stagger based on region ID
        const dayDiff = Math.abs(dayOfYear - peakDay);

        let status, intensity;
        if (dayDiff > 30) {
            status = 'not_started';
            intensity = 1;
        } else if (dayDiff > 20) {
            status = 'progressing';
            intensity = 4;
        } else if (dayDiff > 10) {
            status = 'near_peak';
            intensity = 7;
        } else if (dayDiff <= 10) {
            status = 'peak';
            intensity = 10;
        }

        const color = getColorForStatus(status);
        marker.setStyle({ fillColor: color });
        marker.setRadius(10 + intensity);
    });
}

// Load timeline view
function loadTimeline() {
    const grid = document.getElementById('timeline-grid');
    grid.innerHTML = '';
    
    const weeks = ['Sep Week 1', 'Sep Week 2', 'Sep Week 3', 'Sep Week 4',
                   'Oct Week 1', 'Oct Week 2', 'Oct Week 3', 'Oct Week 4'];
    
    weeks.forEach((week, index) => {
        const card = document.createElement('div');
        card.className = 'timeline-card';
        card.innerHTML = `
            <h4>${week}</h4>
            <div style="margin-top: 1rem;">
                ${getRegionsForWeek(index).map(r => `
                    <div style="padding: 0.25rem; margin: 0.25rem 0; background: ${getColorForStatus(r.status)}20; border-radius: 5px;">
                        ${r.name}
                    </div>
                `).join('')}
            </div>
        `;
        grid.appendChild(card);
    });
}

// Get regions for a specific week
function getRegionsForWeek(weekIndex) {
    // Use real regions data if available
    if (regionsData.length === 0) return [];

    // Simulate which regions peak in which weeks based on typical_peak_week
    return regionsData.filter(region => {
        const peakWeek = region.typical_peak_week || 40;
        const weekNumber = 35 + weekIndex; // Sep Week 1 = week 35
        return Math.abs(peakWeek - weekNumber) <= 1;
    });
}

// Load regions grid
async function loadRegionsGrid() {
    const grid = document.getElementById('regions-grid');
    grid.innerHTML = '';

    for (const region of regionsData) {
        try {
            // Fetch current foliage status
            const foliageResponse = await fetch(`${API_BASE}/api/foliage?region_id=${region.id}`);
            const foliageData = await foliageResponse.json();

            const status = foliageData.data?.peak_status || 'not_started';
            const intensity = foliageData.data?.color_intensity || 0;

            const card = document.createElement('div');
            card.className = 'region-card';
            card.innerHTML = `
                <div class="region-image"></div>
                <div class="region-info">
                    <div class="region-name">${region.name}</div>
                    <div class="region-state">${region.state}</div>
                    <span class="region-status status-${status}">${formatStatus(status)}</span>
                    <div style="margin-top: 1rem;">
                        <div>Color Intensity: ${intensity}/10</div>
                    </div>
                </div>
            `;
            card.addEventListener('click', () => showRegionDetails(region.id));
            grid.appendChild(card);
        } catch (err) {
            console.error(`Failed to load region ${region.id}:`, err);
        }
    }
}

// Load regions
async function loadRegions() {
    try {
        const response = await fetch(`${API_BASE}/api/regions`);
        const data = await response.json();

        if (data && data.regions) {
            regionsData = data.regions;
            console.log('Regions loaded from API:', regionsData);

            // Update the map with real data
            if (map) {
                clearMarkers();
                addFoliageMarkers();
            }
        }
    } catch (err) {
        console.error('Failed to load regions from API:', err);
        // Fallback to sample data for demo
        regionsData = [
            { id: 1, name: 'White Mountains', state: 'New Hampshire', latitude: 44.2700, longitude: -71.3034, current_status: 'near_peak', color_intensity: 7 },
            { id: 2, name: 'Green Mountains', state: 'Vermont', latitude: 43.9207, longitude: -72.8986, current_status: 'peak', color_intensity: 9 },
            { id: 3, name: 'Adirondacks', state: 'New York', latitude: 44.1127, longitude: -74.0524, current_status: 'progressing', color_intensity: 5 },
            { id: 4, name: 'Great Smoky Mountains', state: 'Tennessee', latitude: 35.6532, longitude: -83.5070, current_status: 'near_peak', color_intensity: 8 },
            { id: 5, name: 'Blue Ridge Parkway', state: 'Virginia', latitude: 37.5615, longitude: -79.3553, current_status: 'peak', color_intensity: 10 },
            { id: 6, name: 'Berkshires', state: 'Massachusetts', latitude: 42.3604, longitude: -73.2290, current_status: 'progressing', color_intensity: 6 },
        ];
    }
}

// Clear existing markers
function clearMarkers() {
    markers.forEach(marker => map.removeLayer(marker));
    markers = [];
}

// Load planner regions
function loadPlannerRegions() {
    const selector = document.getElementById('region-selector');
    selector.innerHTML = '';

    regionsData.forEach(region => {
        const checkbox = document.createElement('div');
        checkbox.className = 'region-checkbox';
        checkbox.innerHTML = `
            <label>
                <input type="checkbox" value="${region.id}">
                ${region.name}, ${region.state}
            </label>
        `;
        selector.appendChild(checkbox);
    });
}

// Save trip plan
function saveTripPlan() {
    const name = document.getElementById('trip-name').value;
    const startDate = document.getElementById('start-date').value;
    const endDate = document.getElementById('end-date').value;
    
    const selectedRegions = Array.from(document.querySelectorAll('#region-selector input:checked'))
        .map(cb => cb.value);
    
    const trip = {
        name,
        startDate,
        endDate,
        regions: selectedRegions,
        timestamp: new Date().toISOString()
    };
    
    // Save to localStorage
    const trips = JSON.parse(localStorage.getItem('foliageTrips') || '[]');
    trips.push(trip);
    localStorage.setItem('foliageTrips', JSON.stringify(trips));
    
    alert('Trip saved successfully!');
    displaySavedTrips();
}

// Display saved trips
function displaySavedTrips() {
    const trips = JSON.parse(localStorage.getItem('foliageTrips') || '[]');
    const list = document.getElementById('saved-trips-list');
    
    list.innerHTML = trips.map(trip => `
        <div style="background: white; padding: 1rem; margin: 0.5rem 0; border-radius: 8px;">
            <strong>${trip.name}</strong><br>
            ${trip.startDate} - ${trip.endDate}<br>
            ${trip.regions.length} regions selected
        </div>
    `).join('');
}

// Show region details
async function showRegionDetails(regionId) {
    const region = regionsData.find(r => r.id === regionId);
    if (!region) return;

    const panel = document.getElementById('info-panel');
    const content = document.getElementById('info-content');

    try {
        // Fetch current foliage status
        const foliageResponse = await fetch(`${API_BASE}/api/foliage?region_id=${regionId}`);
        const foliageData = await foliageResponse.json();

        const status = foliageData.data?.peak_status || 'not_started';
        const intensity = foliageData.data?.color_intensity || 0;

        content.innerHTML = `
            <div style="margin-top: 1rem;">
                <p><strong>Location:</strong> ${region.name}, ${region.state}</p>
                <p><strong>Current Status:</strong> ${formatStatus(status)}</p>
                <p><strong>Color Intensity:</strong> ${intensity}/10</p>
                <p><strong>Coordinates:</strong> ${region.latitude.toFixed(4)}, ${region.longitude.toFixed(4)}</p>
                <p><strong>Elevation:</strong> ${region.elevation_meters || 'N/A'} meters</p>

                <div style="margin-top: 1.5rem;">
                    <h4>Forecast</h4>
                    <p>Peak Expected: Week ${region.typical_peak_week || 40} of year</p>
                    <p>Confidence: 85%</p>
                </div>

                <div style="margin-top: 1.5rem;">
                    <h4>Weather Conditions</h4>
                    <p>Temperature: 52°F / 11°C</p>
                    <p>Recent Rainfall: Moderate</p>
                    <p>Night Temps: Optimal for color</p>
                </div>
            </div>
        `;

        document.getElementById('info-title').textContent = region.name;
        panel.classList.add('open');
    } catch (err) {
        console.error(`Failed to show region details for ${regionId}:`, err);
    }
}

// Close info panel
function closeInfoPanel() {
    document.getElementById('info-panel').classList.remove('open');
}