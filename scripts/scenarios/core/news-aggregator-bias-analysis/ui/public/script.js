// News Aggregator Bias Analysis - Enhanced Interactive UI
const API_BASE = 'http://localhost:9300';
const N8N_BASE = 'http://localhost:5678';

// State management
let currentArticles = [];
let currentCategory = 'all';
let currentBiasFilter = 0;
let feeds = [];
let factCheckCache = {};

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    initializeUI();
    loadFeeds();
    loadArticles();
    setupEventListeners();
    updateDate();
    startAutoRefresh();
});

function initializeUI() {
    updateDate();
    drawBiasMeter();
    createParticleBackground();
}

function createParticleBackground() {
    // Add subtle animated particles for visual interest
    const particleContainer = document.createElement('div');
    particleContainer.className = 'particle-container';
    document.body.appendChild(particleContainer);
    
    for (let i = 0; i < 20; i++) {
        const particle = document.createElement('div');
        particle.className = 'particle';
        particle.style.left = Math.random() * 100 + '%';
        particle.style.animationDelay = Math.random() * 10 + 's';
        particle.style.animationDuration = (15 + Math.random() * 10) + 's';
        particleContainer.appendChild(particle);
    }
}

function updateDate() {
    const dateElement = document.getElementById('currentDate');
    const options = { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' };
    dateElement.textContent = new Date().toLocaleDateString('en-US', options);
}

function setupEventListeners() {
    // Category filter buttons
    document.querySelectorAll('.category-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('.category-btn').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
            currentCategory = e.target.dataset.category;
            filterArticles();
        });
    });

    // Bias slider
    const biasSlider = document.getElementById('biasSlider');
    biasSlider.addEventListener('input', (e) => {
        currentBiasFilter = parseInt(e.target.value);
        updateBiasIndicator();
        filterArticles();
    });

    // Refresh button
    document.getElementById('refreshBtn').addEventListener('click', () => {
        refreshFeeds();
    });

    // Add feed button
    document.getElementById('addFeedBtn').addEventListener('click', () => {
        showModal('addFeedModal');
    });

    // Modal close buttons
    document.querySelectorAll('.close-modal').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.target.closest('.modal').classList.remove('active');
        });
    });

    // Add feed form
    document.getElementById('addFeedForm').addEventListener('submit', (e) => {
        e.preventDefault();
        addNewFeed();
    });

    // Close modal on outside click
    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
            }
        });
    });
}

function updateBiasIndicator() {
    const indicator = document.getElementById('biasIndicator');
    let text = 'BALANCED';
    let color = '#6366f1';
    
    if (currentBiasFilter < -50) {
        text = 'FAR LEFT';
        color = '#3b82f6';
    } else if (currentBiasFilter < -20) {
        text = 'LEFT';
        color = '#60a5fa';
    } else if (currentBiasFilter > 50) {
        text = 'FAR RIGHT';
        color = '#dc2626';
    } else if (currentBiasFilter > 20) {
        text = 'RIGHT';
        color = '#f87171';
    }
    
    indicator.textContent = text;
    indicator.style.background = `linear-gradient(135deg, ${color}, ${color}99)`;
}

async function loadArticles() {
    const grid = document.getElementById('newsGrid');
    grid.innerHTML = '<div class="loading-spinner"><div class="spinner"></div><p>Loading fresh perspectives...</p></div>';
    
    try {
        // Simulate loading articles (in production, this would call the API)
        const mockArticles = generateMockArticles();
        currentArticles = mockArticles;
        
        // Analyze bias for each article
        for (const article of currentArticles) {
            article.bias = await analyzeArticleBias(article);
        }
        
        displayArticles();
        updateBiasStats();
        updateTrendingTopics();
    } catch (error) {
        console.error('Error loading articles:', error);
        grid.innerHTML = '<p class="error">Failed to load articles. Please try again.</p>';
    }
}

function generateMockArticles() {
    const sources = ['Global Times', 'Tech Review', 'Economic Daily', 'Science Weekly', 'Political Observer'];
    const categories = ['world', 'tech', 'economics', 'politics', 'general'];
    const biases = [-80, -40, 0, 40, 80];
    
    return Array.from({ length: 12 }, (_, i) => ({
        id: `article_${Date.now()}_${i}`,
        title: `Breaking: Major Development in ${categories[i % categories.length]} Sector`,
        summary: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. Recent studies show significant changes in global markets with implications for policy makers worldwide.',
        source: sources[i % sources.length],
        category: categories[i % categories.length],
        url: `https://example.com/article${i}`,
        published_date: new Date(Date.now() - Math.random() * 86400000 * 7).toISOString(),
        bias_score: biases[i % biases.length] + (Math.random() * 20 - 10),
        thumbnail: `data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" width="400" height="200"><rect fill="%23${Math.floor(Math.random()*16777215).toString(16)}" width="400" height="200"/></svg>`
    }));
}

async function analyzeArticleBias(article) {
    // Check cache first
    if (factCheckCache[article.id]) {
        return factCheckCache[article.id];
    }
    
    try {
        const response = await fetch(`${N8N_BASE}/webhook/bias-analyzer`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(article)
        });
        
        if (response.ok) {
            const result = await response.json();
            factCheckCache[article.id] = result;
            return result;
        }
    } catch (error) {
        console.error('Bias analysis error:', error);
    }
    
    // Return default if analysis fails
    return {
        bias_score: article.bias_score || 0,
        bias_rating: 'center',
        perspective_count: 1
    };
}

async function factCheckArticle(article) {
    try {
        const response = await fetch(`${N8N_BASE}/webhook/fact-checker`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                article_id: article.id,
                title: article.title,
                summary: article.summary,
                source: article.source,
                url: article.url
            })
        });
        
        if (response.ok) {
            return await response.json();
        }
    } catch (error) {
        console.error('Fact check error:', error);
    }
    
    return null;
}

function displayArticles() {
    const grid = document.getElementById('newsGrid');
    const filtered = filterArticlesByBias();
    
    if (filtered.length === 0) {
        grid.innerHTML = '<p class="no-articles">No articles match your filters. Try adjusting the bias slider.</p>';
        return;
    }
    
    grid.innerHTML = filtered.map(article => createArticleCard(article)).join('');
    
    // Add click handlers for article cards
    document.querySelectorAll('.article-card').forEach(card => {
        card.addEventListener('click', () => openArticleModal(card.dataset.articleId));
    });
    
    // Add fact-check button handlers
    document.querySelectorAll('.fact-check-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            e.stopPropagation();
            await handleFactCheck(btn.dataset.articleId);
        });
    });
}

function createArticleCard(article) {
    const biasColor = getBiasColor(article.bias_score || 0);
    const date = new Date(article.published_date).toLocaleDateString();
    
    return `
        <div class="article-card" data-article-id="${article.id}">
            <div class="bias-indicator-bar" style="background: linear-gradient(90deg, ${biasColor}, transparent)"></div>
            <div class="article-header">
                <span class="article-source">${article.source}</span>
                <span class="article-date">${date}</span>
            </div>
            <h3 class="article-title">${article.title}</h3>
            <p class="article-summary">${article.summary}</p>
            <div class="article-footer">
                <span class="bias-badge" style="background: ${biasColor}">
                    ${getBiasLabel(article.bias_score || 0)}
                </span>
                <button class="fact-check-btn" data-article-id="${article.id}">
                    🔍 Fact Check
                </button>
            </div>
        </div>
    `;
}

function getBiasColor(score) {
    if (score < -50) return '#3b82f6';
    if (score < -20) return '#60a5fa';
    if (score < 20) return '#6366f1';
    if (score < 50) return '#f87171';
    return '#dc2626';
}

function getBiasLabel(score) {
    if (score < -66) return 'Far Left';
    if (score < -33) return 'Left';
    if (score < -10) return 'Center-Left';
    if (score < 10) return 'Center';
    if (score < 33) return 'Center-Right';
    if (score < 66) return 'Right';
    return 'Far Right';
}

function filterArticlesByBias() {
    let filtered = currentArticles;
    
    // Filter by category
    if (currentCategory !== 'all') {
        filtered = filtered.filter(a => a.category === currentCategory);
    }
    
    // Filter by bias
    if (currentBiasFilter !== 0) {
        const tolerance = 30;
        filtered = filtered.filter(a => {
            const bias = a.bias_score || 0;
            return Math.abs(bias - currentBiasFilter) <= tolerance;
        });
    }
    
    return filtered;
}

function filterArticles() {
    displayArticles();
}

async function handleFactCheck(articleId) {
    const article = currentArticles.find(a => a.id === articleId);
    if (!article) return;
    
    const btn = document.querySelector(`.fact-check-btn[data-article-id="${articleId}"]`);
    btn.innerHTML = '⏳ Checking...';
    btn.disabled = true;
    
    const result = await factCheckArticle(article);
    
    if (result) {
        article.factCheck = result;
        showFactCheckModal(article, result);
    }
    
    btn.innerHTML = '✓ Checked';
    btn.style.background = '#10b981';
}

function showFactCheckModal(article, factCheck) {
    const modalBody = document.getElementById('modalBody');
    
    const credibilityColor = factCheck.credibility_score > 70 ? '#10b981' : 
                            factCheck.credibility_score > 40 ? '#f59e0b' : '#ef4444';
    
    modalBody.innerHTML = `
        <div class="fact-check-modal">
            <h2>${article.title}</h2>
            <div class="credibility-meter">
                <div class="credibility-score" style="background: ${credibilityColor}">
                    <span class="score-number">${factCheck.credibility_score}</span>
                    <span class="score-label">${factCheck.credibility_rating}</span>
                </div>
            </div>
            <div class="verification-summary">
                <h3>Verification Summary</h3>
                <div class="verification-stats">
                    <div class="stat verified">✓ Verified: ${factCheck.verification_summary.verified}</div>
                    <div class="stat disputed">✗ Disputed: ${factCheck.verification_summary.disputed}</div>
                    <div class="stat unverified">? Unverified: ${factCheck.verification_summary.unverified}</div>
                </div>
            </div>
            <div class="fact-check-details">
                <h3>Claim Analysis</h3>
                ${factCheck.fact_check_results.map(result => `
                    <div class="claim-result ${result.status}">
                        <p class="claim">"${result.claim}"</p>
                        <span class="status-badge">${result.status}</span>
                        <p class="evidence">${result.evidence}</p>
                    </div>
                `).join('')}
            </div>
            <p class="sources-note">Sources consulted: ${factCheck.sources_consulted}</p>
        </div>
    `;
    
    showModal('articleModal');
}

function openArticleModal(articleId) {
    const article = currentArticles.find(a => a.id === articleId);
    if (!article) return;
    
    const modalBody = document.getElementById('modalBody');
    modalBody.innerHTML = `
        <div class="article-modal">
            <h2>${article.title}</h2>
            <div class="article-meta">
                <span>Source: ${article.source}</span>
                <span>Published: ${new Date(article.published_date).toLocaleDateString()}</span>
            </div>
            <p class="article-content">${article.summary}</p>
            <div class="bias-analysis">
                <h3>Bias Analysis</h3>
                <div class="bias-score-display">
                    <div class="bias-gauge">
                        <div class="bias-pointer" style="left: ${50 + article.bias_score / 2}%"></div>
                    </div>
                    <p>Bias Score: ${article.bias_score}</p>
                    <p>Rating: ${getBiasLabel(article.bias_score)}</p>
                </div>
            </div>
            <div class="article-actions">
                <a href="${article.url}" target="_blank" class="read-full-btn">Read Full Article</a>
                <button onclick="handleFactCheck('${article.id}')" class="fact-check-modal-btn">Run Fact Check</button>
            </div>
        </div>
    `;
    
    showModal('articleModal');
}

function showModal(modalId) {
    document.getElementById(modalId).classList.add('active');
}

function loadFeeds() {
    // Load available news feeds
    const feedList = document.getElementById('feedList');
    
    const defaultFeeds = [
        { name: 'BBC News', bias: 'center', active: true },
        { name: 'Fox News', bias: 'right', active: true },
        { name: 'CNN', bias: 'left', active: true },
        { name: 'Reuters', bias: 'center', active: true },
        { name: 'The Guardian', bias: 'left', active: false }
    ];
    
    feeds = defaultFeeds;
    
    feedList.innerHTML = feeds.map(feed => `
        <div class="feed-item ${feed.active ? 'active' : ''}">
            <span class="feed-name">${feed.name}</span>
            <span class="feed-bias ${feed.bias}">${feed.bias}</span>
            <button class="toggle-feed" data-feed="${feed.name}">
                ${feed.active ? '✓' : '○'}
            </button>
        </div>
    `).join('');
    
    // Add toggle handlers
    document.querySelectorAll('.toggle-feed').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const feedName = e.target.dataset.feed;
            const feed = feeds.find(f => f.name === feedName);
            if (feed) {
                feed.active = !feed.active;
                loadFeeds();
                loadArticles();
            }
        });
    });
}

function addNewFeed() {
    const name = document.getElementById('feedName').value;
    const url = document.getElementById('feedUrl').value;
    const category = document.getElementById('feedCategory').value;
    const bias = document.getElementById('feedBias').value;
    
    feeds.push({ name, url, category, bias, active: true });
    loadFeeds();
    
    // Reset form and close modal
    document.getElementById('addFeedForm').reset();
    document.getElementById('addFeedModal').classList.remove('active');
    
    // Refresh articles
    loadArticles();
}

function refreshFeeds() {
    const btn = document.getElementById('refreshBtn');
    btn.classList.add('spinning');
    
    loadArticles().then(() => {
        setTimeout(() => {
            btn.classList.remove('spinning');
        }, 1000);
    });
}

function updateBiasStats() {
    const stats = {
        left: 0,
        center: 0,
        right: 0
    };
    
    currentArticles.forEach(article => {
        const bias = article.bias_score || 0;
        if (bias < -20) stats.left++;
        else if (bias > 20) stats.right++;
        else stats.center++;
    });
    
    document.getElementById('leftCount').textContent = stats.left;
    document.getElementById('centerCount').textContent = stats.center;
    document.getElementById('rightCount').textContent = stats.right;
    
    // Update perspectives banner
    const totalPerspectives = currentArticles.reduce((sum, a) => 
        sum + (a.bias?.perspective_count || 1), 0);
    
    document.getElementById('perspectiveCount').textContent = 
        `${totalPerspectives} unique perspectives across ${currentArticles.length} articles`;
}

function updateTrendingTopics() {
    const topics = [
        { name: 'Climate Summit', count: 24, trend: 'up' },
        { name: 'Tech Regulation', count: 18, trend: 'up' },
        { name: 'Economic Policy', count: 15, trend: 'down' },
        { name: 'Healthcare Reform', count: 12, trend: 'stable' },
        { name: 'Space Exploration', count: 9, trend: 'up' }
    ];
    
    const trendingList = document.getElementById('trendingList');
    trendingList.innerHTML = topics.map(topic => `
        <div class="trending-item">
            <span class="topic-name">${topic.name}</span>
            <span class="topic-count">${topic.count}</span>
            <span class="trend-indicator ${topic.trend}">${
                topic.trend === 'up' ? '↑' : topic.trend === 'down' ? '↓' : '→'
            }</span>
        </div>
    `).join('');
}

function drawBiasMeter() {
    const canvas = document.getElementById('biasMeter');
    const ctx = canvas.getContext('2d');
    const width = canvas.width;
    const height = canvas.height;
    
    // Clear canvas
    ctx.clearRect(0, 0, width, height);
    
    // Draw gradient background
    const gradient = ctx.createLinearGradient(0, 0, width, 0);
    gradient.addColorStop(0, '#3b82f6');
    gradient.addColorStop(0.5, '#6366f1');
    gradient.addColorStop(1, '#dc2626');
    
    ctx.fillStyle = gradient;
    ctx.fillRect(0, height - 20, width, 20);
    
    // Draw labels
    ctx.fillStyle = '#666';
    ctx.font = '10px sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('LEFT', 30, height - 25);
    ctx.fillText('CENTER', width / 2, height - 25);
    ctx.fillText('RIGHT', width - 30, height - 25);
    
    // Animate meter needle
    animateBiasMeter(ctx, width, height);
}

function animateBiasMeter(ctx, width, height) {
    let position = width / 2;
    const targetPosition = width / 2 + (currentBiasFilter / 100) * (width / 2);
    
    function animate() {
        ctx.clearRect(0, 0, width, height - 25);
        
        // Draw needle
        ctx.beginPath();
        ctx.moveTo(position - 5, height - 30);
        ctx.lineTo(position + 5, height - 30);
        ctx.lineTo(position, height - 20);
        ctx.closePath();
        ctx.fillStyle = '#333';
        ctx.fill();
        
        // Move towards target
        if (Math.abs(position - targetPosition) > 1) {
            position += (targetPosition - position) * 0.1;
            requestAnimationFrame(animate);
        }
    }
    
    animate();
}

function startAutoRefresh() {
    // Auto-refresh every 5 minutes
    setInterval(() => {
        loadArticles();
    }, 5 * 60 * 1000);
}