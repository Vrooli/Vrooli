// Token Economy - Wallet Application

import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
    initIframeBridgeChild({ appId: 'token-economy-ui' })
}

const API_PATH = '/api/v1';
const DEFAULT_API_PORT = 11080;

const API_URL = resolveApiUrl();

function resolveApiUrl() {
    if (typeof window === 'undefined') {
        return `http://127.0.0.1:${DEFAULT_API_PORT}${API_PATH}`;
    }

    const proxyBase = resolveProxyBase();
    if (proxyBase) {
        return ensureApiPath(proxyBase);
    }

    const hostname = window.location?.hostname || '';
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
        return `http://localhost:${DEFAULT_API_PORT}${API_PATH}`;
    }

    const origin = window.location?.origin;
    if (origin) {
        return ensureApiPath(origin);
    }

    return API_PATH;
}

function resolveProxyBase() {
    if (typeof window === 'undefined') {
        return undefined;
    }

    const info = window.__APP_MONITOR_PROXY_INFO__;
    if (!info || typeof info !== 'object') {
        return undefined;
    }

    const candidates = [];

    const pushCandidate = (candidate) => {
        if (!candidate) {
            return;
        }
        if (typeof candidate === 'string') {
            const trimmed = candidate.trim();
            if (trimmed) {
                candidates.push(trimmed);
            }
            return;
        }
        if (typeof candidate === 'object') {
            pushCandidate(candidate.apiBase);
            pushCandidate(candidate.url);
            pushCandidate(candidate.target);
            pushCandidate(candidate.path);
        }
    };

    pushCandidate(info.apiBase);
    pushCandidate(info.apiUrl);
    pushCandidate(info.primary);

    const listFields = ['ports', 'endpoints', 'services'];
    listFields.forEach((field) => {
        const value = info[field];
        if (Array.isArray(value)) {
            value.forEach(pushCandidate);
        }
    });

    for (const candidate of candidates) {
        const normalized = normalizeProxyCandidate(candidate);
        if (normalized) {
            return normalized;
        }
    }

    return undefined;
}

function normalizeProxyCandidate(candidate) {
    if (typeof candidate !== 'string') {
        return undefined;
    }

    const trimmed = candidate.trim();
    if (!trimmed) {
        return undefined;
    }

    if (/^https?:\/\//i.test(trimmed)) {
        return stripTrailingSlash(trimmed);
    }

    if (trimmed.startsWith('//')) {
        const protocol = window.location?.protocol || 'https:';
        return stripTrailingSlash(`${protocol}${trimmed}`);
    }

    if (trimmed.startsWith('/')) {
        const origin = window.location?.origin;
        if (!origin) {
            return stripTrailingSlash(trimmed);
        }
        return stripTrailingSlash(`${origin}${trimmed}`);
    }

    if (/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i.test(trimmed)) {
        return stripTrailingSlash(`http://${trimmed}`);
    }

    if (typeof window !== 'undefined' && window.location?.origin) {
        return stripTrailingSlash(`${window.location.origin}/${trimmed.replace(/^\/+/, '')}`);
    }

    return stripTrailingSlash(`/${trimmed.replace(/^\/+/, '')}`);
}

function ensureApiPath(base) {
    const sanitizedBase = stripTrailingSlash(base);
    if (sanitizedBase.endsWith(API_PATH)) {
        return sanitizedBase;
    }
    if (sanitizedBase.endsWith('/api')) {
        return `${sanitizedBase}${API_PATH.slice(4)}`;
    }
    return `${sanitizedBase}${API_PATH}`;
}

function stripTrailingSlash(value) {
    if (typeof value !== 'string') {
        return value;
    }
    return value.replace(/\/+$/, '');
}

// State management
const state = {
    currentUser: null,
    currentWallet: null,
    tokens: [],
    balances: [],
    transactions: [],
    achievements: [],
    exchangeRates: {},
    isAdmin: false
};

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
    setupEventListeners();
    loadUserData();
});

function initializeApp() {
    // Check if user is admin
    const isAdmin = localStorage.getItem('isAdmin') === 'true';
    if (isAdmin) {
        document.body.classList.add('is-admin');
        state.isAdmin = true;
    }
    
    // Set current user
    const userId = localStorage.getItem('userId') || 'user-' + Math.random().toString(36).substr(2, 9);
    localStorage.setItem('userId', userId);
    state.currentUser = userId;
    
    // Update UI with user info
    const userName = localStorage.getItem('userName') || 'User';
    document.getElementById('userName').textContent = userName;
    document.getElementById('userInitial').textContent = userName.charAt(0).toUpperCase();
}

function setupEventListeners() {
    // Navigation tabs
    document.querySelectorAll('.nav-tab').forEach(tab => {
        tab.addEventListener('click', (e) => {
            switchTab(e.target.dataset.tab);
        });
    });
    
    // Quick actions
    document.getElementById('sendButton').addEventListener('click', () => showModal('sendModal'));
    document.getElementById('receiveButton').addEventListener('click', () => showReceiveModal());
    document.getElementById('swapButton').addEventListener('click', () => switchTab('exchange'));
    
    // Refresh balance
    document.getElementById('refreshBalance').addEventListener('click', loadBalances);
    
    // Modal close buttons
    document.querySelectorAll('[data-close]').forEach(btn => {
        btn.addEventListener('click', (e) => {
            closeModal(e.target.dataset.close);
        });
    });
    
    // Send modal
    document.getElementById('confirmSend').addEventListener('click', handleSend);
    document.getElementById('sendToken').addEventListener('change', updateSendAvailable);
    
    // Copy address
    document.getElementById('copyAddress').addEventListener('click', copyAddress);
    
    // Exchange
    document.getElementById('fromAmount').addEventListener('input', calculateExchange);
    document.getElementById('fromToken').addEventListener('change', updateExchangeBalances);
    document.getElementById('toToken').addEventListener('change', updateExchangeBalances);
    document.getElementById('swapDirection').addEventListener('click', swapDirection);
    document.getElementById('executeSwap').addEventListener('click', handleSwap);
    
    // Admin functions
    if (state.isAdmin) {
        document.getElementById('createToken').addEventListener('click', handleCreateToken);
        document.getElementById('setRate').addEventListener('click', handleSetRate);
        document.getElementById('setLimits').addEventListener('click', handleSetLimits);
    }
}

// Tab switching
function switchTab(tabName) {
    // Update nav tabs
    document.querySelectorAll('.nav-tab').forEach(tab => {
        tab.classList.toggle('active', tab.dataset.tab === tabName);
    });
    
    // Update content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === `${tabName}-tab`);
    });
    
    // Load data for the tab
    switch(tabName) {
        case 'wallet':
            loadBalances();
            break;
        case 'transactions':
            loadTransactions();
            break;
        case 'achievements':
            loadAchievements();
            break;
        case 'exchange':
            loadExchangeData();
            break;
        case 'admin':
            loadAdminData();
            break;
    }
}

// Data loading functions
async function loadUserData() {
    await createOrGetWallet();
    await loadBalances();
    await loadTokens();
}

async function createOrGetWallet() {
    try {
        // Check if wallet exists in localStorage
        let walletAddress = localStorage.getItem('walletAddress');
        
        if (!walletAddress) {
            // Create new wallet
            const response = await fetch(`${API_URL}/wallets/create`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    type: 'user',
                    user_id: state.currentUser
                })
            });
            
            const data = await response.json();
            walletAddress = data.address;
            localStorage.setItem('walletAddress', walletAddress);
            localStorage.setItem('walletId', data.wallet_id);
        }
        
        state.currentWallet = walletAddress;
        document.getElementById('walletAddress').value = walletAddress;
        
    } catch (error) {
        console.error('Failed to create/get wallet:', error);
        showToast('Failed to initialize wallet', 'error');
    }
}

async function loadBalances() {
    if (!state.currentWallet) return;
    
    try {
        const response = await fetch(`${API_URL}/wallets/${state.currentWallet}/balance`);
        const data = await response.json();
        
        state.balances = data.balances || [];
        updateBalanceDisplay();
        updateTokenList();
        
    } catch (error) {
        console.error('Failed to load balances:', error);
        showToast('Failed to load balances', 'error');
    }
}

async function loadTokens() {
    try {
        const response = await fetch(`${API_URL}/tokens`);
        const data = await response.json();
        
        state.tokens = data || [];
        updateTokenSelects();
        
    } catch (error) {
        console.error('Failed to load tokens:', error);
    }
}

async function loadTransactions() {
    try {
        const response = await fetch(`${API_URL}/transactions?wallet=${state.currentWallet}&limit=50`);
        const data = await response.json();
        
        state.transactions = data || [];
        updateTransactionList();
        
    } catch (error) {
        console.error('Failed to load transactions:', error);
        showToast('Failed to load transactions', 'error');
    }
}

async function loadAchievements() {
    try {
        const response = await fetch(`${API_URL}/achievements/${state.currentUser}`);
        const data = await response.json();
        
        state.achievements = data.achievements || [];
        updateAchievementDisplay();
        
    } catch (error) {
        console.error('Failed to load achievements:', error);
        showToast('Failed to load achievements', 'error');
    }
}

async function loadExchangeData() {
    await loadTokens();
    updateExchangeSelects();
}

async function loadAdminData() {
    if (!state.isAdmin) return;
    
    try {
        const response = await fetch(`${API_URL}/admin/analytics`);
        const data = await response.json();
        
        document.getElementById('totalTransactions').textContent = data.total_transactions || 0;
        document.getElementById('activeWallets').textContent = data.active_wallets || 0;
        document.getElementById('totalVolume').textContent = formatAmount(data.total_volume || 0);
        
    } catch (error) {
        console.error('Failed to load admin data:', error);
    }
}

// UI update functions
function updateBalanceDisplay() {
    const totalBalance = state.balances.reduce((sum, b) => sum + b.amount, 0);
    document.getElementById('totalBalance').textContent = formatAmount(totalBalance);
}

function updateTokenList() {
    const container = document.getElementById('tokenList');
    container.innerHTML = '';
    
    if (state.balances.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: var(--gray-500);">No tokens yet</p>';
        return;
    }
    
    state.balances.forEach(balance => {
        const item = document.createElement('div');
        item.className = 'token-item';
        item.innerHTML = `
            <div class="token-info">
                <div class="token-icon">${balance.symbol ? balance.symbol.substring(0, 2) : '??'}</div>
                <div class="token-details">
                    <span class="token-symbol">${balance.symbol || 'Unknown'}</span>
                    <span class="token-name">Token</span>
                </div>
            </div>
            <div class="token-balance">
                <div class="token-amount">${formatAmount(balance.amount)}</div>
                <div class="token-value">$${formatAmount(balance.value_usd || 0)}</div>
            </div>
        `;
        container.appendChild(item);
    });
}

function updateTransactionList() {
    const container = document.getElementById('transactionList');
    container.innerHTML = '';
    
    if (state.transactions.length === 0) {
        container.innerHTML = '<p style="text-align: center; color: var(--gray-500);">No transactions yet</p>';
        return;
    }
    
    state.transactions.forEach(tx => {
        const item = document.createElement('div');
        item.className = 'transaction-item';
        
        const isIncoming = tx.to_wallet === state.currentWallet;
        const amount = isIncoming ? `+${formatAmount(tx.amount)}` : `-${formatAmount(tx.amount)}`;
        const amountClass = isIncoming ? 'positive' : 'negative';
        
        item.innerHTML = `
            <div class="transaction-info">
                <div class="transaction-icon ${tx.type}">
                    ${getTransactionIcon(tx.type)}
                </div>
                <div class="transaction-details">
                    <span class="transaction-type">${capitalizeFirst(tx.type)}</span>
                    <span class="transaction-time">${formatDate(tx.created_at)}</span>
                </div>
            </div>
            <div class="transaction-amount ${amountClass}">${amount}</div>
        `;
        container.appendChild(item);
    });
}

function updateAchievementDisplay() {
    const grid = document.getElementById('achievementGrid');
    grid.innerHTML = '';
    
    if (state.achievements.length === 0) {
        grid.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: var(--gray-500);">No achievements yet</p>';
        return;
    }
    
    // Update stats
    document.getElementById('totalAchievements').textContent = state.achievements.length;
    document.getElementById('rareAchievements').textContent = state.achievements.filter(a => 
        ['rare', 'epic', 'legendary'].includes(a.rarity)).length;
    
    state.achievements.forEach(achievement => {
        const card = document.createElement('div');
        card.className = `achievement-card ${achievement.rarity}`;
        card.innerHTML = `
            <div class="achievement-icon">🏆</div>
            <div class="achievement-title">${achievement.title}</div>
            <div class="achievement-scenario">${achievement.scenario}</div>
        `;
        grid.appendChild(card);
    });
}

function updateTokenSelects() {
    const selects = ['sendToken', 'fromToken', 'toToken', 'rateFromToken', 'rateToToken'];
    
    selects.forEach(selectId => {
        const select = document.getElementById(selectId);
        if (!select) return;
        
        const currentValue = select.value;
        select.innerHTML = '<option value="">Select token</option>';
        
        state.tokens.forEach(token => {
            const option = document.createElement('option');
            option.value = token.id;
            option.textContent = `${token.symbol} - ${token.name}`;
            select.appendChild(option);
        });
        
        select.value = currentValue;
    });
}

function updateExchangeSelects() {
    updateTokenSelects();
}

function updateExchangeBalances() {
    const fromToken = document.getElementById('fromToken').value;
    const toToken = document.getElementById('toToken').value;
    
    const fromBalance = state.balances.find(b => b.token_id === fromToken);
    const toBalance = state.balances.find(b => b.token_id === toToken);
    
    document.getElementById('fromBalance').textContent = formatAmount(fromBalance?.amount || 0);
    document.getElementById('toBalance').textContent = formatAmount(toBalance?.amount || 0);
    
    calculateExchange();
}

function updateSendAvailable() {
    const tokenId = document.getElementById('sendToken').value;
    const balance = state.balances.find(b => b.token_id === tokenId);
    document.getElementById('sendAvailable').textContent = formatAmount(balance?.amount || 0);
}

// Transaction handlers
async function handleSend() {
    const recipient = document.getElementById('sendRecipient').value;
    const tokenId = document.getElementById('sendToken').value;
    const amount = parseFloat(document.getElementById('sendAmount').value);
    const memo = document.getElementById('sendMemo').value;
    
    if (!recipient || !tokenId || !amount) {
        showToast('Please fill in all required fields', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/tokens/transfer`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                from_wallet: state.currentWallet,
                to_wallet: recipient,
                token_id: tokenId,
                amount: amount,
                memo: memo
            })
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Transfer failed');
        }
        
        const data = await response.json();
        showToast('Transfer successful!', 'success');
        closeModal('sendModal');
        
        // Reset form
        document.getElementById('sendRecipient').value = '';
        document.getElementById('sendAmount').value = '';
        document.getElementById('sendMemo').value = '';
        
        // Reload balances
        loadBalances();
        loadTransactions();
        
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function handleSwap() {
    const fromToken = document.getElementById('fromToken').value;
    const toToken = document.getElementById('toToken').value;
    const amount = parseFloat(document.getElementById('fromAmount').value);
    
    if (!fromToken || !toToken || !amount) {
        showToast('Please fill in all fields', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/tokens/swap`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                from_token: fromToken,
                to_token: toToken,
                amount: amount
            })
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Swap failed');
        }
        
        const data = await response.json();
        showToast(`Swapped successfully! Received ${data.received_amount} tokens`, 'success');
        
        // Reset form
        document.getElementById('fromAmount').value = '';
        document.getElementById('toAmount').value = '';
        
        // Reload balances
        loadBalances();
        loadTransactions();
        
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function handleCreateToken() {
    const symbol = document.getElementById('newTokenSymbol').value;
    const name = document.getElementById('newTokenName').value;
    const type = document.getElementById('newTokenType').value;
    
    if (!symbol || !name || !type) {
        showToast('Please fill in all fields', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/tokens/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ symbol, name, type })
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to create token');
        }
        
        const data = await response.json();
        showToast('Token created successfully!', 'success');
        
        // Reset form
        document.getElementById('newTokenSymbol').value = '';
        document.getElementById('newTokenName').value = '';
        
        // Reload tokens
        loadTokens();
        
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function handleSetRate() {
    // TODO: Implement when admin endpoint is ready
    showToast('Exchange rate setting not yet implemented', 'info');
}

async function handleSetLimits() {
    // TODO: Implement when admin endpoint is ready
    showToast('User limits setting not yet implemented', 'info');
}

// Helper functions
function calculateExchange() {
    const fromAmount = parseFloat(document.getElementById('fromAmount').value) || 0;
    const fromToken = document.getElementById('fromToken').value;
    const toToken = document.getElementById('toToken').value;
    
    if (!fromToken || !toToken || fromAmount === 0) {
        document.getElementById('toAmount').value = '';
        document.getElementById('exchangeRate').textContent = '-';
        document.getElementById('executeSwap').disabled = true;
        return;
    }
    
    // For now, use a simple 1:1 rate (would fetch from API in production)
    const rate = 1;
    const toAmount = fromAmount * rate;
    
    document.getElementById('toAmount').value = toAmount.toFixed(2);
    document.getElementById('exchangeRate').textContent = `1 ${fromToken} = ${rate} ${toToken}`;
    document.getElementById('executeSwap').disabled = false;
}

function swapDirection() {
    const fromToken = document.getElementById('fromToken').value;
    const toToken = document.getElementById('toToken').value;
    const fromAmount = document.getElementById('fromAmount').value;
    const toAmount = document.getElementById('toAmount').value;
    
    document.getElementById('fromToken').value = toToken;
    document.getElementById('toToken').value = fromToken;
    document.getElementById('fromAmount').value = toAmount;
    
    calculateExchange();
}

function showReceiveModal() {
    showModal('receiveModal');
    generateQRCode(state.currentWallet);
}

function generateQRCode(address) {
    const canvas = document.getElementById('qrCode');
    const ctx = canvas.getContext('2d');
    
    // Simple placeholder - would use QR library in production
    ctx.fillStyle = '#000';
    ctx.fillRect(0, 0, 200, 200);
    ctx.fillStyle = '#fff';
    ctx.font = '12px monospace';
    ctx.fillText('QR Code', 75, 100);
}

function copyAddress() {
    const input = document.getElementById('walletAddress');
    input.select();
    document.execCommand('copy');
    showToast('Address copied to clipboard!', 'success');
}

// Modal functions
function showModal(modalId) {
    document.getElementById(modalId).classList.add('active');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

// Toast notifications
function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.innerHTML = `<span class="toast-message">${message}</span>`;
    
    container.appendChild(toast);
    
    setTimeout(() => {
        toast.style.animation = 'fadeOut 0.3s';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// Utility functions
function formatAmount(amount) {
    return new Intl.NumberFormat('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
    }).format(amount);
}

function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diff = (now - date) / 1000; // seconds
    
    if (diff < 60) return 'Just now';
    if (diff < 3600) return `${Math.floor(diff / 60)} min ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)} hours ago`;
    
    return date.toLocaleDateString();
}

function capitalizeFirst(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

function getTransactionIcon(type) {
    const icons = {
        mint: '⬇',
        transfer: '➡',
        burn: '🔥',
        swap: '🔄'
    };
    return icons[type] || '•';
}

// Click outside to close modals
window.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) {
        e.target.classList.remove('active');
    }
});
