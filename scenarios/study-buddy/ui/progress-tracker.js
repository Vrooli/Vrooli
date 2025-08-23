// Real-time Progress Tracking Module for Study Buddy
class ProgressTracker {
    constructor() {
        this.sessionData = {
            startTime: Date.now(),
            cardsReviewed: 0,
            correctAnswers: 0,
            streak: 0,
            maxStreak: 0,
            focusTime: 0,
            breakTime: 0,
            lastActivity: Date.now()
        };
        
        this.adaptiveMetrics = {
            comprehensionRate: 0,
            retentionScore: 0,
            difficultyTrend: [],
            topicMastery: {}
        };
        
        this.init();
    }

    init() {
        this.createProgressUI();
        this.startRealTimeTracking();
        this.initializeAnimations();
        this.connectToAdaptiveEngine();
    }

    createProgressUI() {
        const progressContainer = document.createElement('div');
        progressContainer.className = 'progress-tracker-container';
        progressContainer.innerHTML = `
            <div class="progress-header">
                <div class="progress-title">
                    <span class="progress-icon">📊</span>
                    <span>Live Progress</span>
                </div>
                <button class="progress-toggle">
                    <svg width="16" height="16" viewBox="0 0 16 16">
                        <path d="M8 12l-6-6h12z" fill="currentColor"/>
                    </svg>
                </button>
            </div>
            
            <div class="progress-content">
                <!-- Adaptive Learning Meter -->
                <div class="adaptive-meter">
                    <div class="meter-label">Adaptation Level</div>
                    <div class="meter-bar">
                        <div class="meter-fill" id="adaptive-fill"></div>
                        <div class="meter-particles"></div>
                    </div>
                    <div class="meter-value" id="adaptive-value">Analyzing...</div>
                </div>

                <!-- Real-time Stats Grid -->
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-value" id="accuracy-stat">0%</div>
                        <div class="stat-label">Accuracy</div>
                        <div class="stat-trend" id="accuracy-trend"></div>
                    </div>
                    
                    <div class="stat-card">
                        <div class="stat-value" id="streak-stat">0</div>
                        <div class="stat-label">Streak</div>
                        <div class="streak-fire" id="streak-fire">🔥</div>
                    </div>
                    
                    <div class="stat-card">
                        <div class="stat-value" id="focus-stat">0m</div>
                        <div class="stat-label">Focus Time</div>
                        <div class="focus-ring" id="focus-ring"></div>
                    </div>
                    
                    <div class="stat-card">
                        <div class="stat-value" id="mastery-stat">0</div>
                        <div class="stat-label">Cards Mastered</div>
                        <div class="mastery-stars" id="mastery-stars"></div>
                    </div>
                </div>

                <!-- Learning Velocity Chart -->
                <div class="velocity-chart">
                    <canvas id="velocity-canvas" width="280" height="100"></canvas>
                    <div class="velocity-label">Learning Velocity</div>
                </div>

                <!-- Achievement Notifications -->
                <div class="achievement-container" id="achievements"></div>
            </div>
        `;

        // Add styles
        const styles = `
            <style>
                .progress-tracker-container {
                    position: fixed;
                    top: 80px;
                    right: 20px;
                    width: 320px;
                    background: linear-gradient(135deg, rgba(255, 255, 255, 0.98) 0%, rgba(248, 241, 255, 0.98) 100%);
                    border-radius: 20px;
                    box-shadow: 0 10px 40px rgba(107, 70, 122, 0.2);
                    z-index: 1000;
                    overflow: hidden;
                    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                }

                .progress-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 1rem 1.25rem;
                    background: linear-gradient(135deg, var(--accent-purple) 0%, var(--accent-pink) 100%);
                    color: white;
                }

                .progress-title {
                    display: flex;
                    align-items: center;
                    gap: 0.5rem;
                    font-weight: 600;
                }

                .progress-toggle {
                    background: rgba(255, 255, 255, 0.2);
                    border: none;
                    border-radius: 50%;
                    width: 28px;
                    height: 28px;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    cursor: pointer;
                    transition: all 0.2s;
                }

                .progress-toggle:hover {
                    background: rgba(255, 255, 255, 0.3);
                    transform: scale(1.1);
                }

                .progress-content {
                    padding: 1.25rem;
                }

                .adaptive-meter {
                    margin-bottom: 1.5rem;
                }

                .meter-label {
                    font-size: 0.85rem;
                    color: var(--text-light);
                    margin-bottom: 0.5rem;
                }

                .meter-bar {
                    height: 24px;
                    background: rgba(155, 114, 170, 0.1);
                    border-radius: 12px;
                    overflow: hidden;
                    position: relative;
                }

                .meter-fill {
                    height: 100%;
                    background: linear-gradient(90deg, #9b72aa 0%, #d4a5c8 100%);
                    border-radius: 12px;
                    width: 0%;
                    transition: width 1s cubic-bezier(0.4, 0, 0.2, 1);
                    position: relative;
                }

                .meter-fill::after {
                    content: '';
                    position: absolute;
                    top: 0;
                    left: 0;
                    right: 0;
                    bottom: 0;
                    background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.4), transparent);
                    animation: shimmer 2s infinite;
                }

                @keyframes shimmer {
                    0% { transform: translateX(-100%); }
                    100% { transform: translateX(100%); }
                }

                .meter-particles {
                    position: absolute;
                    top: 0;
                    left: 0;
                    width: 100%;
                    height: 100%;
                    pointer-events: none;
                }

                .particle {
                    position: absolute;
                    width: 4px;
                    height: 4px;
                    background: var(--accent-yellow);
                    border-radius: 50%;
                    animation: float-particle 3s infinite;
                }

                @keyframes float-particle {
                    0% { transform: translateY(24px) scale(0); opacity: 0; }
                    50% { opacity: 1; }
                    100% { transform: translateY(-20px) scale(1); opacity: 0; }
                }

                .stats-grid {
                    display: grid;
                    grid-template-columns: repeat(2, 1fr);
                    gap: 1rem;
                    margin-bottom: 1.5rem;
                }

                .stat-card {
                    background: white;
                    padding: 1rem;
                    border-radius: 12px;
                    text-align: center;
                    position: relative;
                    overflow: hidden;
                    transition: transform 0.2s;
                }

                .stat-card:hover {
                    transform: translateY(-2px);
                    box-shadow: 0 4px 12px rgba(107, 70, 122, 0.1);
                }

                .stat-value {
                    font-size: 1.5rem;
                    font-weight: 700;
                    background: linear-gradient(135deg, var(--accent-purple) 0%, var(--accent-pink) 100%);
                    -webkit-background-clip: text;
                    -webkit-text-fill-color: transparent;
                    background-clip: text;
                }

                .stat-label {
                    font-size: 0.75rem;
                    color: var(--text-light);
                    margin-top: 0.25rem;
                }

                .streak-fire {
                    position: absolute;
                    top: 10px;
                    right: 10px;
                    font-size: 1.2rem;
                    opacity: 0;
                    transition: opacity 0.3s;
                }

                .streak-fire.active {
                    opacity: 1;
                    animation: fire-bounce 0.5s infinite alternate;
                }

                @keyframes fire-bounce {
                    0% { transform: scale(1); }
                    100% { transform: scale(1.2); }
                }

                .velocity-chart {
                    background: white;
                    border-radius: 12px;
                    padding: 1rem;
                    position: relative;
                }

                .velocity-label {
                    position: absolute;
                    bottom: 0.5rem;
                    left: 1rem;
                    font-size: 0.75rem;
                    color: var(--text-light);
                }

                .achievement-container {
                    position: absolute;
                    bottom: -60px;
                    left: 50%;
                    transform: translateX(-50%);
                    width: 90%;
                }

                .achievement {
                    background: linear-gradient(135deg, var(--accent-yellow) 0%, #ffd98f 100%);
                    color: var(--text-dark);
                    padding: 0.75rem 1rem;
                    border-radius: 12px;
                    text-align: center;
                    font-weight: 600;
                    box-shadow: 0 4px 12px rgba(249, 217, 120, 0.4);
                    animation: achievement-pop 0.5s cubic-bezier(0.68, -0.55, 0.265, 1.55);
                }

                @keyframes achievement-pop {
                    0% { transform: scale(0) translateY(20px); opacity: 0; }
                    100% { transform: scale(1) translateY(0); opacity: 1; }
                }

                .progress-tracker-container.collapsed {
                    height: 60px;
                }

                .progress-tracker-container.collapsed .progress-content {
                    display: none;
                }
            </style>
        `;

        document.head.insertAdjacentHTML('beforeend', styles);
        document.body.appendChild(progressContainer);

        // Toggle functionality
        document.querySelector('.progress-toggle').addEventListener('click', () => {
            progressContainer.classList.toggle('collapsed');
        });
    }

    startRealTimeTracking() {
        // Update every second
        setInterval(() => {
            this.updateFocusTime();
            this.updateUI();
        }, 1000);

        // Check for achievements every 10 seconds
        setInterval(() => {
            this.checkAchievements();
        }, 10000);

        // Sync with adaptive engine every 30 seconds
        setInterval(() => {
            this.syncWithAdaptiveEngine();
        }, 30000);
    }

    updateFocusTime() {
        const now = Date.now();
        const timeSinceLastActivity = now - this.sessionData.lastActivity;
        
        if (timeSinceLastActivity < 120000) { // Active if within 2 minutes
            this.sessionData.focusTime++;
        } else {
            this.sessionData.breakTime++;
        }
    }

    updateUI() {
        // Update accuracy
        const accuracy = this.sessionData.cardsReviewed > 0 
            ? Math.round((this.sessionData.correctAnswers / this.sessionData.cardsReviewed) * 100)
            : 0;
        document.getElementById('accuracy-stat').textContent = `${accuracy}%`;

        // Update streak with fire effect
        document.getElementById('streak-stat').textContent = this.sessionData.streak;
        const fireElement = document.getElementById('streak-fire');
        if (this.sessionData.streak >= 5) {
            fireElement.classList.add('active');
        } else {
            fireElement.classList.remove('active');
        }

        // Update focus time
        const focusMinutes = Math.floor(this.sessionData.focusTime / 60);
        document.getElementById('focus-stat').textContent = `${focusMinutes}m`;

        // Update mastery count
        const masteryCount = Object.values(this.adaptiveMetrics.topicMastery)
            .filter(score => score >= 0.8).length;
        document.getElementById('mastery-stat').textContent = masteryCount;
    }

    recordActivity(type, data) {
        this.sessionData.lastActivity = Date.now();
        
        switch(type) {
            case 'card_reviewed':
                this.sessionData.cardsReviewed++;
                if (data.correct) {
                    this.sessionData.correctAnswers++;
                    this.sessionData.streak++;
                    this.sessionData.maxStreak = Math.max(this.sessionData.streak, this.sessionData.maxStreak);
                } else {
                    this.sessionData.streak = 0;
                }
                this.updateAdaptiveMetrics(data);
                break;
                
            case 'quiz_answered':
                this.sessionData.cardsReviewed++;
                if (data.correct) {
                    this.sessionData.correctAnswers++;
                }
                break;
        }
        
        this.updateUI();
        this.animateProgress(type, data.correct);
    }

    updateAdaptiveMetrics(data) {
        // Update comprehension rate
        this.adaptiveMetrics.comprehensionRate = 
            this.adaptiveMetrics.comprehensionRate * 0.9 + (data.correct ? 0.1 : 0);
        
        // Update topic mastery
        if (data.topic) {
            if (!this.adaptiveMetrics.topicMastery[data.topic]) {
                this.adaptiveMetrics.topicMastery[data.topic] = 0;
            }
            this.adaptiveMetrics.topicMastery[data.topic] = 
                this.adaptiveMetrics.topicMastery[data.topic] * 0.8 + (data.correct ? 0.2 : 0);
        }
        
        // Update difficulty trend
        this.adaptiveMetrics.difficultyTrend.push(data.difficulty || 2);
        if (this.adaptiveMetrics.difficultyTrend.length > 10) {
            this.adaptiveMetrics.difficultyTrend.shift();
        }
        
        // Update adaptive meter
        const adaptiveScore = this.calculateAdaptiveScore();
        this.updateAdaptiveMeter(adaptiveScore);
    }

    calculateAdaptiveScore() {
        const accuracy = this.sessionData.cardsReviewed > 0 
            ? this.sessionData.correctAnswers / this.sessionData.cardsReviewed 
            : 0;
        
        const consistency = this.sessionData.maxStreak > 0 
            ? this.sessionData.streak / this.sessionData.maxStreak 
            : 0;
        
        const mastery = Object.values(this.adaptiveMetrics.topicMastery).length > 0
            ? Object.values(this.adaptiveMetrics.topicMastery).reduce((a, b) => a + b, 0) / 
              Object.values(this.adaptiveMetrics.topicMastery).length
            : 0;
        
        return (accuracy * 0.4 + consistency * 0.3 + mastery * 0.3) * 100;
    }

    updateAdaptiveMeter(score) {
        const fill = document.getElementById('adaptive-fill');
        const value = document.getElementById('adaptive-value');
        
        fill.style.width = `${Math.min(100, score)}%`;
        
        let label = 'Warming Up';
        if (score >= 80) label = 'Peak Performance! 🚀';
        else if (score >= 60) label = 'Great Progress! ⭐';
        else if (score >= 40) label = 'Building Momentum 📈';
        else if (score >= 20) label = 'Getting Started 💪';
        
        value.textContent = label;
        
        // Add particles for high scores
        if (score >= 60) {
            this.createParticles();
        }
    }

    createParticles() {
        const container = document.querySelector('.meter-particles');
        const particle = document.createElement('div');
        particle.className = 'particle';
        particle.style.left = `${Math.random() * 100}%`;
        particle.style.animationDelay = `${Math.random() * 2}s`;
        container.appendChild(particle);
        
        setTimeout(() => particle.remove(), 3000);
    }

    animateProgress(type, success) {
        const card = document.querySelector(`.stat-card:nth-child(${success ? 1 : 2})`);
        card.style.animation = 'pulse 0.5s';
        setTimeout(() => {
            card.style.animation = '';
        }, 500);
    }

    checkAchievements() {
        const achievements = [];
        
        if (this.sessionData.streak === 10) {
            achievements.push('🔥 10 Streak! Keep it up!');
        }
        
        if (this.sessionData.cardsReviewed === 50) {
            achievements.push('📚 50 Cards Reviewed!');
        }
        
        if (this.sessionData.focusTime === 1800) { // 30 minutes
            achievements.push('⏰ 30 Minutes of Focus!');
        }
        
        const accuracy = this.sessionData.cardsReviewed > 0 
            ? this.sessionData.correctAnswers / this.sessionData.cardsReviewed 
            : 0;
        if (accuracy >= 0.9 && this.sessionData.cardsReviewed >= 20) {
            achievements.push('🎯 90% Accuracy Master!');
        }
        
        achievements.forEach((achievement, index) => {
            setTimeout(() => this.showAchievement(achievement), index * 1000);
        });
    }

    showAchievement(text) {
        const container = document.getElementById('achievements');
        const achievement = document.createElement('div');
        achievement.className = 'achievement';
        achievement.textContent = text;
        container.appendChild(achievement);
        
        setTimeout(() => achievement.remove(), 5000);
    }

    async syncWithAdaptiveEngine() {
        try {
            const response = await fetch('http://localhost:5678/webhook/learning/adapt', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    user_id: 'current-user',
                    subject_id: currentSubject,
                    performance_data: {
                        recent_scores: this.getRecentScores(),
                        time_spent: [this.sessionData.focusTime],
                        difficulty_ratings: this.adaptiveMetrics.difficultyTrend,
                        topics_covered: Object.keys(this.adaptiveMetrics.topicMastery)
                    }
                })
            });
            
            if (response.ok) {
                const adaptivePlan = await response.json();
                this.applyAdaptivePlan(adaptivePlan);
            }
        } catch (error) {
            console.error('Failed to sync with adaptive engine:', error);
        }
    }

    getRecentScores() {
        // Calculate scores for recent sessions
        const recentScores = [];
        const batchSize = 10;
        for (let i = 0; i < this.sessionData.cardsReviewed; i += batchSize) {
            const batch = Math.min(batchSize, this.sessionData.cardsReviewed - i);
            const score = Math.random() * 0.3 + 0.6; // Simulate varying scores
            recentScores.push(score);
        }
        return recentScores.slice(-4);
    }

    applyAdaptivePlan(plan) {
        console.log('Applying adaptive plan:', plan);
        // Update UI or adjust difficulty based on plan
        if (plan.data?.adaptive_settings) {
            // Apply settings like session duration, break frequency, etc.
            this.sessionSettings = plan.data.adaptive_settings;
        }
    }

    async connectToAdaptiveEngine() {
        // Initialize connection to adaptive learning engine
        console.log('Connecting to Adaptive Learning Engine...');
        await this.syncWithAdaptiveEngine();
    }

    initializeAnimations() {
        // Add CSS animations
        const style = document.createElement('style');
        style.textContent = `
            @keyframes pulse {
                0% { transform: scale(1); }
                50% { transform: scale(1.05); }
                100% { transform: scale(1); }
            }
        `;
        document.head.appendChild(style);
    }

    drawVelocityChart() {
        const canvas = document.getElementById('velocity-canvas');
        if (!canvas) return;
        
        const ctx = canvas.getContext('2d');
        const width = canvas.width;
        const height = canvas.height;
        
        // Clear canvas
        ctx.clearRect(0, 0, width, height);
        
        // Draw gradient background
        const gradient = ctx.createLinearGradient(0, height, 0, 0);
        gradient.addColorStop(0, 'rgba(155, 114, 170, 0)');
        gradient.addColorStop(1, 'rgba(155, 114, 170, 0.2)');
        
        // Generate sample data
        const points = [];
        for (let i = 0; i < 10; i++) {
            points.push({
                x: (width / 10) * i,
                y: height - (Math.random() * height * 0.7 + height * 0.2)
            });
        }
        
        // Draw curve
        ctx.beginPath();
        ctx.moveTo(0, height);
        
        points.forEach((point, i) => {
            if (i === 0) {
                ctx.lineTo(point.x, point.y);
            } else {
                const prevPoint = points[i - 1];
                const cpx = (prevPoint.x + point.x) / 2;
                const cpy = (prevPoint.y + point.y) / 2;
                ctx.quadraticCurveTo(prevPoint.x, prevPoint.y, cpx, cpy);
            }
        });
        
        ctx.lineTo(width, height);
        ctx.fillStyle = gradient;
        ctx.fill();
        
        // Draw line
        ctx.beginPath();
        ctx.strokeStyle = 'rgba(155, 114, 170, 0.8)';
        ctx.lineWidth = 2;
        
        points.forEach((point, i) => {
            if (i === 0) {
                ctx.moveTo(point.x, point.y);
            } else {
                const prevPoint = points[i - 1];
                const cpx = (prevPoint.x + point.x) / 2;
                const cpy = (prevPoint.y + point.y) / 2;
                ctx.quadraticCurveTo(prevPoint.x, prevPoint.y, cpx, cpy);
            }
        });
        
        ctx.stroke();
    }
}

// Initialize progress tracker when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.progressTracker = new ProgressTracker();
    });
} else {
    window.progressTracker = new ProgressTracker();
}

// Export for use in main script
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ProgressTracker;
}