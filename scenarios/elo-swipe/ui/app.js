// Elo Swipe Application
class EloSwipe {
    constructor() {
        this.apiUrl = `http://localhost:${window.API_PORT || 30400}/api/v1`;
        this.currentList = null;
        this.currentComparison = null;
        this.comparisonHistory = [];
        this.rankings = [];
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadLists();
        this.checkForActiveSession();
    }

    setupEventListeners() {
        // Navigation
        document.getElementById('listSelector').addEventListener('click', () => this.toggleListDropdown());
        document.getElementById('viewRankings').addEventListener('click', () => this.showRankings());
        document.getElementById('createList').addEventListener('click', () => this.showCreateListModal());
        document.getElementById('startRanking').addEventListener('click', () => this.startRanking());

        // Actions
        document.getElementById('skipBtn').addEventListener('click', () => this.skipComparison());
        document.getElementById('undoBtn').addEventListener('click', () => this.undoComparison());

        // Modal controls
        document.getElementById('closeRankings').addEventListener('click', () => this.hideModal('rankingsModal'));
        document.getElementById('closeCreateList').addEventListener('click', () => this.hideModal('createListModal'));
        document.getElementById('continueRanking').addEventListener('click', () => {
            this.hideModal('rankingsModal');
            this.loadNextComparison();
        });
        document.getElementById('exportRankings').addEventListener('click', () => this.exportRankings());

        // Form
        document.getElementById('createListForm').addEventListener('submit', (e) => this.handleCreateList(e));
        document.getElementById('importItems').addEventListener('click', () => this.importItems());

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => this.handleKeyboard(e));

        // Touch/swipe support
        this.setupSwipeGestures();
    }

    handleKeyboard(e) {
        if (this.currentComparison) {
            switch(e.key) {
                case 'ArrowLeft':
                    e.preventDefault();
                    this.selectWinner('left');
                    break;
                case 'ArrowRight':
                    e.preventDefault();
                    this.selectWinner('right');
                    break;
                case ' ':
                    e.preventDefault();
                    this.skipComparison();
                    break;
                case 'z':
                case 'Z':
                    if (e.ctrlKey || e.metaKey) {
                        e.preventDefault();
                        this.undoComparison();
                    }
                    break;
            }
        }
    }

    setupSwipeGestures() {
        let startX = null;
        let currentCard = null;

        const handleStart = (e) => {
            const card = e.target.closest('.comparison-card');
            if (card) {
                currentCard = card;
                startX = e.touches ? e.touches[0].clientX : e.clientX;
                card.style.transition = 'none';
            }
        };

        const handleMove = (e) => {
            if (!startX || !currentCard) return;
            
            const currentX = e.touches ? e.touches[0].clientX : e.clientX;
            const diffX = currentX - startX;
            
            currentCard.style.transform = `translateX(${diffX}px) rotate(${diffX * 0.05}deg)`;
            currentCard.style.opacity = 1 - Math.abs(diffX) / 300;
        };

        const handleEnd = (e) => {
            if (!currentCard) return;
            
            const endX = e.changedTouches ? e.changedTouches[0].clientX : e.clientX;
            const diffX = endX - startX;
            
            currentCard.style.transition = 'all 0.3s ease';
            
            if (Math.abs(diffX) > 100) {
                const side = currentCard.classList.contains('left') ? 'left' : 'right';
                this.selectWinner(side);
            } else {
                currentCard.style.transform = '';
                currentCard.style.opacity = '';
            }
            
            startX = null;
            currentCard = null;
        };

        document.addEventListener('mousedown', handleStart);
        document.addEventListener('mousemove', handleMove);
        document.addEventListener('mouseup', handleEnd);
        document.addEventListener('touchstart', handleStart);
        document.addEventListener('touchmove', handleMove);
        document.addEventListener('touchend', handleEnd);
    }

    async loadLists() {
        try {
            const response = await fetch(`${this.apiUrl}/lists`);
            const lists = await response.json();
            this.updateListDropdown(lists);
        } catch (error) {
            console.error('Failed to load lists:', error);
        }
    }

    updateListDropdown(lists) {
        const dropdown = document.getElementById('listDropdown');
        const content = dropdown.querySelector('.dropdown-content');
        
        content.innerHTML = lists.map(list => `
            <div class="dropdown-item ${list.id === this.currentList?.id ? 'active' : ''}" 
                 data-list-id="${list.id}">
                <div>${list.name}</div>
                <div style="font-size: 12px; color: var(--gray);">${list.item_count} items</div>
            </div>
        `).join('');
        
        content.querySelectorAll('.dropdown-item').forEach(item => {
            item.addEventListener('click', () => this.selectList(item.dataset.listId));
        });
    }

    async selectList(listId) {
        try {
            const response = await fetch(`${this.apiUrl}/lists/${listId}`);
            this.currentList = await response.json();
            
            document.querySelector('.list-name').textContent = this.currentList.name;
            this.hideDropdown();
            this.loadNextComparison();
        } catch (error) {
            console.error('Failed to select list:', error);
        }
    }

    toggleListDropdown() {
        const dropdown = document.getElementById('listDropdown');
        dropdown.style.display = dropdown.style.display === 'none' ? 'block' : 'none';
    }

    hideDropdown() {
        document.getElementById('listDropdown').style.display = 'none';
    }

    async startRanking() {
        if (!this.currentList) {
            alert('Please select or create a list first');
            return;
        }
        
        document.getElementById('instructions').style.display = 'none';
        document.getElementById('cardStack').style.display = 'block';
        
        this.loadNextComparison();
    }

    async loadNextComparison() {
        if (!this.currentList) return;
        
        try {
            const response = await fetch(`${this.apiUrl}/lists/${this.currentList.id}/next-comparison`);
            
            if (!response.ok) {
                this.showComplete();
                return;
            }
            
            this.currentComparison = await response.json();
            this.renderComparison();
            this.updateProgress();
        } catch (error) {
            console.error('Failed to load comparison:', error);
        }
    }

    renderComparison() {
        const stack = document.getElementById('cardStack');
        
        stack.innerHTML = `
            <div class="comparison-card left" data-item-id="${this.currentComparison.item_a.id}">
                <span class="card-label left">Option A</span>
                <div class="card-content">${this.formatContent(this.currentComparison.item_a.content)}</div>
            </div>
            <div class="comparison-card right" data-item-id="${this.currentComparison.item_b.id}">
                <span class="card-label right">Option B</span>
                <div class="card-content">${this.formatContent(this.currentComparison.item_b.content)}</div>
            </div>
            <div class="vs-indicator">VS</div>
        `;
        
        // Add click handlers
        stack.querySelectorAll('.comparison-card').forEach(card => {
            card.addEventListener('click', () => {
                const side = card.classList.contains('left') ? 'left' : 'right';
                this.selectWinner(side);
            });
        });
    }

    formatContent(content) {
        if (typeof content === 'string') return content;
        if (content.title) return content.title;
        if (content.name) return content.name;
        if (content.text) return content.text;
        return JSON.stringify(content);
    }

    async selectWinner(side) {
        if (!this.currentComparison) return;
        
        const winnerId = side === 'left' 
            ? this.currentComparison.item_a.id 
            : this.currentComparison.item_b.id;
        const loserId = side === 'left' 
            ? this.currentComparison.item_b.id 
            : this.currentComparison.item_a.id;
        
        // Animate selection
        const cards = document.querySelectorAll('.comparison-card');
        cards.forEach(card => {
            if (card.classList.contains(side)) {
                card.classList.add(`swipe-${side}`);
            } else {
                card.style.opacity = '0.5';
            }
        });
        
        // Submit comparison
        try {
            const response = await fetch(`${this.apiUrl}/comparisons`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    list_id: this.currentList.id,
                    winner_id: winnerId,
                    loser_id: loserId
                })
            });
            
            const result = await response.json();
            this.comparisonHistory.push(result);
            
            // Load next comparison after animation
            setTimeout(() => this.loadNextComparison(), 500);
        } catch (error) {
            console.error('Failed to submit comparison:', error);
        }
    }

    async skipComparison() {
        if (!this.currentComparison) return;
        
        // Animate skip
        const cards = document.querySelectorAll('.comparison-card');
        cards.forEach(card => {
            card.style.transform = 'scale(0.9)';
            card.style.opacity = '0.5';
        });
        
        setTimeout(() => this.loadNextComparison(), 300);
    }

    async undoComparison() {
        if (this.comparisonHistory.length === 0) return;
        
        const lastComparison = this.comparisonHistory.pop();
        
        try {
            await fetch(`${this.apiUrl}/comparisons/${lastComparison.id}`, {
                method: 'DELETE'
            });
            
            this.loadNextComparison();
        } catch (error) {
            console.error('Failed to undo comparison:', error);
        }
    }

    updateProgress() {
        if (!this.currentComparison?.progress) return;
        
        const { completed, total } = this.currentComparison.progress;
        const percentage = (completed / total) * 100;
        const confidence = Math.min(95, completed * 5); // Rough confidence estimate
        
        document.querySelector('.comparisons-made').textContent = `${completed} comparisons`;
        document.querySelector('.confidence-level').textContent = `${confidence}% confident`;
        document.querySelector('.progress-fill').style.width = `${percentage}%`;
    }

    async showRankings() {
        if (!this.currentList) {
            alert('Please select a list first');
            return;
        }
        
        try {
            const response = await fetch(`${this.apiUrl}/lists/${this.currentList.id}/rankings`);
            this.rankings = await response.json();
            
            this.renderRankings();
            this.showModal('rankingsModal');
        } catch (error) {
            console.error('Failed to load rankings:', error);
        }
    }

    renderRankings() {
        const container = document.getElementById('rankingsList');
        
        container.innerHTML = this.rankings.rankings.map((item, index) => `
            <div class="ranking-item">
                <div class="ranking-position">${index + 1}</div>
                <div class="ranking-content">
                    <div>${this.formatContent(item.item)}</div>
                    <div class="ranking-stats">
                        <span>Rating: <span class="rating-badge">${item.elo_rating.toFixed(0)}</span></span>
                        <span>Confidence: ${(item.confidence * 100).toFixed(0)}%</span>
                    </div>
                </div>
            </div>
        `).join('');
    }

    exportRankings() {
        if (!this.rankings) return;
        
        const csv = [
            ['Rank', 'Item', 'Elo Rating', 'Confidence'],
            ...this.rankings.rankings.map((item, index) => [
                index + 1,
                this.formatContent(item.item),
                item.elo_rating.toFixed(0),
                (item.confidence * 100).toFixed(0) + '%'
            ])
        ].map(row => row.join(',')).join('\n');
        
        const blob = new Blob([csv], { type: 'text/csv' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${this.currentList.name}-rankings.csv`;
        a.click();
        URL.revokeObjectURL(url);
    }

    showCreateListModal() {
        this.showModal('createListModal');
    }

    async handleCreateList(e) {
        e.preventDefault();
        
        const name = document.getElementById('listName').value;
        const description = document.getElementById('listDescription').value;
        const itemsText = document.getElementById('listItems').value;
        
        const items = itemsText.split('\n')
            .map(line => line.trim())
            .filter(line => line.length > 0)
            .map(line => ({ content: line }));
        
        if (items.length < 2) {
            alert('Please enter at least 2 items to rank');
            return;
        }
        
        try {
            const response = await fetch(`${this.apiUrl}/lists`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, description, items })
            });
            
            const newList = await response.json();
            
            this.hideModal('createListModal');
            document.getElementById('createListForm').reset();
            
            await this.loadLists();
            await this.selectList(newList.list_id);
        } catch (error) {
            console.error('Failed to create list:', error);
            alert('Failed to create list');
        }
    }

    importItems() {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.json';
        
        input.onchange = async (e) => {
            const file = e.target.files[0];
            if (!file) return;
            
            try {
                const text = await file.text();
                const data = JSON.parse(text);
                
                let items = [];
                if (Array.isArray(data)) {
                    items = data.map(item => 
                        typeof item === 'string' ? item : JSON.stringify(item)
                    );
                } else if (data.items) {
                    items = data.items.map(item => 
                        typeof item === 'string' ? item : JSON.stringify(item)
                    );
                }
                
                document.getElementById('listItems').value = items.join('\n');
            } catch (error) {
                alert('Failed to parse JSON file');
            }
        };
        
        input.click();
    }

    showComplete() {
        document.getElementById('cardStack').innerHTML = `
            <div style="text-align: center; background: white; padding: 40px; border-radius: 20px;">
                <h2>Ranking Complete!</h2>
                <p style="margin: 20px 0; color: var(--gray);">
                    You've compared enough items to establish a reliable ranking.
                </p>
                <button class="btn-primary" onclick="eloSwipe.showRankings()">View Final Rankings</button>
            </div>
        `;
    }

    showModal(modalId) {
        document.getElementById(modalId).style.display = 'flex';
    }

    hideModal(modalId) {
        document.getElementById(modalId).style.display = 'none';
    }

    checkForActiveSession() {
        // Check if there's an active session from URL params
        const params = new URLSearchParams(window.location.search);
        const listId = params.get('list');
        
        if (listId) {
            this.selectList(listId);
        }
    }
}

// Initialize app
const eloSwipe = new EloSwipe();