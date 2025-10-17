import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

// Ensure the iframe bridge is active when embedded in orchestrator preview
function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window || window.__choreTrackingBridgeInitialized) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[ChoreQuest] Unable to parse parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'chore-tracking' });
    window.__choreTrackingBridgeInitialized = true;
}

bootstrapIframeBridge();

// ChoreQuest - Gamified Chore Tracking Application
// Fun, quirky, and motivating household task management

class ChoreQuest {
    constructor() {
        this.currentView = 'quests';
        this.userData = {
            level: 1,
            points: 0,
            xp: 0,
            streak: 0,
            combo: 0,
            avatar: '🦸',
            displayName: 'Hero',
            completedQuests: 0,
            achievements: [],
            powerUps: {
                doubleXP: 0,
                timeFreeze: 0,
                helperBot: 0
            },
            dailyChallenge: null,
            lastActiveDate: new Date().toDateString()
        };
        this.quests = [];
        this.rewards = [];
        this.leaderboard = [];
        this.dailyChallenges = [];
        this.comboTimer = null;
        this.activePowerUp = null;
        this.soundEnabled = true;
        this.particles = [];
        this.init();
    }

    init() {
        this.loadUserData();
        this.checkDailyLogin();
        this.setupEventListeners();
        this.loadQuests();
        this.loadRewards();
        this.loadAchievements();
        this.loadLeaderboard();
        this.updateUserStats();
        
        // Generate daily challenge if needed
        if (!this.userData.dailyChallenge || this.userData.dailyChallenge.date !== new Date().toDateString()) {
            this.generateDailyChallenge();
        }
        
        // Show daily challenge notification
        if (this.userData.dailyChallenge) {
            setTimeout(() => {
                this.showNotification(`Today's Challenge: ${this.userData.dailyChallenge.description}`, '🎯');
            }, 2000);
        }
    }
    
    checkDailyLogin() {
        const today = new Date().toDateString();
        const yesterday = new Date(Date.now() - 86400000).toDateString();
        
        if (this.userData.lastActiveDate === yesterday) {
            // Consecutive day - increase streak
            this.userData.streak++;
            this.showNotification(`${this.userData.streak} day streak! Keep it up!`, '🔥');
            
            // Bonus points for streaks
            if (this.userData.streak % 7 === 0) {
                this.userData.points += 100;
                this.showNotification(`Weekly streak bonus: +100 points!`, '🎁');
            }
        } else if (this.userData.lastActiveDate !== today) {
            // Streak broken
            if (this.userData.streak > 0) {
                this.showNotification(`Streak broken! You had ${this.userData.streak} days`, '💔');
            }
            this.userData.streak = 1;
        }
        
        this.userData.lastActiveDate = today;
        this.saveUserData();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const view = e.currentTarget.dataset.view;
                this.switchView(view);
            });
        });

        // Add quest button
        const addQuestBtn = document.getElementById('add-quest-btn');
        if (addQuestBtn) {
            addQuestBtn.addEventListener('click', () => this.showAddQuestModal());
        }

        // Quest filters
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');
                this.filterQuests(e.target.dataset.filter);
            });
        });

        // Modal close
        const modalClose = document.getElementById('modal-close');
        if (modalClose) {
            modalClose.addEventListener('click', () => this.closeModal());
        }

        // Quest action buttons
        const startQuestBtn = document.getElementById('start-quest-btn');
        const completeQuestBtn = document.getElementById('complete-quest-btn');
        if (startQuestBtn) {
            startQuestBtn.addEventListener('click', () => this.startQuest());
        }
        if (completeQuestBtn) {
            completeQuestBtn.addEventListener('click', () => this.completeQuest());
        }
        
        // Power-up badges
        document.querySelectorAll('.power-up-badge').forEach(badge => {
            badge.addEventListener('click', (e) => {
                const type = e.currentTarget.dataset.type;
                this.activatePowerUp(type);
            });
        });
        
        // Sound toggle
        const soundToggle = document.getElementById('sound-toggle');
        if (soundToggle) {
            soundToggle.addEventListener('click', () => {
                this.soundEnabled = !this.soundEnabled;
                const soundIcon = document.getElementById('sound-icon');
                soundIcon.textContent = this.soundEnabled ? '🔊' : '🔇';
                soundToggle.classList.toggle('muted', !this.soundEnabled);
                this.playSound('click');
            });
        }
    }

    switchView(view) {
        // Update navigation
        document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
        document.querySelector(`[data-view="${view}"]`).classList.add('active');

        // Update content
        document.querySelectorAll('.view-container').forEach(container => {
            container.classList.remove('active');
        });
        const viewElement = document.getElementById(`${view}-view`);
        if (viewElement) {
            viewElement.classList.add('active');
        }

        this.currentView = view;
    }

    loadUserData() {
        // Load from localStorage or API
        const savedData = localStorage.getItem('chorequest-user');
        if (savedData) {
            this.userData = { ...this.userData, ...JSON.parse(savedData) };
        }
    }

    saveUserData() {
        localStorage.setItem('chorequest-user', JSON.stringify(this.userData));
        this.updateUserStats();
    }

    updateUserStats() {
        document.getElementById('user-level').textContent = this.userData.level;
        document.getElementById('user-points').textContent = this.userData.points;
        document.getElementById('user-streak').textContent = this.userData.streak;
        document.getElementById('user-avatar').textContent = this.userData.avatar;
        document.getElementById('level-badge').textContent = this.userData.level;
        
        const availablePoints = document.getElementById('available-points');
        if (availablePoints) {
            availablePoints.textContent = this.userData.points;
        }
        
        // Update power-up counts
        Object.keys(this.userData.powerUps).forEach(powerUp => {
            const countElement = document.getElementById(`${powerUp}-count`);
            if (countElement) {
                countElement.textContent = this.userData.powerUps[powerUp];
            }
        });
        
        // Update daily challenge display
        if (this.userData.dailyChallenge) {
            const challengeDesc = document.getElementById('challenge-description');
            const progressBar = document.getElementById('challenge-progress-bar');
            
            if (challengeDesc) {
                challengeDesc.textContent = this.userData.dailyChallenge.description;
            }
            
            if (progressBar) {
                const progress = (this.userData.dailyChallenge.progress / this.userData.dailyChallenge.target) * 100;
                progressBar.style.width = `${Math.min(progress, 100)}%`;
            }
        }
    }

    loadQuests() {
        // Sample quests - in production, would load from API
        this.quests = [
            {
                id: 1,
                title: "Slay the Dish Dragon",
                description: "Wash all the dishes in the sink",
                points: 15,
                difficulty: "daily",
                icon: "🍽️",
                status: "pending",
                timeEstimate: "15 min"
            },
            {
                id: 2,
                title: "Vacuum the Realm",
                description: "Vacuum the living room carpet",
                points: 25,
                difficulty: "weekly",
                icon: "🏠",
                status: "pending",
                timeEstimate: "30 min"
            },
            {
                id: 3,
                title: "Laundry Mountain Expedition",
                description: "Complete a full load of laundry",
                points: 30,
                difficulty: "weekly",
                icon: "👕",
                status: "pending",
                timeEstimate: "45 min"
            },
            {
                id: 4,
                title: "Bathroom Boss Battle",
                description: "Deep clean the bathroom",
                points: 50,
                difficulty: "epic",
                icon: "🚿",
                status: "pending",
                timeEstimate: "60 min"
            }
        ];

        this.renderQuests();
    }

    renderQuests() {
        const questsGrid = document.getElementById('quests-grid');
        if (!questsGrid) return;

        questsGrid.innerHTML = this.quests.map(quest => `
            <div class="quest-card ${quest.difficulty} ${quest.status}" data-quest-id="${quest.id}">
                <div class="quest-icon">${quest.icon}</div>
                <div class="quest-content">
                    <h3 class="quest-title">${quest.title}</h3>
                    <p class="quest-description">${quest.description}</p>
                    <div class="quest-meta">
                        <span class="quest-points">💎 ${quest.points} pts</span>
                        <span class="quest-time">⏱️ ${quest.timeEstimate}</span>
                    </div>
                </div>
                <div class="quest-status-badge">${this.getStatusEmoji(quest.status)}</div>
            </div>
        `).join('');

        // Add click handlers to quest cards
        document.querySelectorAll('.quest-card').forEach(card => {
            card.addEventListener('click', () => {
                const questId = parseInt(card.dataset.questId);
                this.showQuestDetails(questId);
            });
        });
    }

    getStatusEmoji(status) {
        const statusMap = {
            'pending': '⏳',
            'in-progress': '🎯',
            'completed': '✅'
        };
        return statusMap[status] || '❓';
    }

    showQuestDetails(questId) {
        const quest = this.quests.find(q => q.id === questId);
        if (!quest) return;

        this.currentQuest = quest;
        const modal = document.getElementById('quest-modal');
        const details = document.getElementById('quest-details');
        
        details.innerHTML = `
            <div class="quest-detail-header">
                <div class="quest-detail-icon">${quest.icon}</div>
                <h2>${quest.title}</h2>
            </div>
            <p class="quest-detail-description">${quest.description}</p>
            <div class="quest-detail-rewards">
                <div class="reward-item">
                    <span class="reward-icon">💎</span>
                    <span class="reward-text">${quest.points} Points</span>
                </div>
                <div class="reward-item">
                    <span class="reward-icon">⏱️</span>
                    <span class="reward-text">${quest.timeEstimate}</span>
                </div>
                <div class="reward-item">
                    <span class="reward-icon">🎯</span>
                    <span class="reward-text">${quest.difficulty}</span>
                </div>
            </div>
        `;

        // Update action buttons based on quest status
        const startBtn = document.getElementById('start-quest-btn');
        const completeBtn = document.getElementById('complete-quest-btn');
        
        if (quest.status === 'pending') {
            startBtn.style.display = 'block';
            completeBtn.style.display = 'none';
        } else if (quest.status === 'in-progress') {
            startBtn.style.display = 'none';
            completeBtn.style.display = 'block';
        } else {
            startBtn.style.display = 'none';
            completeBtn.style.display = 'none';
        }

        modal.classList.add('active');
    }

    closeModal() {
        document.getElementById('quest-modal').classList.remove('active');
    }

    startQuest() {
        if (!this.currentQuest) return;
        
        this.currentQuest.status = 'in-progress';
        this.renderQuests();
        this.showNotification('Quest Started!', '⚔️');
        this.closeModal();
        
        // Call API to update quest status
        this.updateQuestStatus(this.currentQuest.id, 'in-progress');
    }

    completeQuest() {
        if (!this.currentQuest) return;
        
        // Calculate points with combo and power-up multipliers
        let basePoints = this.currentQuest.points;
        let multiplier = 1;
        
        // Apply combo multiplier
        if (this.userData.combo > 0) {
            multiplier += (this.userData.combo * 0.1); // 10% bonus per combo
        }
        
        // Apply power-up multiplier
        if (this.activePowerUp === 'doubleXP') {
            multiplier *= 2;
        }
        
        const points = Math.floor(basePoints * multiplier);
        const xpGained = Math.floor(basePoints * 1.5 * multiplier);
        
        this.currentQuest.status = 'completed';
        this.userData.points += points;
        this.userData.xp += xpGained;
        this.userData.completedQuests++;
        
        // Update combo
        this.updateCombo();
        
        // Create particle effects
        this.createParticleExplosion();
        
        // Check for level up
        const oldLevel = this.userData.level;
        const xpForNextLevel = this.getXPForLevel(this.userData.level + 1);
        if (this.userData.xp >= xpForNextLevel) {
            this.userData.level++;
            this.userData.xp -= xpForNextLevel;
            this.showLevelUp(this.userData.level);
            this.unlockPowerUp();
        } else {
            this.showNotification(`Quest Complete! +${points} points (x${multiplier.toFixed(1)} combo!)`, '🎉');
        }
        
        // Check daily challenge
        this.checkDailyChallenge(this.currentQuest);
        
        this.saveUserData();
        this.renderQuests();
        this.closeModal();
        
        // Call API to update quest status
        this.updateQuestStatus(this.currentQuest.id, 'completed');
    }
    
    updateCombo() {
        this.userData.combo++;
        
        // Reset combo timer
        if (this.comboTimer) {
            clearTimeout(this.comboTimer);
        }
        
        // Combo expires after 2 hours
        this.comboTimer = setTimeout(() => {
            this.userData.combo = 0;
            this.showNotification('Combo expired! Complete quests to build it up again', '⏰');
            this.saveUserData();
        }, 2 * 60 * 60 * 1000);
        
        // Show combo notification
        if (this.userData.combo > 1) {
            this.showNotification(`${this.userData.combo}x Combo! Keep it going!`, '🔥');
        }
    }
    
    getXPForLevel(level) {
        // XP required increases with each level
        return level * 100 + (level - 1) * 50;
    }
    
    unlockPowerUp() {
        const powerUps = ['doubleXP', 'timeFreeze', 'helperBot'];
        const randomPowerUp = powerUps[Math.floor(Math.random() * powerUps.length)];
        this.userData.powerUps[randomPowerUp]++;
        this.showNotification(`Power-up unlocked: ${this.getPowerUpName(randomPowerUp)}!`, '⚡');
    }
    
    getPowerUpName(powerUp) {
        const names = {
            doubleXP: '2x XP Boost',
            timeFreeze: 'Time Freeze',
            helperBot: 'Helper Bot'
        };
        return names[powerUp] || powerUp;
    }
    
    activatePowerUp(type) {
        if (this.userData.powerUps[type] <= 0) return;
        
        this.userData.powerUps[type]--;
        this.activePowerUp = type;
        
        const duration = 30 * 60 * 1000; // 30 minutes
        setTimeout(() => {
            this.activePowerUp = null;
            this.showNotification(`Power-up expired: ${this.getPowerUpName(type)}`, '⏱️');
        }, duration);
        
        this.showNotification(`Power-up activated: ${this.getPowerUpName(type)}!`, '⚡');
        this.saveUserData();
    }
    
    createParticleExplosion() {
        // Create visual particle effects
        const particles = ['✨', '⭐', '💎', '🎉', '🎊', '🌟'];
        const colors = ['#F59E0B', '#EC4899', '#8B5CF6', '#3B82F6', '#10B981'];
        
        for (let i = 0; i < 20; i++) {
            const particle = document.createElement('div');
            particle.className = 'particle';
            particle.textContent = particles[Math.floor(Math.random() * particles.length)];
            particle.style.left = '50%';
            particle.style.top = '50%';
            particle.style.color = colors[Math.floor(Math.random() * colors.length)];
            
            // Random direction
            const angle = (Math.PI * 2 * i) / 20;
            const velocity = 100 + Math.random() * 100;
            particle.style.setProperty('--tx', `${Math.cos(angle) * velocity}px`);
            particle.style.setProperty('--ty', `${Math.sin(angle) * velocity}px`);
            
            document.body.appendChild(particle);
            
            setTimeout(() => particle.remove(), 1000);
        }
        
        this.playSound('success');
    }
    
    playSound(type) {
        if (!this.soundEnabled) return;
        
        // Sound effects placeholder - in production would use actual audio files
        const sounds = {
            'click': '🔊 Click!',
            'success': '🎉 Success!',
            'levelup': '🎺 Level Up!',
            'powerup': '⚡ Power Up!',
            'combo': '🔥 Combo!'
        };
        
        console.log(sounds[type] || '🔊 Sound!');
    }
    
    checkDailyChallenge(quest) {
        if (!this.userData.dailyChallenge) return;
        
        if (this.userData.dailyChallenge.type === quest.difficulty) {
            this.userData.dailyChallenge.progress++;
            
            if (this.userData.dailyChallenge.progress >= this.userData.dailyChallenge.target) {
                this.completeDailyChallenge();
            }
        }
    }
    
    completeDailyChallenge() {
        const bonusPoints = 100;
        const bonusXP = 150;
        
        this.userData.points += bonusPoints;
        this.userData.xp += bonusXP;
        
        this.showNotification(`Daily Challenge Complete! +${bonusPoints} points, +${bonusXP} XP!`, '🏆');
        this.userData.dailyChallenge = null;
        this.generateDailyChallenge();
        this.saveUserData();
    }
    
    generateDailyChallenge() {
        const challenges = [
            { type: 'daily', target: 3, description: 'Complete 3 daily quests' },
            { type: 'weekly', target: 2, description: 'Complete 2 weekly quests' },
            { type: 'epic', target: 1, description: 'Complete 1 epic quest' },
            { type: 'any', target: 5, description: 'Complete 5 quests of any type' }
        ];
        
        this.userData.dailyChallenge = {
            ...challenges[Math.floor(Math.random() * challenges.length)],
            progress: 0,
            date: new Date().toDateString()
        };
    }

    async updateQuestStatus(questId, status) {
        try {
            // Call n8n workflow via webhook
            const response = await fetch('/api/quests/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    questId,
                    status,
                    userId: this.userData.displayName,
                    timestamp: new Date().toISOString()
                })
            });
            
            if (!response.ok) {
                console.error('Failed to update quest status');
            }
        } catch (error) {
            console.error('Error updating quest:', error);
        }
    }

    loadRewards() {
        // Sample rewards
        this.rewards = [
            { id: 1, name: "Extra Screen Time", cost: 50, icon: "📱", description: "30 minutes bonus" },
            { id: 2, name: "Movie Night", cost: 100, icon: "🎬", description: "Choose the movie" },
            { id: 3, name: "Ice Cream Treat", cost: 75, icon: "🍦", description: "Your favorite flavor" },
            { id: 4, name: "Game Pass", cost: 150, icon: "🎮", description: "1 hour gaming" },
            { id: 5, name: "Late Bedtime", cost: 80, icon: "🌙", description: "Stay up 1 hour later" },
            { id: 6, name: "Pizza Party", cost: 200, icon: "🍕", description: "Order your favorite" }
        ];

        this.renderRewards();
    }

    renderRewards() {
        const rewardsGrid = document.getElementById('rewards-grid');
        if (!rewardsGrid) return;

        rewardsGrid.innerHTML = this.rewards.map(reward => `
            <div class="reward-card ${this.userData.points >= reward.cost ? 'available' : 'locked'}">
                <div class="reward-icon">${reward.icon}</div>
                <h3 class="reward-name">${reward.name}</h3>
                <p class="reward-description">${reward.description}</p>
                <div class="reward-cost">
                    <span class="cost-icon">💎</span>
                    <span class="cost-value">${reward.cost}</span>
                </div>
                <button class="claim-btn" ${this.userData.points >= reward.cost ? '' : 'disabled'} 
                        onclick="choreQuest.claimReward(${reward.id})">
                    ${this.userData.points >= reward.cost ? 'Claim' : 'Locked'}
                </button>
            </div>
        `).join('');
    }

    claimReward(rewardId) {
        const reward = this.rewards.find(r => r.id === rewardId);
        if (!reward || this.userData.points < reward.cost) return;

        this.userData.points -= reward.cost;
        this.saveUserData();
        this.renderRewards();
        this.showNotification(`${reward.name} claimed!`, reward.icon);
    }

    loadAchievements() {
        const achievements = [
            { id: 1, name: "First Quest", icon: "🌟", unlocked: this.userData.completedQuests >= 1 },
            { id: 2, name: "Week Warrior", icon: "⚔️", unlocked: this.userData.streak >= 7 },
            { id: 3, name: "Point Master", icon: "💎", unlocked: this.userData.points >= 500 },
            { id: 4, name: "Level 5 Hero", icon: "🎖️", unlocked: this.userData.level >= 5 },
            { id: 5, name: "Chore Champion", icon: "🏆", unlocked: this.userData.completedQuests >= 50 },
            { id: 6, name: "Super Streak", icon: "🔥", unlocked: this.userData.streak >= 30 }
        ];

        const achievementsGrid = document.getElementById('achievements-grid');
        if (achievementsGrid) {
            achievementsGrid.innerHTML = achievements.map(ach => `
                <div class="achievement-badge ${ach.unlocked ? 'unlocked' : 'locked'}">
                    <div class="achievement-icon">${ach.icon}</div>
                    <div class="achievement-name">${ach.name}</div>
                </div>
            `).join('');
        }
    }

    loadLeaderboard() {
        // Sample leaderboard data
        this.leaderboard = [
            { rank: 1, name: "ChoreKing", points: 2450, level: 12, avatar: "👑" },
            { rank: 2, name: "CleanQueen", points: 2200, level: 11, avatar: "👸" },
            { rank: 3, name: "TaskMaster", points: 1950, level: 10, avatar: "🦹" },
            { rank: 4, name: this.userData.displayName, points: this.userData.points, level: this.userData.level, avatar: this.userData.avatar },
            { rank: 5, name: "DustBuster", points: 850, level: 5, avatar: "🧹" }
        ];

        // Sort by points
        this.leaderboard.sort((a, b) => b.points - a.points);
        
        // Update ranks
        this.leaderboard.forEach((entry, index) => {
            entry.rank = index + 1;
        });

        this.renderLeaderboard();
    }

    renderLeaderboard() {
        const leaderboardList = document.getElementById('leaderboard-list');
        if (!leaderboardList) return;

        leaderboardList.innerHTML = this.leaderboard.map(entry => `
            <div class="leaderboard-entry ${entry.name === this.userData.displayName ? 'current-user' : ''}">
                <div class="rank-badge rank-${entry.rank}">
                    ${entry.rank <= 3 ? this.getRankEmoji(entry.rank) : entry.rank}
                </div>
                <div class="player-info">
                    <span class="player-avatar">${entry.avatar}</span>
                    <span class="player-name">${entry.name}</span>
                </div>
                <div class="player-stats">
                    <span class="player-level">Lvl ${entry.level}</span>
                    <span class="player-points">💎 ${entry.points}</span>
                </div>
            </div>
        `).join('');
    }

    getRankEmoji(rank) {
        const rankEmojis = { 1: '🥇', 2: '🥈', 3: '🥉' };
        return rankEmojis[rank] || rank;
    }

    showNotification(message, icon = '🎉') {
        const notification = document.getElementById('notification');
        const notificationIcon = document.getElementById('notification-icon');
        const notificationMessage = document.getElementById('notification-message');
        
        notificationIcon.textContent = icon;
        notificationMessage.textContent = message;
        notification.classList.add('show');
        
        setTimeout(() => {
            notification.classList.remove('show');
        }, 3000);
    }

    showLevelUp(newLevel) {
        const levelUpElement = document.getElementById('level-up');
        const newLevelElement = document.getElementById('new-level');
        const levelRewards = document.getElementById('level-rewards');
        
        newLevelElement.textContent = newLevel;
        levelRewards.innerHTML = `
            <div class="level-reward">🎁 Bonus: 50 points!</div>
            <div class="level-reward">🏆 New Achievement Unlocked!</div>
        `;
        
        this.userData.points += 50;
        this.saveUserData();
        
        levelUpElement.classList.add('show');
        
        setTimeout(() => {
            levelUpElement.classList.remove('show');
        }, 4000);
    }

    filterQuests(filter) {
        const questCards = document.querySelectorAll('.quest-card');
        questCards.forEach(card => {
            if (filter === 'all') {
                card.style.display = 'block';
            } else {
                const quest = this.quests.find(q => q.id === parseInt(card.dataset.questId));
                card.style.display = quest && quest.difficulty === filter ? 'block' : 'none';
            }
        });
    }

    showAddQuestModal() {
        // This would open a modal to add a new quest
        // For now, just show a notification
        this.showNotification('Quest editor coming soon!', '🚧');
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.choreQuest = new ChoreQuest();
});
