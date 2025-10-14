import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
    initIframeBridgeChild({ appId: 'typing-test-ui' })
}

// Typing Test Game Logic
const API_URL = window.location.hostname === 'localhost' 
    ? `http://localhost:${window.location.port === '3200' ? '9200' : window.location.port.replace('3', '9')}`
    : '/api';

// Game state
let gameState = {
    isPlaying: false,
    currentText: '',
    typedText: '',
    startTime: null,
    timeLeft: 60,
    timer: null,
    errors: 0,
    correctChars: 0,
    totalChars: 0,
    wpm: 0,
    accuracy: 100,
    score: 0,
    combo: 0,
    maxCombo: 0,
    multiplier: 1,
    mode: 'classic',
    difficulty: 'easy',
    // New adaptive features
    errorTracking: {
        charErrors: new Map(),
        wordErrors: new Map(),
        positionErrors: new Map()
    },
    sessionHistory: [],
    adaptiveMode: true,
    userId: 'user_' + Date.now()
};

// Sample texts for different difficulties
const sampleTexts = {
    easy: [
        "The quick brown fox jumps over the lazy dog near the riverbank.",
        "Technology makes our lives easier and more connected every single day.",
        "Learning to type fast is a valuable skill in the digital age.",
        "Practice makes perfect when it comes to improving your typing speed.",
        "The sun sets beautifully over the horizon painting the sky orange."
    ],
    medium: [
        "In the depths of the ancient library, countless volumes of forgotten knowledge await those brave enough to seek them out.",
        "The quantum computer processes information using principles of quantum mechanics, revolutionizing computational capabilities.",
        "Artificial intelligence continues to evolve, transforming industries and reshaping how we interact with technology daily.",
        "The symphony orchestra performed a magnificent rendition of Beethoven's ninth symphony to a captivated audience.",
        "Exploring the vast cosmos reveals the infinite beauty and mystery of our universe's countless galaxies and stars."
    ],
    hard: [
        "The juxtaposition of quantum mechanics and general relativity presents one of physics' most perplexing conundrums, challenging our fundamental understanding of reality.",
        "Cryptocurrency's blockchain technology utilizes cryptographic hashing algorithms to ensure immutable, decentralized transaction verification across distributed networks.",
        "The philosopher's paradoxical assertion regarding epistemological uncertainty fundamentally undermines conventional paradigms of objective truth and knowledge acquisition.",
        "Bioluminescent organisms exhibit fascinating evolutionary adaptations, producing light through complex biochemical reactions involving luciferin and luciferase enzymes.",
        "The Renaissance period's artistic achievements exemplified humanity's capacity for creative expression, technical innovation, and intellectual advancement."
    ]
};

// DOM elements
const elements = {
    textDisplay: document.getElementById('text-content'),
    typingInput: document.getElementById('typing-input'),
    wpmDisplay: document.getElementById('wpm'),
    accuracyDisplay: document.getElementById('accuracy'),
    timerDisplay: document.getElementById('timer'),
    scoreDisplay: document.getElementById('score'),
    comboFill: document.getElementById('combo-fill'),
    comboMultiplier: document.getElementById('combo-multiplier'),
    startBtn: document.getElementById('start-btn'),
    modeBtn: document.getElementById('mode-btn'),
    difficultyBtn: document.getElementById('difficulty-btn'),
    leaderboardBtn: document.getElementById('leaderboard-btn'),
    leaderboardPanel: document.getElementById('leaderboard-panel'),
    leaderboardList: document.getElementById('leaderboard-list'),
    closeLeaderboard: document.getElementById('close-leaderboard'),
    gameOverPanel: document.getElementById('game-over-panel'),
    finalWpm: document.getElementById('final-wpm'),
    finalAccuracy: document.getElementById('final-accuracy'),
    finalScore: document.getElementById('final-score'),
    finalCombo: document.getElementById('final-combo'),
    playerName: document.getElementById('player-name'),
    submitScore: document.getElementById('submit-score'),
    playAgain: document.getElementById('play-again'),
    keyboardOverlay: document.getElementById('keyboard-overlay'),
    keyboardToggle: document.getElementById('keyboard-toggle')
};

// Keyboard functionality
let keyboardVisible = true;
let currentKeyElement = null;

// Initialize game
function init() {
    setupEventListeners();
    loadLeaderboard('today');
}

// Setup event listeners
function setupEventListeners() {
    elements.startBtn.addEventListener('click', startGame);
    elements.typingInput.addEventListener('input', handleTyping);
    elements.typingInput.addEventListener('keydown', handleKeyDown);
    
    elements.modeBtn.addEventListener('click', toggleMode);
    elements.difficultyBtn.addEventListener('click', toggleDifficulty);
    elements.leaderboardBtn.addEventListener('click', showLeaderboard);
    elements.closeLeaderboard.addEventListener('click', hideLeaderboard);
    
    elements.submitScore.addEventListener('click', submitPlayerScore);
    elements.playAgain.addEventListener('click', resetGame);
    
    // Keyboard overlay toggle
    elements.keyboardToggle.addEventListener('click', toggleKeyboard);
    
    // Tab buttons for leaderboard
    document.querySelectorAll('.tab-button').forEach(tab => {
        tab.addEventListener('click', (e) => {
            document.querySelectorAll('.tab-button').forEach(t => t.classList.remove('active'));
            e.target.classList.add('active');
            loadLeaderboard(e.target.dataset.period);
        });
    });
}

// Start game
async function startGame() {
    if (gameState.isPlaying) return;
    
    resetGameState();
    gameState.isPlaying = true;
    gameState.startTime = Date.now();
    
    // Get text - adaptive if enabled, otherwise fallback to static
    if (gameState.adaptiveMode && gameState.sessionHistory.length > 0) {
        try {
            const adaptiveText = await getAdaptiveText();
            if (adaptiveText) {
                gameState.currentText = adaptiveText;
            } else {
                // Fallback to static texts
                const texts = sampleTexts[gameState.difficulty];
                gameState.currentText = texts[Math.floor(Math.random() * texts.length)];
            }
        } catch (error) {
            console.log('Adaptive text generation failed, using static text:', error);
            const texts = sampleTexts[gameState.difficulty];
            gameState.currentText = texts[Math.floor(Math.random() * texts.length)];
        }
    } else {
        // Use static texts for first session or when adaptive mode is disabled
        const texts = sampleTexts[gameState.difficulty];
        gameState.currentText = texts[Math.floor(Math.random() * texts.length)];
    }
    
    // Display text
    displayText();
    
    // Enable input
    elements.typingInput.disabled = false;
    elements.typingInput.value = '';
    elements.typingInput.focus();
    
    // Start timer
    startTimer();
    
    // Initialize keyboard highlighting
    highlightCurrentKey();
    
    // Update button
    elements.startBtn.querySelector('.button-label').textContent = 'GAME IN PROGRESS';
    elements.startBtn.disabled = true;
}

// Display text with proper formatting
function displayText() {
    const typed = gameState.typedText;
    const text = gameState.currentText;
    let html = '';
    
    for (let i = 0; i < text.length; i++) {
        if (i < typed.length) {
            if (typed[i] === text[i]) {
                html += `<span class="typed">${text[i]}</span>`;
            } else {
                html += `<span class="incorrect">${text[i]}</span>`;
            }
        } else if (i === typed.length) {
            html += `<span class="current">${text[i]}</span>`;
        } else {
            html += text[i];
        }
    }
    
    elements.textDisplay.innerHTML = html;
}

// Handle typing input
function handleTyping(e) {
    if (!gameState.isPlaying) return;
    
    const typed = e.target.value;
    const currentLength = typed.length;
    const previousLength = gameState.typedText.length;
    
    // Check if user is typing forward
    if (currentLength > previousLength) {
        const lastChar = typed[currentLength - 1];
        const expectedChar = gameState.currentText[currentLength - 1];
        
        gameState.totalChars++;
        
        if (lastChar === expectedChar) {
            gameState.correctChars++;
            gameState.combo++;
            updateCombo();
            
            // Visual feedback for correct key press
            handleKeyPress(lastChar, true);
        } else {
            gameState.errors++;
            gameState.combo = 0;
            updateCombo();
            
            // Visual feedback for incorrect key press
            handleKeyPress(lastChar, false);
            
            // Track the error for adaptive learning
            const currentWord = getCurrentWord(currentLength - 1);
            trackError(expectedChar, currentLength - 1, currentWord);
        }
    }
    
    gameState.typedText = typed;
    
    // Check if text is complete
    if (typed === gameState.currentText) {
        completeText();
    }
    
    updateStats();
    displayText();
    
    // Update keyboard highlighting for next character
    highlightCurrentKey();
}

// Handle special keys
function handleKeyDown(e) {
    if (e.key === ' ' && !gameState.isPlaying) {
        e.preventDefault();
        startGame();
    }
}

// Complete current text
function completeText() {
    // Bonus points for completing text
    gameState.score += 100 * gameState.multiplier;
    
    // Get new text
    const texts = sampleTexts[gameState.difficulty];
    gameState.currentText = texts[Math.floor(Math.random() * texts.length)];
    gameState.typedText = '';
    elements.typingInput.value = '';
    
    displayText();
}

// Update game statistics
function updateStats() {
    // Calculate WPM
    if (gameState.startTime) {
        const timeElapsed = (Date.now() - gameState.startTime) / 1000 / 60; // in minutes
        const words = gameState.correctChars / 5; // standard word length
        gameState.wpm = Math.round(words / timeElapsed) || 0;
    }
    
    // Calculate accuracy
    if (gameState.totalChars > 0) {
        gameState.accuracy = Math.round((gameState.correctChars / gameState.totalChars) * 100);
    }
    
    // Calculate score
    gameState.score = Math.round(
        (gameState.correctChars * 10) * 
        (gameState.accuracy / 100) * 
        gameState.multiplier
    );
    
    // Update display
    elements.wpmDisplay.textContent = gameState.wpm;
    elements.accuracyDisplay.textContent = gameState.accuracy + '%';
    elements.scoreDisplay.textContent = gameState.score;
}

// Update combo meter
function updateCombo() {
    // Update max combo
    if (gameState.combo > gameState.maxCombo) {
        gameState.maxCombo = gameState.combo;
    }
    
    // Calculate multiplier
    if (gameState.combo >= 50) {
        gameState.multiplier = 5;
    } else if (gameState.combo >= 30) {
        gameState.multiplier = 3;
    } else if (gameState.combo >= 15) {
        gameState.multiplier = 2;
    } else if (gameState.combo >= 5) {
        gameState.multiplier = 1.5;
    } else {
        gameState.multiplier = 1;
    }
    
    // Update combo bar
    const fillPercent = Math.min((gameState.combo / 50) * 100, 100);
    elements.comboFill.style.width = fillPercent + '%';
    elements.comboMultiplier.textContent = 'x' + gameState.multiplier.toFixed(1);
    
    // Add glow effect for high combos
    if (gameState.combo >= 30) {
        elements.comboFill.style.boxShadow = '0 0 20px rgba(255, 0, 255, 0.8)';
    } else if (gameState.combo >= 15) {
        elements.comboFill.style.boxShadow = '0 0 15px rgba(0, 255, 255, 0.6)';
    } else {
        elements.comboFill.style.boxShadow = '0 0 10px rgba(255, 0, 255, 0.5)';
    }
}

// Start game timer
function startTimer() {
    gameState.timeLeft = gameState.mode === 'classic' ? 60 : 120;
    
    gameState.timer = setInterval(() => {
        gameState.timeLeft--;
        elements.timerDisplay.textContent = gameState.timeLeft;
        
        if (gameState.timeLeft <= 10) {
            elements.timerDisplay.style.color = '#ff0000';
            elements.timerDisplay.style.animation = 'flash 0.5s infinite';
        }
        
        if (gameState.timeLeft <= 0) {
            endGame();
        }
    }, 1000);
}

// End game
function endGame() {
    gameState.isPlaying = false;
    clearInterval(gameState.timer);
    
    // Save session data for adaptive learning
    saveSessionData();
    
    // Clear keyboard highlights
    clearKeyboardHighlights();
    
    // Disable input
    elements.typingInput.disabled = true;
    
    // Update button
    elements.startBtn.querySelector('.button-label').textContent = 'START GAME';
    elements.startBtn.disabled = false;
    
    // Show game over panel
    elements.finalWpm.textContent = gameState.wpm;
    elements.finalAccuracy.textContent = gameState.accuracy + '%';
    elements.finalScore.textContent = gameState.score;
    elements.finalCombo.textContent = gameState.maxCombo;
    elements.gameOverPanel.style.display = 'block';
    
    // Reset timer display
    elements.timerDisplay.style.color = '#00ff00';
    elements.timerDisplay.style.animation = 'none';
}

// Reset game
function resetGame() {
    resetGameState();
    clearKeyboardHighlights();
    elements.gameOverPanel.style.display = 'none';
    elements.typingInput.value = '';
    elements.textDisplay.innerHTML = 'Press SPACE or click START GAME to begin...';
}

// Reset game state
function resetGameState() {
    gameState = {
        ...gameState,
        isPlaying: false,
        currentText: '',
        typedText: '',
        startTime: null,
        timeLeft: 60,
        timer: null,
        errors: 0,
        correctChars: 0,
        totalChars: 0,
        wpm: 0,
        accuracy: 100,
        score: 0,
        combo: 0,
        maxCombo: 0,
        multiplier: 1
    };
    
    // Reset displays
    elements.wpmDisplay.textContent = '0';
    elements.accuracyDisplay.textContent = '100%';
    elements.timerDisplay.textContent = '60';
    elements.scoreDisplay.textContent = '0';
    elements.comboFill.style.width = '0%';
    elements.comboMultiplier.textContent = 'x1';
}

// Toggle game mode
function toggleMode() {
    if (gameState.isPlaying) return;
    
    gameState.mode = gameState.mode === 'classic' ? 'endurance' : 'classic';
    elements.modeBtn.querySelector('.button-label').textContent = 
        `MODE: ${gameState.mode.toUpperCase()}`;
}

// Toggle difficulty
function toggleDifficulty() {
    if (gameState.isPlaying) return;
    
    const difficulties = ['easy', 'medium', 'hard'];
    const currentIndex = difficulties.indexOf(gameState.difficulty);
    gameState.difficulty = difficulties[(currentIndex + 1) % 3];
    
    elements.difficultyBtn.querySelector('.button-label').textContent = 
        `LEVEL: ${gameState.difficulty.toUpperCase()}`;
}

// Show leaderboard
function showLeaderboard() {
    elements.leaderboardPanel.style.display = 'block';
    loadLeaderboard('today');
}

// Hide leaderboard
function hideLeaderboard() {
    elements.leaderboardPanel.style.display = 'none';
}

// Load leaderboard from API
async function loadLeaderboard(period) {
    try {
        const response = await fetch(`${API_URL}/api/leaderboard?period=${period}`);
        const data = await response.json();
        
        displayLeaderboard(data.scores || []);
    } catch (error) {
        console.error('Failed to load leaderboard:', error);
        // Display mock data for now
        displayLeaderboard(getMockLeaderboard(period));
    }
}

// Display leaderboard
function displayLeaderboard(scores) {
    if (scores.length === 0) {
        elements.leaderboardList.innerHTML = '<div class="loading-scores">No scores yet. Be the first!</div>';
        return;
    }
    
    let html = '';
    scores.forEach((score, index) => {
        const rank = index + 1;
        const rankEmoji = rank === 1 ? '🥇' : rank === 2 ? '🥈' : rank === 3 ? '🥉' : `#${rank}`;
        
        html += `
            <div class="score-entry">
                <span class="score-rank">${rankEmoji}</span>
                <span class="score-name">${score.name}</span>
                <span class="score-value">${score.score.toLocaleString()}</span>
            </div>
        `;
    });
    
    elements.leaderboardList.innerHTML = html;
}

// Submit player score
async function submitPlayerScore() {
    const name = elements.playerName.value.trim().toUpperCase();
    if (!name) {
        elements.playerName.style.borderColor = '#ff0000';
        return;
    }
    
    const scoreData = {
        name: name,
        score: gameState.score,
        wpm: gameState.wpm,
        accuracy: gameState.accuracy,
        maxCombo: gameState.maxCombo,
        difficulty: gameState.difficulty,
        mode: gameState.mode
    };
    
    try {
        await fetch(`${API_URL}/api/submit-score`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(scoreData)
        });
        
        // Hide name entry and show leaderboard
        elements.gameOverPanel.style.display = 'none';
        showLeaderboard();
    } catch (error) {
        console.error('Failed to submit score:', error);
        // Still hide the panel
        elements.gameOverPanel.style.display = 'none';
    }
}

// Get mock leaderboard data
function getMockLeaderboard(period) {
    const names = ['ACE', 'NEO', 'MAX', 'ZEN', 'KAI', 'REX', 'JET', 'SKY', 'FOX', 'LUX'];
    const scores = [];
    
    for (let i = 0; i < 10; i++) {
        scores.push({
            name: names[i],
            score: Math.floor(Math.random() * 10000) + 5000 - (i * 500),
            wpm: Math.floor(Math.random() * 50) + 50 - (i * 3),
            accuracy: Math.floor(Math.random() * 20) + 80
        });
    }
    
    return scores.sort((a, b) => b.score - a.score);
}

// Helper function to get current word being typed
function getCurrentWord(position) {
    if (!gameState.currentText || position < 0) return '';
    
    // Find word boundaries around the current position
    let start = position;
    let end = position;
    
    // Move start to beginning of word
    while (start > 0 && gameState.currentText[start - 1] !== ' ') {
        start--;
    }
    
    // Move end to end of word
    while (end < gameState.currentText.length && gameState.currentText[end] !== ' ') {
        end++;
    }
    
    return gameState.currentText.slice(start, end).toLowerCase();
}

// Adaptive text generation
async function getAdaptiveText() {
    try {
        // Build request from current error tracking and session history
        const targetWords = Array.from(gameState.errorTracking.wordErrors.entries())
            .sort((a, b) => b[1] - a[1])
            .slice(0, 8)
            .map(([word]) => word);
            
        const problemChars = Array.from(gameState.errorTracking.charErrors.entries())
            .sort((a, b) => b[1] - a[1])
            .slice(0, 5)
            .map(([char]) => char);
            
        const previousMistakes = [
            ...Array.from(gameState.errorTracking.wordErrors.entries()).map(([word, count]) => ({
                word: word,
                errorCount: count
            })),
            ...Array.from(gameState.errorTracking.charErrors.entries()).map(([char, count]) => ({
                char: char,
                errorCount: count,
                position: 'any'
            }))
        ];

        // Determine user level based on recent performance
        const avgWpm = gameState.sessionHistory.length > 0 
            ? gameState.sessionHistory.reduce((sum, s) => sum + s.wpm, 0) / gameState.sessionHistory.length 
            : 0;
        const avgAccuracy = gameState.sessionHistory.length > 0 
            ? gameState.sessionHistory.reduce((sum, s) => sum + s.accuracy, 0) / gameState.sessionHistory.length 
            : 100;
            
        let userLevel = 'beginner';
        if (avgWpm >= 60 && avgAccuracy >= 95) userLevel = 'advanced';
        else if (avgWpm >= 40 && avgAccuracy >= 90) userLevel = 'intermediate';

        const requestData = {
            userId: gameState.userId,
            difficulty: gameState.difficulty,
            targetWords: targetWords,
            problemChars: problemChars,
            userLevel: userLevel,
            textLength: 'medium',
            previousMistakes: previousMistakes
        };

        const response = await fetch(`${API_URL}/api/adaptive-text`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(requestData)
        });

        if (response.ok) {
            const data = await response.json();
            console.log('Generated adaptive text:', data);
            return data.text;
        } else {
            console.warn('Adaptive text API failed, using fallback');
            return null;
        }
    } catch (error) {
        console.error('Error generating adaptive text:', error);
        return null;
    }
}

// Track errors for adaptive learning
function trackError(char, position, currentWord) {
    // Track character errors
    const charCount = gameState.errorTracking.charErrors.get(char) || 0;
    gameState.errorTracking.charErrors.set(char, charCount + 1);
    
    // Track word errors if available
    if (currentWord && currentWord.length > 1) {
        const wordCount = gameState.errorTracking.wordErrors.get(currentWord) || 0;
        gameState.errorTracking.wordErrors.set(currentWord, wordCount + 1);
    }
    
    // Track position-based errors
    const posKey = `${char}_${position % 10}`;
    const posCount = gameState.errorTracking.positionErrors.get(posKey) || 0;
    gameState.errorTracking.positionErrors.set(posKey, posCount + 1);
}

// Save session data for adaptive learning
function saveSessionData() {
    const sessionData = {
        wpm: gameState.wpm,
        accuracy: gameState.accuracy,
        difficulty: gameState.difficulty,
        score: gameState.score,
        maxCombo: gameState.maxCombo,
        timestamp: Date.now(),
        errors: gameState.errors,
        textLength: gameState.currentText.length
    };
    
    gameState.sessionHistory.push(sessionData);
    
    // Keep only last 10 sessions to prevent memory bloat
    if (gameState.sessionHistory.length > 10) {
        gameState.sessionHistory = gameState.sessionHistory.slice(-10);
    }
    
    // Save to localStorage for persistence
    try {
        localStorage.setItem('typingTest_sessionHistory', JSON.stringify(gameState.sessionHistory));
        localStorage.setItem('typingTest_errorTracking', JSON.stringify({
            charErrors: Array.from(gameState.errorTracking.charErrors.entries()),
            wordErrors: Array.from(gameState.errorTracking.wordErrors.entries()),
            positionErrors: Array.from(gameState.errorTracking.positionErrors.entries())
        }));
    } catch (e) {
        console.warn('Could not save session data to localStorage:', e);
    }
}

// Load session data on startup
function loadSessionData() {
    try {
        const sessionHistory = localStorage.getItem('typingTest_sessionHistory');
        if (sessionHistory) {
            gameState.sessionHistory = JSON.parse(sessionHistory);
        }
        
        const errorTracking = localStorage.getItem('typingTest_errorTracking');
        if (errorTracking) {
            const data = JSON.parse(errorTracking);
            gameState.errorTracking.charErrors = new Map(data.charErrors || []);
            gameState.errorTracking.wordErrors = new Map(data.wordErrors || []);
            gameState.errorTracking.positionErrors = new Map(data.positionErrors || []);
        }
    } catch (e) {
        console.warn('Could not load session data from localStorage:', e);
    }
}

// Keyboard visual functions
function toggleKeyboard() {
    keyboardVisible = !keyboardVisible;
    if (keyboardVisible) {
        elements.keyboardOverlay.classList.remove('hidden');
        elements.keyboardToggle.textContent = 'HIDE';
    } else {
        elements.keyboardOverlay.classList.add('hidden');
        elements.keyboardToggle.textContent = 'SHOW';
    }
}

function highlightCurrentKey() {
    // Clear previous highlights
    clearKeyboardHighlights();
    
    if (!gameState.isPlaying || !keyboardVisible) return;
    
    const currentChar = gameState.currentText[gameState.typedText.length];
    if (!currentChar) return;
    
    // Find the key element for the current character
    const keyElement = findKeyElement(currentChar);
    if (keyElement) {
        keyElement.classList.add('current');
        currentKeyElement = keyElement;
        
        // Highlight the corresponding finger
        const finger = keyElement.dataset.finger;
        if (finger) {
            const fingerElement = document.querySelector(`.finger.${finger}`);
            if (fingerElement) {
                fingerElement.classList.add('active');
            }
        }
    }
}

function handleKeyPress(char, isCorrect) {
    if (!keyboardVisible) return;
    
    const keyElement = findKeyElement(char);
    if (keyElement) {
        // Add visual feedback
        keyElement.classList.add('active');
        
        // Add correct/incorrect feedback
        if (isCorrect) {
            keyElement.classList.add('correct');
            setTimeout(() => keyElement.classList.remove('correct'), 300);
        } else {
            keyElement.classList.add('incorrect');
            setTimeout(() => keyElement.classList.remove('incorrect'), 500);
        }
        
        // Remove active state after animation
        setTimeout(() => keyElement.classList.remove('active'), 200);
    }
}

function clearKeyboardHighlights() {
    // Clear all key highlights
    document.querySelectorAll('.key').forEach(key => {
        key.classList.remove('current', 'active', 'correct', 'incorrect');
    });
    
    // Clear finger highlights
    document.querySelectorAll('.finger').forEach(finger => {
        finger.classList.remove('active');
    });
    
    currentKeyElement = null;
}

function findKeyElement(char) {
    // Handle special characters and convert to lowercase
    const keyMap = {
        ' ': ' ',
        'Enter': 'Enter',
        'Backspace': 'Backspace',
        'Tab': 'Tab',
        'Shift': 'Shift',
        'CapsLock': 'CapsLock',
        'Control': 'Control',
        'Alt': 'Alt',
        'Meta': 'Meta'
    };
    
    const searchKey = keyMap[char] || char.toLowerCase();
    
    // Find key by data-key attribute
    return document.querySelector(`.key[data-key="${searchKey}"]`);
}

// Initialize on load
document.addEventListener('DOMContentLoaded', () => {
    loadSessionData();
    init();
});
