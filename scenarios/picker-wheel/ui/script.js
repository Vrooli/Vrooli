// Picker Wheel - Interactive Script

const API_BASE = 'http://localhost:8100/api';
const N8N_BASE = 'http://localhost:5678/webhook';

class PickerWheel {
    constructor() {
        this.currentWheel = null;
        this.isSpinning = false;
        this.history = [];
        this.savedWheels = [];
        this.customOptions = [];
        this.settings = {
            sound: true,
            confetti: true,
            speed: 'normal',
            theme: 'neon'
        };
        
        this.init();
    }

    init() {
        this.loadSettings();
        this.setupEventListeners();
        this.loadPresetWheel('dinner');
        this.loadHistory();
        this.loadSavedWheels();
        this.generateSessionId();
    }

    generateSessionId() {
        this.sessionId = 'session_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }

    setupEventListeners() {
        // Spin button
        document.getElementById('spinButton').addEventListener('click', () => this.spin());

        // Tab navigation
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => this.switchTab(e.target.dataset.tab));
        });

        // Preset wheels
        document.querySelectorAll('.preset-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const wheelType = e.currentTarget.dataset.wheel;
                this.loadPresetWheel(wheelType);
            });
        });

        // Custom wheel controls
        document.getElementById('addOptionBtn').addEventListener('click', () => this.addCustomOption());
        document.getElementById('clearAllBtn').addEventListener('click', () => this.clearCustomOptions());
        document.getElementById('saveWheelBtn').addEventListener('click', () => this.saveCustomWheel());

        // Settings
        document.getElementById('settingsToggle').addEventListener('click', () => this.toggleSettings());
        document.getElementById('soundToggle').addEventListener('change', (e) => {
            this.settings.sound = e.target.checked;
            this.saveSettings();
        });
        document.getElementById('confettiToggle').addEventListener('change', (e) => {
            this.settings.confetti = e.target.checked;
            this.saveSettings();
        });
        document.getElementById('speedSelect').addEventListener('change', (e) => {
            this.settings.speed = e.target.value;
            this.saveSettings();
        });
        document.getElementById('themeSelect').addEventListener('change', (e) => {
            this.settings.theme = e.target.value;
            this.applyTheme();
            this.saveSettings();
        });
    }

    switchTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });

        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(tabName + 'Tab').classList.add('active');
    }

    loadPresetWheel(wheelType) {
        // Define preset wheel configurations
        const presets = {
            'dinner': {
                name: 'Dinner Decider',
                options: [
                    { label: 'Pizza 🍕', color: '#FF6B6B', weight: 1 },
                    { label: 'Sushi 🍱', color: '#4ECDC4', weight: 1 },
                    { label: 'Tacos 🌮', color: '#FFD93D', weight: 1 },
                    { label: 'Burger 🍔', color: '#6C5CE7', weight: 1 },
                    { label: 'Pasta 🍝', color: '#A8E6CF', weight: 1 },
                    { label: 'Salad 🥗', color: '#95E77E', weight: 0.5 },
                    { label: 'Surprise! 🎲', color: '#FF1744', weight: 0.5 }
                ]
            },
            'yes-no': {
                name: 'Yes or No',
                options: [
                    { label: 'YES! ✅', color: '#4CAF50', weight: 1 },
                    { label: 'NO ❌', color: '#F44336', weight: 1 },
                    { label: 'Maybe? 🤔', color: '#FFC107', weight: 0.2 }
                ]
            },
            'teams': {
                name: 'Team Picker',
                options: [
                    { label: 'Team Alpha', color: '#E91E63', weight: 1 },
                    { label: 'Team Beta', color: '#9C27B0', weight: 1 },
                    { label: 'Team Gamma', color: '#3F51B5', weight: 1 },
                    { label: 'Team Delta', color: '#2196F3', weight: 1 }
                ]
            },
            'd20': {
                name: 'D20 Roll',
                options: Array.from({length: 20}, (_, i) => ({
                    label: (i + 1).toString() + (i === 19 ? ' 🎯' : ''),
                    color: `hsl(${i * 18}, 70%, 50%)`,
                    weight: 1
                }))
            },
            'workout': {
                name: 'Workout Chooser',
                options: [
                    { label: 'Cardio 🏃', color: '#FF5722', weight: 1 },
                    { label: 'Weights 🏋️', color: '#795548', weight: 1 },
                    { label: 'Yoga 🧘', color: '#9C27B0', weight: 1 },
                    { label: 'Swimming 🏊', color: '#03A9F4', weight: 1 },
                    { label: 'Rest Day 😴', color: '#607D8B', weight: 0.3 }
                ]
            },
            'movie': {
                name: 'Movie Genre',
                options: [
                    { label: 'Action 💥', color: '#F44336', weight: 1 },
                    { label: 'Comedy 😂', color: '#FFC107', weight: 1 },
                    { label: 'Drama 🎭', color: '#9C27B0', weight: 1 },
                    { label: 'Horror 👻', color: '#212121', weight: 1 },
                    { label: 'Sci-Fi 🚀', color: '#3F51B5', weight: 1 },
                    { label: 'Romance 💕', color: '#E91E63', weight: 1 }
                ]
            }
        };

        const wheel = presets[wheelType] || presets['dinner'];
        this.currentWheel = wheel;
        this.drawWheel(wheel.options);
    }

    drawWheel(options) {
        const svg = document.getElementById('wheel');
        svg.innerHTML = ''; // Clear existing wheel

        const centerX = 250;
        const centerY = 250;
        const radius = 230;

        // Calculate angles
        const totalWeight = options.reduce((sum, opt) => sum + (opt.weight || 1), 0);
        let currentAngle = -90; // Start from top

        options.forEach((option, index) => {
            const weight = option.weight || 1;
            const angle = (weight / totalWeight) * 360;
            const endAngle = currentAngle + angle;

            // Create path for segment
            const path = this.createSegmentPath(centerX, centerY, radius, currentAngle, endAngle);
            
            const segment = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            segment.setAttribute('d', path);
            segment.setAttribute('fill', option.color || '#333');
            segment.setAttribute('stroke', '#fff');
            segment.setAttribute('stroke-width', '2');
            segment.classList.add('wheel-segment');
            svg.appendChild(segment);

            // Add text label
            const textAngle = currentAngle + angle / 2;
            const textRadius = radius * 0.7;
            const textX = centerX + textRadius * Math.cos(textAngle * Math.PI / 180);
            const textY = centerY + textRadius * Math.sin(textAngle * Math.PI / 180);

            const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
            text.setAttribute('x', textX);
            text.setAttribute('y', textY);
            text.setAttribute('text-anchor', 'middle');
            text.setAttribute('dominant-baseline', 'middle');
            text.setAttribute('fill', '#fff');
            text.setAttribute('font-size', '14');
            text.setAttribute('font-weight', 'bold');
            text.setAttribute('transform', `rotate(${textAngle + 90}, ${textX}, ${textY})`);
            text.textContent = option.label;
            svg.appendChild(text);

            currentAngle = endAngle;
        });
    }

    createSegmentPath(cx, cy, radius, startAngle, endAngle) {
        const start = this.polarToCartesian(cx, cy, radius, endAngle);
        const end = this.polarToCartesian(cx, cy, radius, startAngle);
        const largeArcFlag = endAngle - startAngle <= 180 ? '0' : '1';

        return [
            'M', cx, cy,
            'L', start.x, start.y,
            'A', radius, radius, 0, largeArcFlag, 0, end.x, end.y,
            'Z'
        ].join(' ');
    }

    polarToCartesian(centerX, centerY, radius, angleInDegrees) {
        const angleInRadians = (angleInDegrees - 90) * Math.PI / 180.0;
        return {
            x: centerX + (radius * Math.cos(angleInRadians)),
            y: centerY + (radius * Math.sin(angleInRadians))
        };
    }

    async spin() {
        if (this.isSpinning) return;
        
        const spinButton = document.getElementById('spinButton');
        const wheel = document.getElementById('wheel');
        
        this.isSpinning = true;
        spinButton.disabled = true;
        spinButton.querySelector('.spin-text').textContent = 'SPINNING...';
        
        // Play spin sound
        if (this.settings.sound) {
            this.playSound('spin');
        }

        // Hide previous result
        document.getElementById('resultDisplay').classList.add('hidden');

        // Calculate spin parameters
        const speedMultiplier = this.settings.speed === 'fast' ? 1.5 : this.settings.speed === 'slow' ? 0.5 : 1;
        const duration = (3000 + Math.random() * 2000) / speedMultiplier;
        const rotations = 5 + Math.floor(Math.random() * 5);
        const finalAngle = Math.random() * 360;
        const totalRotation = rotations * 360 + finalAngle;

        // Apply rotation animation
        wheel.style.transition = `transform ${duration}ms cubic-bezier(0.25, 0.46, 0.45, 0.94)`;
        wheel.style.transform = `rotate(${totalRotation}deg)`;

        // Call API to get result
        try {
            const response = await fetch(`${N8N_BASE}/wheel/spin`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wheel_id: this.currentWheel.name.toLowerCase().replace(/\s+/g, '-'),
                    session_id: this.sessionId,
                    options: this.currentWheel.options
                })
            });

            const result = await response.json();
            
            // Wait for animation to complete
            setTimeout(() => {
                this.handleSpinResult(result);
            }, duration);

        } catch (error) {
            console.error('Spin error:', error);
            // Fallback to local random selection
            const localResult = this.localRandomSelection();
            setTimeout(() => {
                this.handleSpinResult(localResult);
            }, duration);
        }
    }

    localRandomSelection() {
        const options = this.currentWheel.options;
        const totalWeight = options.reduce((sum, opt) => sum + (opt.weight || 1), 0);
        let random = Math.random() * totalWeight;
        
        for (const option of options) {
            random -= option.weight || 1;
            if (random <= 0) {
                return {
                    result: option.label,
                    color: option.color,
                    timestamp: new Date().toISOString()
                };
            }
        }
        
        return options[0];
    }

    handleSpinResult(result) {
        this.isSpinning = false;
        const spinButton = document.getElementById('spinButton');
        spinButton.disabled = false;
        spinButton.querySelector('.spin-text').textContent = 'SPIN!';

        // Show result
        const resultDisplay = document.getElementById('resultDisplay');
        const resultText = document.getElementById('resultText');
        resultText.textContent = result.result;
        resultDisplay.classList.remove('hidden');

        // Play win sound and confetti
        if (this.settings.sound) {
            this.playSound('win');
        }
        if (this.settings.confetti) {
            this.launchConfetti();
        }

        // Add to history
        this.addToHistory(result);
    }

    addCustomOption() {
        const optionsList = document.getElementById('optionsList');
        const optionId = 'option_' + Date.now();
        
        const optionItem = document.createElement('div');
        optionItem.className = 'option-item';
        optionItem.id = optionId;
        optionItem.innerHTML = `
            <input type="text" class="option-input" placeholder="Option name" value="Option ${this.customOptions.length + 1}">
            <input type="color" class="color-picker" value="${this.getRandomColor()}">
            <input type="number" class="weight-input" min="0.1" max="10" step="0.1" value="1" placeholder="Weight">
            <button class="remove-btn" onclick="pickerWheel.removeCustomOption('${optionId}')">×</button>
        `;
        
        optionsList.appendChild(optionItem);
        
        const input = optionItem.querySelector('.option-input');
        const colorPicker = optionItem.querySelector('.color-picker');
        const weightInput = optionItem.querySelector('.weight-input');
        
        const option = {
            id: optionId,
            label: input.value,
            color: colorPicker.value,
            weight: parseFloat(weightInput.value)
        };
        
        this.customOptions.push(option);
        
        // Update on change
        input.addEventListener('change', () => {
            option.label = input.value;
            this.updateCustomWheel();
        });
        colorPicker.addEventListener('change', () => {
            option.color = colorPicker.value;
            this.updateCustomWheel();
        });
        weightInput.addEventListener('change', () => {
            option.weight = parseFloat(weightInput.value);
            this.updateCustomWheel();
        });
        
        this.updateCustomWheel();
    }

    removeCustomOption(optionId) {
        document.getElementById(optionId).remove();
        this.customOptions = this.customOptions.filter(opt => opt.id !== optionId);
        this.updateCustomWheel();
    }

    clearCustomOptions() {
        document.getElementById('optionsList').innerHTML = '';
        this.customOptions = [];
        this.updateCustomWheel();
    }

    updateCustomWheel() {
        if (this.customOptions.length > 0) {
            this.currentWheel = {
                name: 'Custom Wheel',
                options: this.customOptions
            };
            this.drawWheel(this.customOptions);
        }
    }

    async saveCustomWheel() {
        if (this.customOptions.length < 2) {
            alert('Please add at least 2 options to save the wheel');
            return;
        }
        
        const name = prompt('Enter a name for your wheel:');
        if (!name) return;
        
        try {
            const response = await fetch(`${API_BASE}/wheels`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: name,
                    options: this.customOptions,
                    session_id: this.sessionId
                })
            });
            
            if (response.ok) {
                alert('Wheel saved successfully!');
                this.loadSavedWheels();
            }
        } catch (error) {
            console.error('Save error:', error);
            // Save locally as fallback
            const savedWheels = JSON.parse(localStorage.getItem('savedWheels') || '[]');
            savedWheels.push({
                id: Date.now(),
                name: name,
                options: this.customOptions,
                created_at: new Date().toISOString()
            });
            localStorage.setItem('savedWheels', JSON.stringify(savedWheels));
            alert('Wheel saved locally!');
            this.loadSavedWheels();
        }
    }

    async loadSavedWheels() {
        try {
            const response = await fetch(`${API_BASE}/wheels?session_id=${this.sessionId}`);
            if (response.ok) {
                this.savedWheels = await response.json();
            }
        } catch (error) {
            // Load from local storage as fallback
            this.savedWheels = JSON.parse(localStorage.getItem('savedWheels') || '[]');
        }
        
        this.renderSavedWheels();
    }

    renderSavedWheels() {
        const container = document.getElementById('savedWheelsList');
        container.innerHTML = '';
        
        if (this.savedWheels.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: var(--text-secondary);">No saved wheels yet</p>';
            return;
        }
        
        this.savedWheels.forEach(wheel => {
            const wheelCard = document.createElement('div');
            wheelCard.className = 'saved-wheel-card';
            wheelCard.innerHTML = `
                <h4>${wheel.name}</h4>
                <p>${wheel.options.length} options</p>
                <button onclick="pickerWheel.loadSavedWheel('${wheel.id}')">Load</button>
                <button onclick="pickerWheel.deleteSavedWheel('${wheel.id}')">Delete</button>
            `;
            container.appendChild(wheelCard);
        });
    }

    loadSavedWheel(wheelId) {
        const wheel = this.savedWheels.find(w => w.id == wheelId);
        if (wheel) {
            this.currentWheel = wheel;
            this.drawWheel(wheel.options);
            this.switchTab('presets');
        }
    }

    async deleteSavedWheel(wheelId) {
        if (!confirm('Are you sure you want to delete this wheel?')) return;
        
        try {
            await fetch(`${API_BASE}/wheels/${wheelId}`, { method: 'DELETE' });
        } catch (error) {
            // Delete from local storage
            const savedWheels = JSON.parse(localStorage.getItem('savedWheels') || '[]');
            const filtered = savedWheels.filter(w => w.id != wheelId);
            localStorage.setItem('savedWheels', JSON.stringify(filtered));
        }
        
        this.loadSavedWheels();
    }

    addToHistory(result) {
        this.history.unshift({
            ...result,
            wheel: this.currentWheel.name,
            timestamp: new Date()
        });
        
        // Keep only last 50 items
        if (this.history.length > 50) {
            this.history.pop();
        }
        
        // Save to local storage
        localStorage.setItem('wheelHistory', JSON.stringify(this.history));
        
        this.updateHistoryDisplay();
    }

    loadHistory() {
        this.history = JSON.parse(localStorage.getItem('wheelHistory') || '[]');
        this.updateHistoryDisplay();
    }

    updateHistoryDisplay() {
        // Update stats
        const totalSpins = this.history.length;
        const todaySpins = this.history.filter(h => {
            const date = new Date(h.timestamp);
            const today = new Date();
            return date.toDateString() === today.toDateString();
        }).length;
        
        // Find most common result
        const resultCounts = {};
        this.history.forEach(h => {
            resultCounts[h.result] = (resultCounts[h.result] || 0) + 1;
        });
        const favoriteResult = Object.keys(resultCounts).reduce((a, b) => 
            resultCounts[a] > resultCounts[b] ? a : b, '-'
        );
        
        document.getElementById('totalSpins').textContent = totalSpins;
        document.getElementById('todaySpins').textContent = todaySpins;
        document.getElementById('favoriteResult').textContent = favoriteResult;
        
        // Update history list
        const historyList = document.getElementById('historyList');
        historyList.innerHTML = '';
        
        this.history.slice(0, 10).forEach(item => {
            const historyItem = document.createElement('div');
            historyItem.className = 'history-item';
            historyItem.innerHTML = `
                <span class="history-result">${item.result}</span>
                <span class="history-time">${this.formatTime(item.timestamp)}</span>
            `;
            historyList.appendChild(historyItem);
        });
    }

    formatTime(timestamp) {
        const date = new Date(timestamp);
        const now = new Date();
        const diff = now - date;
        
        if (diff < 60000) return 'Just now';
        if (diff < 3600000) return Math.floor(diff / 60000) + ' min ago';
        if (diff < 86400000) return Math.floor(diff / 3600000) + ' hours ago';
        return date.toLocaleDateString();
    }

    toggleSettings() {
        const menu = document.getElementById('settingsMenu');
        menu.classList.toggle('hidden');
    }

    loadSettings() {
        const saved = localStorage.getItem('wheelSettings');
        if (saved) {
            this.settings = JSON.parse(saved);
            document.getElementById('soundToggle').checked = this.settings.sound;
            document.getElementById('confettiToggle').checked = this.settings.confetti;
            document.getElementById('speedSelect').value = this.settings.speed;
            document.getElementById('themeSelect').value = this.settings.theme;
            this.applyTheme();
        }
    }

    saveSettings() {
        localStorage.setItem('wheelSettings', JSON.stringify(this.settings));
    }

    applyTheme() {
        document.body.className = 'theme-' + this.settings.theme;
    }

    playSound(type) {
        const audio = document.getElementById(type + 'Sound');
        if (audio) {
            audio.currentTime = 0;
            audio.play().catch(() => {});
        }
    }

    launchConfetti() {
        const canvas = document.getElementById('confettiCanvas');
        const ctx = canvas.getContext('2d');
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
        
        const particles = [];
        const colors = ['#FF00FF', '#00FFFF', '#FFD700', '#FF1744', '#00FF88'];
        
        for (let i = 0; i < 100; i++) {
            particles.push({
                x: Math.random() * canvas.width,
                y: -10,
                vx: (Math.random() - 0.5) * 5,
                vy: Math.random() * 3 + 2,
                color: colors[Math.floor(Math.random() * colors.length)],
                size: Math.random() * 3 + 2
            });
        }
        
        function animate() {
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            
            particles.forEach((p, index) => {
                p.x += p.vx;
                p.y += p.vy;
                p.vy += 0.1;
                
                ctx.fillStyle = p.color;
                ctx.fillRect(p.x, p.y, p.size, p.size);
                
                if (p.y > canvas.height) {
                    particles.splice(index, 1);
                }
            });
            
            if (particles.length > 0) {
                requestAnimationFrame(animate);
            }
        }
        
        animate();
    }

    getRandomColor() {
        const colors = ['#FF6B6B', '#4ECDC4', '#FFD93D', '#6C5CE7', '#A8E6CF', 
                       '#95E77E', '#FF1744', '#E91E63', '#9C27B0', '#3F51B5'];
        return colors[Math.floor(Math.random() * colors.length)];
    }
}

// Initialize the wheel when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.pickerWheel = new PickerWheel();
});