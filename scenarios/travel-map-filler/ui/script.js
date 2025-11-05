import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const BRIDGE_FLAG = '__travelMapFillerBridgeInitialized'
const LOOPBACK_HOST = '127.0.0.1'
const PROXY_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__']
const DEFAULT_API_BASE_PATH = '/api'

function ensureBridge() {
    if (typeof window === 'undefined') {
        return
    }

    if (window.parent === window) {
        return
    }

    if (window[BRIDGE_FLAG]) {
        return
    }

    try {
        initIframeBridgeChild({
            appId: 'travel-map-filler-ui',
            captureLogs: { enabled: true },
            captureNetwork: { enabled: true }
        })
        window[BRIDGE_FLAG] = true
    } catch (error) {
        console.warn('[Travel Map Filler] Failed to initialize iframe bridge', error)
    }
}

function coerceString(value) {
    if (typeof value === 'string') {
        const trimmed = value.trim()
        return trimmed.length > 0 ? trimmed : undefined
    }
    return undefined
}

function stripTrailingSlash(value) {
    if (!value) {
        return ''
    }
    const normalized = value.replace(/\/+$/, '')
    return normalized || '/'
}

function ensureLeadingSlash(value) {
    if (!value) {
        return '/'
    }
    return value.startsWith('/') ? value : `/${value}`
}

function sanitizePort(port) {
    if (typeof port === 'number') {
        return port > 0 ? String(port) : ''
    }

    if (typeof port === 'string') {
        const trimmed = port.trim()
        return trimmed ? trimmed : ''
    }

    return ''
}

function collectProxyCandidates(value, seen, list) {
    if (!value) {
        return
    }

    if (Array.isArray(value)) {
        value.forEach((item) => collectProxyCandidates(item, seen, list))
        return
    }

    if (typeof value === 'string') {
        const candidate = value.trim()
        if (candidate && !seen.has(candidate)) {
            seen.add(candidate)
            list.push(candidate)
        }
        return
    }

    if (typeof value === 'object') {
        const record = value
        collectProxyCandidates(record.url, seen, list)
        collectProxyCandidates(record.path, seen, list)
        collectProxyCandidates(record.target, seen, list)
        collectProxyCandidates(record.primary, seen, list)
        if (Array.isArray(record.endpoints)) {
            record.endpoints.forEach((endpoint) => collectProxyCandidates(endpoint, seen, list))
        }
    }
}

function resolveProxyBase() {
    if (typeof window === 'undefined') {
        return undefined
    }

    const seen = new Set()
    const candidates = []

    PROXY_KEYS.forEach((key) => {
        try {
            collectProxyCandidates(window[key], seen, candidates)
        } catch (error) {
            console.warn(`[Travel Map Filler] Unable to process proxy key ${key}`, error)
        }
    })

    const candidate = candidates.find(Boolean)
    if (!candidate) {
        return undefined
    }

    if (/^https?:\/\//i.test(candidate)) {
        return stripTrailingSlash(candidate)
    }

    const origin = coerceString(window.location && window.location.origin)
    if (!origin) {
        return undefined
    }

    return stripTrailingSlash(origin + ensureLeadingSlash(candidate))
}

function resolveApiBase(basePath = DEFAULT_API_BASE_PATH) {
    if (typeof window === 'undefined') {
        return basePath
    }

    const proxyBase = resolveProxyBase()
    if (proxyBase) {
        return stripTrailingSlash(proxyBase) + ensureLeadingSlash(basePath)
    }

    const origin = coerceString(window.location && window.location.origin)
    if (origin) {
        return stripTrailingSlash(origin) + ensureLeadingSlash(basePath)
    }

    const protocol = (window.location && window.location.protocol) || 'http:'
    const hostname = coerceString(window.location && window.location.hostname) || LOOPBACK_HOST
    const portSegment = sanitizePort(window.location && window.location.port)
    const port = portSegment ? `:${portSegment}` : ''

    return `${protocol}//${hostname}${port}` + ensureLeadingSlash(basePath)
}

function buildApiUrl(path = '', basePath = DEFAULT_API_BASE_PATH) {
    const base = resolveApiBase(basePath)
    if (!path) {
        return base
    }

    if (/^https?:\/\//i.test(path)) {
        return path
    }

    return stripTrailingSlash(base) + ensureLeadingSlash(path)
}

ensureBridge()

// Travel Map Filler - Interactive Script

class TravelMapFiller {
    constructor() {
        this.map = null;
        this.markers = [];
        this.travels = [];
        this.achievements = [];
        this.stats = {
            countries: 0,
            cities: 0,
            continents: 0,
            coverage: 0
        };
        this.visitedCountries = new Set();
        this.visitedContinents = new Set();
        this.heatmapEnabled = false;
        
        this.init();
    }

    async init() {
        this.initMap();
        this.setupEventListeners();
        await this.loadTravels();
        this.updateStats();
        this.checkAchievements();
        this.loadRecentTravels();
    }

    initMap() {
        // Initialize Leaflet map
        this.map = L.map('worldMap').setView([20, 0], 2);
        
        // Add tile layer with a nice map style
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors',
            maxZoom: 18,
            minZoom: 2
        }).addTo(this.map);

        // Custom marker icon
        this.markerIcon = L.divIcon({
            className: 'custom-marker',
            html: '<div style="background: linear-gradient(135deg, #FF6B35, #F7931E); width: 12px; height: 12px; border-radius: 50%; border: 2px solid white; box-shadow: 0 2px 5px rgba(0,0,0,0.3);"></div>',
            iconSize: [16, 16],
            iconAnchor: [8, 8]
        });

        // Visited country icon
        this.visitedIcon = L.divIcon({
            className: 'visited-marker',
            html: '<div style="background: #52B788; width: 20px; height: 20px; border-radius: 50%; border: 3px solid white; box-shadow: 0 2px 8px rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; color: white; font-weight: bold;">✓</div>',
            iconSize: [26, 26],
            iconAnchor: [13, 13]
        });
    }

    setupEventListeners() {
        // Map controls
        document.getElementById('zoomInBtn').addEventListener('click', () => {
            this.map.zoomIn();
        });

        document.getElementById('zoomOutBtn').addEventListener('click', () => {
            this.map.zoomOut();
        });

        document.getElementById('resetViewBtn').addEventListener('click', () => {
            this.map.setView([20, 0], 2);
        });

        document.getElementById('heatmapToggle').addEventListener('click', () => {
            this.toggleHeatmap();
        });

        // Search
        document.getElementById('searchBtn').addEventListener('click', () => {
            this.searchLocation();
        });

        document.getElementById('locationSearch').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.searchLocation();
            }
        });

        // Add travel form
        document.getElementById('addTravelForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            await this.addTravel();
        });

        // Footer buttons
        document.getElementById('exportBtn').addEventListener('click', () => {
            this.exportMap();
        });

        document.getElementById('shareBtn').addEventListener('click', () => {
            this.shareMap();
        });

        document.getElementById('statsBtn').addEventListener('click', () => {
            this.showDetailedStats();
        });

        document.getElementById('bucketListBtn').addEventListener('click', () => {
            this.showBucketList();
        });

        // Modal close
        document.getElementById('modalClose').addEventListener('click', () => {
            this.closeModal();
        });

        // Click outside modal to close
        document.getElementById('travelModal').addEventListener('click', (e) => {
            if (e.target.id === 'travelModal') {
                this.closeModal();
            }
        });
    }

    async loadTravels() {
        try {
            const response = await fetch(buildApiUrl('/travels'));
            if (response.ok) {
                this.travels = await response.json();
                this.displayTravels();
            }
        } catch (error) {
            console.error('Error loading travels:', error);
            // Use sample data for demo
            this.travels = this.getSampleTravels();
            this.displayTravels();
        }
    }

    getSampleTravels() {
        return [
            { id: 1, location: 'Paris, France', lat: 48.8566, lng: 2.3522, date: '2023-06-15', type: 'vacation', notes: 'Amazing trip to the City of Light!' },
            { id: 2, location: 'Tokyo, Japan', lat: 35.6762, lng: 139.6503, date: '2023-09-20', type: 'adventure', notes: 'Incredible culture and food' },
            { id: 3, location: 'New York, USA', lat: 40.7128, lng: -74.0060, date: '2023-03-10', type: 'business', notes: 'Conference at Times Square' },
            { id: 4, location: 'Sydney, Australia', lat: -33.8688, lng: 151.2093, date: '2024-01-05', type: 'vacation', notes: 'New Year at the Opera House' },
            { id: 5, location: 'Cairo, Egypt', lat: 30.0444, lng: 31.2357, date: '2023-11-12', type: 'adventure', notes: 'Pyramids and ancient history' }
        ];
    }

    displayTravels() {
        // Clear existing markers
        this.markers.forEach(marker => marker.remove());
        this.markers = [];

        // Add markers for each travel
        this.travels.forEach(travel => {
            const marker = L.marker([travel.lat, travel.lng], { icon: this.markerIcon })
                .addTo(this.map)
                .bindPopup(this.createPopupContent(travel));
            
            marker.on('click', () => {
                this.showTravelDetails(travel);
            });

            this.markers.push(marker);

            // Track countries and continents
            this.visitedCountries.add(travel.location.split(',')[1]?.trim() || travel.location);
            this.visitedContinents.add(this.getContinent(travel.lat, travel.lng));
        });

        this.updateStats();
    }

    createPopupContent(travel) {
        const typeEmoji = {
            vacation: '🏖️',
            business: '💼',
            adventure: '🏔️',
            family: '👨‍👩‍👧‍👦',
            transit: '✈️',
            lived: '🏠'
        };

        return `
            <div style="min-width: 200px;">
                <h3>${travel.location}</h3>
                <p>${typeEmoji[travel.type] || '📍'} ${travel.type}</p>
                <p>📅 ${new Date(travel.date).toLocaleDateString()}</p>
                ${travel.notes ? `<p style="font-style: italic;">"${travel.notes}"</p>` : ''}
            </div>
        `;
    }

    getContinent(lat, lng) {
        // Simplified continent detection based on coordinates
        if (lat > 35 && lng > -25 && lng < 65) return 'Europe';
        if (lat > 10 && lng > 60 && lng < 150) return 'Asia';
        if (lat < 10 && lat > -35 && lng > -20 && lng < 55) return 'Africa';
        if (lat > 15 && lng > -170 && lng < -30) return 'North America';
        if (lat < 15 && lng > -85 && lng < -30) return 'South America';
        if (lat < -10 && lng > 110) return 'Oceania';
        return 'Unknown';
    }

    updateStats() {
        this.stats.countries = this.visitedCountries.size;
        this.stats.cities = this.travels.length;
        this.stats.continents = this.visitedContinents.size;
        this.stats.coverage = Math.round((this.stats.countries / 195) * 100);

        document.getElementById('countriesCount').textContent = this.stats.countries;
        document.getElementById('citiesCount').textContent = this.stats.cities;
        document.getElementById('continentsCount').textContent = this.stats.continents;
        document.getElementById('coveragePercent').textContent = `${this.stats.coverage}%`;
    }

    checkAchievements() {
        const achievementsList = document.getElementById('achievementsList');
        const achievements = [
            { icon: '🌍', name: 'First Trip', unlocked: this.travels.length >= 1 },
            { icon: '🗺️', name: '5 Countries', unlocked: this.stats.countries >= 5 },
            { icon: '✈️', name: '10 Cities', unlocked: this.stats.cities >= 10 },
            { icon: '🌎', name: '3 Continents', unlocked: this.stats.continents >= 3 },
            { icon: '🏆', name: '25% World', unlocked: this.stats.coverage >= 25 },
            { icon: '👑', name: 'World Explorer', unlocked: this.stats.coverage >= 50 },
            { icon: '🚀', name: 'Adventurer', unlocked: this.travels.filter(t => t.type === 'adventure').length >= 5 },
            { icon: '💼', name: 'Business Pro', unlocked: this.travels.filter(t => t.type === 'business').length >= 3 },
            { icon: '🌟', name: 'Globetrotter', unlocked: this.stats.continents >= 6 }
        ];

        achievementsList.innerHTML = achievements.map(achievement => `
            <div class="achievement ${achievement.unlocked ? 'unlocked' : ''}" title="${achievement.name}">
                <div class="achievement-icon">${achievement.icon}</div>
                <div class="achievement-name">${achievement.name}</div>
            </div>
        `).join('');
    }

    loadRecentTravels() {
        const recentList = document.getElementById('recentList');
        const recent = this.travels
            .sort((a, b) => new Date(b.date) - new Date(a.date))
            .slice(0, 5);

        recentList.innerHTML = recent.map(travel => `
            <div class="recent-item" onclick="travelMap.showTravelDetails(${travel.id})">
                <div class="recent-icon">📍</div>
                <div class="recent-details">
                    <div class="recent-location">${travel.location}</div>
                    <div class="recent-date">${new Date(travel.date).toLocaleDateString()}</div>
                </div>
            </div>
        `).join('');
    }

    async searchLocation() {
        const query = document.getElementById('locationSearch').value;
        if (!query) return;

        try {
            // Use Nominatim API for geocoding
            const response = await fetch(`https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(query)}`);
            const results = await response.json();

            if (results.length > 0) {
                const result = results[0];
                const lat = parseFloat(result.lat);
                const lng = parseFloat(result.lon);
                
                this.map.setView([lat, lng], 10);
                
                // Add temporary marker
                const tempMarker = L.marker([lat, lng])
                    .addTo(this.map)
                    .bindPopup(`<b>${result.display_name}</b><br>Click "Add New Travel" to save this location`)
                    .openPopup();
                
                setTimeout(() => tempMarker.remove(), 5000);
            } else {
                alert('Location not found. Please try a different search.');
            }
        } catch (error) {
            console.error('Search error:', error);
            alert('Error searching for location');
        }
    }

    async addTravel() {
        const location = document.getElementById('locationInput').value;
        const date = document.getElementById('dateInput').value;
        const type = document.getElementById('typeSelect').value;
        const notes = document.getElementById('notesInput').value;

        try {
            // Geocode the location
            const geoResponse = await fetch(`https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(location)}`);
            const geoResults = await geoResponse.json();

            if (geoResults.length === 0) {
                alert('Location not found. Please enter a valid city or country.');
                return;
            }

            const lat = parseFloat(geoResults[0].lat);
            const lng = parseFloat(geoResults[0].lon);

            const newTravel = {
                id: Date.now(),
                location,
                lat,
                lng,
                date,
                type,
                notes
            };

            // Try to save to API
            try {
                const response = await fetch(buildApiUrl('/travels'), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(newTravel)
                });

                if (!response.ok) throw new Error('API error');
            } catch (error) {
                console.log('Using local storage as fallback');
            }

            // Add to local array
            this.travels.push(newTravel);
            this.displayTravels();
            this.loadRecentTravels();
            this.checkAchievements();

            // Clear form
            document.getElementById('addTravelForm').reset();

            // Pan to new location
            this.map.setView([lat, lng], 8);

            // Show success message
            this.showNotification('Travel added successfully!');
        } catch (error) {
            console.error('Error adding travel:', error);
            alert('Error adding travel. Please try again.');
        }
    }

    toggleHeatmap() {
        this.heatmapEnabled = !this.heatmapEnabled;
        
        if (this.heatmapEnabled) {
            // Add heatmap layer (simplified version)
            this.markers.forEach(marker => {
                const pos = marker.getLatLng();
                L.circle([pos.lat, pos.lng], {
                    color: 'red',
                    fillColor: '#ff6b35',
                    fillOpacity: 0.3,
                    radius: 500000
                }).addTo(this.map);
            });
        } else {
            // Remove heatmap circles
            this.map.eachLayer(layer => {
                if (layer instanceof L.Circle) {
                    this.map.removeLayer(layer);
                }
            });
        }
    }

    showTravelDetails(travelId) {
        const travel = this.travels.find(t => t.id === travelId);
        if (!travel) return;

        document.getElementById('modalTitle').textContent = travel.location;
        document.getElementById('modalBody').innerHTML = `
            <p><strong>Date:</strong> ${new Date(travel.date).toLocaleDateString()}</p>
            <p><strong>Type:</strong> ${travel.type}</p>
            ${travel.notes ? `<p><strong>Notes:</strong> ${travel.notes}</p>` : ''}
            <div style="margin-top: 1rem;">
                <button class="submit-btn" onclick="travelMap.editTravel(${travel.id})">Edit</button>
                <button class="submit-btn" style="background: #E63946;" onclick="travelMap.deleteTravel(${travel.id})">Delete</button>
            </div>
        `;
        
        document.getElementById('travelModal').classList.remove('hidden');
    }

    closeModal() {
        document.getElementById('travelModal').classList.add('hidden');
    }

    async deleteTravel(id) {
        if (!confirm('Are you sure you want to delete this travel?')) return;

        try {
            await fetch(buildApiUrl(`/travels/${id}`), { method: 'DELETE' });
        } catch (error) {
            console.log('Using local deletion');
        }

        this.travels = this.travels.filter(t => t.id !== id);
        this.displayTravels();
        this.loadRecentTravels();
        this.checkAchievements();
        this.closeModal();
        this.showNotification('Travel deleted');
    }

    exportMap() {
        const data = {
            travels: this.travels,
            stats: this.stats,
            exportDate: new Date().toISOString()
        };

        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `travel-map-${new Date().toISOString().split('T')[0]}.json`;
        a.click();
        URL.revokeObjectURL(url);

        this.showNotification('Map exported successfully!');
    }

    shareMap() {
        const shareUrl = `${window.location.origin}/share/${btoa(JSON.stringify(this.travels))}`;
        
        if (navigator.share) {
            navigator.share({
                title: 'My Travel Map',
                text: `I've visited ${this.stats.countries} countries and ${this.stats.cities} cities!`,
                url: shareUrl
            });
        } else {
            navigator.clipboard.writeText(shareUrl);
            this.showNotification('Share link copied to clipboard!');
        }
    }

    showDetailedStats() {
        const statsContent = `
            <h3>Detailed Statistics</h3>
            <p><strong>Total Trips:</strong> ${this.travels.length}</p>
            <p><strong>Countries Visited:</strong> ${this.stats.countries}</p>
            <p><strong>Cities Visited:</strong> ${this.stats.cities}</p>
            <p><strong>Continents:</strong> ${this.stats.continents}/7</p>
            <p><strong>World Coverage:</strong> ${this.stats.coverage}%</p>
            <hr>
            <h4>Travel Types</h4>
            <p>Vacation: ${this.travels.filter(t => t.type === 'vacation').length}</p>
            <p>Business: ${this.travels.filter(t => t.type === 'business').length}</p>
            <p>Adventure: ${this.travels.filter(t => t.type === 'adventure').length}</p>
            <p>Family: ${this.travels.filter(t => t.type === 'family').length}</p>
            <hr>
            <h4>Most Recent Trip</h4>
            <p>${this.travels[0]?.location || 'None yet'}</p>
        `;

        document.getElementById('modalTitle').textContent = '📊 Travel Statistics';
        document.getElementById('modalBody').innerHTML = statsContent;
        document.getElementById('travelModal').classList.remove('hidden');
    }

    showBucketList() {
        const bucketListContent = `
            <h3>Travel Bucket List</h3>
            <p>Plan your future adventures!</p>
            <textarea id="bucketListInput" class="form-textarea" rows="10" placeholder="Enter destinations you want to visit..."></textarea>
            <button class="submit-btn" onclick="travelMap.saveBucketList()">Save Bucket List</button>
        `;

        document.getElementById('modalTitle').textContent = '📝 My Bucket List';
        document.getElementById('modalBody').innerHTML = bucketListContent;
        document.getElementById('travelModal').classList.remove('hidden');

        // Load existing bucket list
        const savedList = localStorage.getItem('travelBucketList');
        if (savedList) {
            document.getElementById('bucketListInput').value = savedList;
        }
    }

    saveBucketList() {
        const bucketList = document.getElementById('bucketListInput').value;
        localStorage.setItem('travelBucketList', bucketList);
        this.showNotification('Bucket list saved!');
        this.closeModal();
    }

    showNotification(message) {
        const notification = document.createElement('div');
        notification.className = 'notification';
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: linear-gradient(135deg, #52B788, #40916c);
            color: white;
            padding: 1rem 2rem;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.2);
            z-index: 10000;
            animation: slideIn 0.3s ease;
        `;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => notification.remove(), 300);
        }, 3000);
    }
}

// Initialize the app
let travelMap;
document.addEventListener('DOMContentLoaded', () => {
    travelMap = new TravelMapFiller();
});

// Add animation styles
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);
