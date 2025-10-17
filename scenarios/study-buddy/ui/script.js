// Study Buddy Interactive JavaScript
const API_URL = 'http://localhost:8500/api';
const N8N_URL = 'http://localhost:5678/webhook';

// Global state
let currentSubject = 'math';
let currentMode = 'flashcards';
let sessionId = null;
let timer = 0;
let timerInterval = null;
let flashcards = [];
let currentCardIndex = 0;
let quizQuestions = [];
let currentQuestionIndex = 0;
let score = 0;

// Pomodoro state
let pomodoroState = {
    active: false,
    sessionType: 'study',
    timeRemaining: 25 * 60, // seconds
    sessionId: null,
    autoStartBreaks: true,
    sessionsCompleted: 0,
    energyLevel: 7
};

// Additional helper functions
function loadNotes() {
    const notes = JSON.parse(localStorage.getItem('study_notes') || '[]');
    const notesForSubject = notes.filter(n => n.subject === currentSubject);
    
    if (notesForSubject.length > 0) {
        const latestNote = notesForSubject[notesForSubject.length - 1];
        document.getElementById('notes-editor').value = latestNote.content;
    }
}

function updateProgressBars() {
    // Update flashcard progress
    const flashcardBar = document.getElementById('flashcard-progress-bar');
    if (flashcardBar && flashcards.length > 0) {
        flashcardBar.style.width = ((currentCardIndex / flashcards.length) * 100) + '%';
    }
    
    // Update quiz progress
    const quizBar = document.getElementById('quiz-progress-bar');
    if (quizBar && quizQuestions.length > 0) {
        const percentage = Math.round((score / quizQuestions.length) * 100);
        quizBar.style.width = percentage + '%';
    }
    
    // Update daily goal
    const points = parseInt(document.getElementById('points').textContent.split(' ')[0]);
    const goalBar = document.getElementById('goal-progress-bar');
    if (goalBar) {
        goalBar.style.width = Math.min(100, (points / 50) * 100) + '%';
    }
}

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    startStudySession();
    startTimer();
    loadFlashcards();
});

function initializeEventListeners() {
    // Subject selection
    document.querySelectorAll('.subject-item').forEach(item => {
        item.addEventListener('click', (e) => {
            document.querySelector('.subject-item.active')?.classList.remove('active');
            e.target.classList.add('active');
            currentSubject = e.target.dataset.subject;
            loadContentForSubject();
        });
    });

    // Mode selection
    document.querySelectorAll('.mode-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelector('.mode-btn.active')?.classList.remove('active');
            e.target.classList.add('active');
            currentMode = e.target.dataset.mode;
            switchMode(currentMode);
        });
    });

    // Flashcard flip
    document.getElementById('flashcard').addEventListener('click', () => {
        document.getElementById('flashcard').classList.toggle('flipped');
    });

    // Flashcard difficulty buttons with spaced repetition
    document.querySelectorAll('.control-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            handleFlashcardResponse(e);
            handleSpacedRepetition(e.target.dataset.quality);
        });
    });

    // Quiz options
    document.querySelectorAll('.quiz-option').forEach(option => {
        option.addEventListener('click', handleQuizAnswer);
    });

    // Save note button
    document.querySelector('.save-note-btn')?.addEventListener('click', saveNote);

    // Floating action button
    document.querySelector('.floating-action')?.addEventListener('click', () => {
        generateFlashcards();
    });

    // Add subject button
    document.querySelector('.add-subject-btn')?.addEventListener('click', () => {
        const name = prompt('Enter subject name:');
        if (name) addSubject(name);
    });
}

function switchMode(mode) {
    // Hide all containers
    document.querySelector('.flashcard-container').style.display = 'none';
    document.querySelector('.flashcard-controls').style.display = 'none';
    document.querySelector('.quiz-container').style.display = 'none';
    document.querySelector('.notes-container').style.display = 'none';

    // Show selected mode
    switch(mode) {
        case 'flashcards':
            document.querySelector('.flashcard-container').style.display = 'block';
            document.querySelector('.flashcard-controls').style.display = 'flex';
            loadFlashcards();
            break;
        case 'quiz':
            document.querySelector('.quiz-container').style.display = 'block';
            loadQuizQuestions();
            break;
        case 'notes':
            document.querySelector('.notes-container').style.display = 'block';
            loadNotes();
            break;
    }
}

async function startStudySession() {
    try {
        const response = await fetch(`${N8N_URL}/study-session/start`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                user_id: getUserId(),
                subject_id: currentSubject,
                session_type: currentMode
            })
        });
        const data = await response.json();
        sessionId = data.session_id;
        
        // Show study tips if available
        if (data.study_tips) {
            console.log('Study tips:', data.study_tips);
        }
    } catch (error) {
        console.error('Error starting study session:', error);
    }
}

async function loadFlashcards() {
    // For demo, use hardcoded flashcards
    flashcards = [
        { front: "What is the derivative of x²?", back: "2x", id: 1 },
        { front: "What is the integral of 2x?", back: "x² + C", id: 2 },
        { front: "What is the Pythagorean theorem?", back: "a² + b² = c²", id: 3 },
        { front: "What is the quadratic formula?", back: "x = (-b ± √(b²-4ac)) / 2a", id: 4 },
        { front: "What is the area of a circle?", back: "πr²", id: 5 }
    ];
    
    currentCardIndex = 0;
    displayFlashcard();
}

function displayFlashcard() {
    if (currentCardIndex < flashcards.length) {
        const card = flashcards[currentCardIndex];
        document.getElementById('card-front-content').textContent = card.front;
        document.getElementById('card-back-content').innerHTML = card.back;
        document.getElementById('flashcard').classList.remove('flipped');
        updateProgress();
    } else {
        // All cards reviewed
        showCompletionMessage();
    }
}

function handleFlashcardResponse(e) {
    const difficulty = e.target.dataset.difficulty || 'medium';
    
    // Animate button press
    e.target.style.transform = 'scale(0.95)';
    setTimeout(() => e.target.style.transform = '', 100);
    
    // Update progress
    updateStats(difficulty);
    
    // Move to next card
    currentCardIndex++;
    setTimeout(() => displayFlashcard(), 300);
}

async function loadQuizQuestions() {
    // Demo quiz questions
    quizQuestions = [
        {
            question: "Which planet is known as the Red Planet?",
            options: ["Venus", "Mars", "Jupiter", "Saturn"],
            correct: 1
        },
        {
            question: "What is the largest mammal?",
            options: ["Elephant", "Blue Whale", "Giraffe", "Hippopotamus"],
            correct: 1
        },
        {
            question: "What is 15 × 15?",
            options: ["215", "225", "235", "245"],
            correct: 1
        }
    ];
    
    currentQuestionIndex = 0;
    score = 0;
    displayQuizQuestion();
}

function displayQuizQuestion() {
    if (currentQuestionIndex < quizQuestions.length) {
        const q = quizQuestions[currentQuestionIndex];
        document.querySelector('.quiz-question').textContent = q.question;
        
        const optionsContainer = document.querySelector('.quiz-options');
        optionsContainer.innerHTML = '';
        
        q.options.forEach((option, index) => {
            const optionDiv = document.createElement('div');
            optionDiv.className = 'quiz-option';
            optionDiv.textContent = option;
            optionDiv.dataset.index = index;
            optionDiv.addEventListener('click', handleQuizAnswer);
            optionsContainer.appendChild(optionDiv);
        });
        
        updateProgress();
    } else {
        showQuizResults();
    }
}

function handleQuizAnswer(e) {
    const selectedIndex = parseInt(e.target.dataset.index);
    const correct = quizQuestions[currentQuestionIndex].correct;
    
    // Disable all options
    document.querySelectorAll('.quiz-option').forEach(opt => {
        opt.style.pointerEvents = 'none';
    });
    
    if (selectedIndex === correct) {
        e.target.classList.add('correct');
        score++;
        updateXP(10);
    } else {
        e.target.classList.add('incorrect');
        document.querySelectorAll('.quiz-option')[correct].classList.add('correct');
    }
    
    currentQuestionIndex++;
    setTimeout(() => displayQuizQuestion(), 1500);
}

async function generateFlashcards() {
    const content = prompt('Enter text to generate flashcards from:');
    if (!content) return;
    
    try {
        const response = await fetch(`${N8N_URL}/flashcards/generate`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({
                user_id: getUserId(),
                subject_id: currentSubject,
                content: content,
                count: 5,
                difficulty: 'medium'
            })
        });
        
        const data = await response.json();
        if (data.flashcards) {
            flashcards = data.flashcards;
            currentCardIndex = 0;
            displayFlashcard();
            showNotification('Generated ' + data.flashcards.length + ' new flashcards!');
        }
    } catch (error) {
        console.error('Error generating flashcards:', error);
    }
}

function saveNote() {
    const noteContent = document.getElementById('notes-editor').value;
    if (!noteContent) return;
    
    // Save to local storage for demo
    const notes = JSON.parse(localStorage.getItem('study_notes') || '[]');
    notes.push({
        subject: currentSubject,
        content: noteContent,
        timestamp: new Date().toISOString()
    });
    localStorage.setItem('study_notes', JSON.stringify(notes));
    
    // Clear input and show confirmation
    document.getElementById('notes-editor').value = '';
    showNotification('Note saved successfully!');
}

function updateProgress() {
    if (currentMode === 'flashcards') {
        const progress = (currentCardIndex / flashcards.length) * 100;
        const progressEl = document.getElementById('flashcard-progress');
        progressEl.textContent = `${currentCardIndex}/${flashcards.length}`;
        const progressBar = document.getElementById('flashcard-progress-bar');
        progressBar.style.width = progress + '%';
    } else if (currentMode === 'quiz') {
        const progress = (currentQuestionIndex / quizQuestions.length) * 100;
        const scoreEl = document.getElementById('quiz-score');
        const percentage = quizQuestions.length > 0 ? Math.round((score / quizQuestions.length) * 100) : 0;
        scoreEl.textContent = `${percentage}%`;
        const progressBar = document.getElementById('quiz-progress-bar');
        progressBar.style.width = progress + '%';
    }
}

function updateStats(difficulty) {
    // Update cards studied count
    const cardsEl = document.getElementById('cards-studied');
    const currentCount = parseInt(cardsEl.textContent.split(' ')[0]);
    cardsEl.textContent = `${currentCount + 1} cards today`;
    
    // Update XP
    const xpGain = difficulty === 'easy' ? 5 : difficulty === 'medium' ? 10 : 15;
    updateXP(xpGain);
    
    // Update progress bars
    updateProgressBars();
}

function updateXP(amount) {
    const pointsEl = document.getElementById('points');
    const currentPoints = parseInt(pointsEl.textContent.split(' ')[0]);
    const newPoints = currentPoints + amount;
    pointsEl.textContent = `${newPoints} points`;
    
    // Animate points gain
    pointsEl.style.transform = 'scale(1.2)';
    pointsEl.style.color = '#f9d978';
    setTimeout(() => {
        pointsEl.style.transform = '';
        pointsEl.style.color = '';
    }, 300);
    
    // Update daily goal progress
    const goalEl = document.getElementById('daily-goal');
    goalEl.textContent = `${newPoints}/50 XP`;
    const goalBar = document.getElementById('goal-progress-bar');
    goalBar.style.width = Math.min(100, (newPoints / 50) * 100) + '%';
}

function startTimer() {
    timerInterval = setInterval(() => {
        timer++;
        const minutes = Math.floor(timer / 60);
        const seconds = timer % 60;
        document.getElementById('timer').textContent = 
            `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }, 1000);
}

function showCompletionMessage() {
    const container = document.querySelector('.flashcard-container');
    container.innerHTML = `
        <div style="text-align: center; padding: 2rem;">
            <h2 style="color: #667eea; margin-bottom: 1rem;">🎉 Great Job!</h2>
            <p style="font-size: 1.2rem; margin-bottom: 2rem;">You've reviewed all flashcards!</p>
            <button onclick="loadFlashcards()" style="padding: 1rem 2rem; background: linear-gradient(135deg, #667eea, #764ba2); color: white; border: none; border-radius: 0.5rem; cursor: pointer; font-size: 1rem;">
                Review Again
            </button>
        </div>
    `;
}

function showQuizResults() {
    const container = document.querySelector('.quiz-container');
    const percentage = Math.round((score / quizQuestions.length) * 100);
    container.innerHTML = `
        <div style="text-align: center; padding: 2rem;">
            <h2 style="color: #667eea; margin-bottom: 1rem;">Quiz Complete!</h2>
            <p style="font-size: 2rem; margin-bottom: 1rem;">${score} / ${quizQuestions.length}</p>
            <p style="font-size: 1.5rem; color: ${percentage >= 70 ? '#4ade80' : '#fbbf24'};">
                ${percentage}%
            </p>
            <button onclick="loadQuizQuestions()" style="margin-top: 2rem; padding: 1rem 2rem; background: linear-gradient(135deg, #667eea, #764ba2); color: white; border: none; border-radius: 0.5rem; cursor: pointer; font-size: 1rem;">
                Try Again
            </button>
        </div>
    `;
}

function showNotification(message) {
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: 2rem;
        right: 2rem;
        background: white;
        padding: 1rem 1.5rem;
        border-radius: 0.5rem;
        box-shadow: 0 4px 20px rgba(0,0,0,0.15);
        animation: slideIn 0.3s;
        z-index: 1000;
    `;
    notification.textContent = message;
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

function getUserId() {
    // Get or create user ID from localStorage
    let userId = localStorage.getItem('study_buddy_user_id');
    if (!userId) {
        userId = 'user-' + Math.random().toString(36).substr(2, 9);
        localStorage.setItem('study_buddy_user_id', userId);
    }
    return userId;
}

function addSubject(name) {
    const list = document.getElementById('subjects-list');
    const newItem = document.createElement('div');
    newItem.className = 'subject-item';
    newItem.dataset.subject = name.toLowerCase().replace(/\s+/g, '-');
    newItem.innerHTML = `<span>📚</span> ${name}`;
    newItem.addEventListener('click', (e) => {
        document.querySelector('.subject-item.active')?.classList.remove('active');
        e.currentTarget.classList.add('active');
        currentSubject = e.currentTarget.dataset.subject;
        loadContentForSubject();
    });
    list.appendChild(newItem);
}

function loadContentForSubject() {
    // Reset and reload content for new subject
    currentCardIndex = 0;
    currentQuestionIndex = 0;
    
    if (currentMode === 'flashcards') {
        loadFlashcards();
    } else if (currentMode === 'quiz') {
        loadQuizQuestions();
    }
}

// Pomodoro Timer Functions
async function startPomodoroSession(sessionType = 'study') {
    try {
        // For now, use local timer logic since n8n integration may not be ready
        pomodoroState.active = true;
        pomodoroState.sessionType = sessionType;
        pomodoroState.sessionId = 'session-' + Date.now();
        
        // Set duration based on session type
        if (sessionType === 'study') {
            pomodoroState.timeRemaining = 25 * 60; // 25 minutes
        } else if (sessionType === 'short_break') {
            pomodoroState.timeRemaining = 5 * 60; // 5 minutes
        } else if (sessionType === 'long_break') {
            pomodoroState.timeRemaining = 15 * 60; // 15 minutes
        }
        
        startPomodoroTimer();
        
        // Show encouraging message based on session type
        const messages = {
            'study': 'Focus time! Let\'s dive deep 🎯',
            'short_break': 'Quick break - stretch and breathe 🌸',
            'long_break': 'Long break - time to recharge ☕'
        };
        showNotification(messages[sessionType] || 'Session started!');
        
        // Update UI to show Pomodoro state
        updatePomodoroUI();
        
        // Try to call n8n workflow if available (non-blocking)
        fetch(`${N8N_URL}/timer/pomodoro`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                user_id: getUserId(),
                subject_id: currentSubject,
                action: 'start',
                session_type: sessionType,
                current_topic: currentMode,
                difficulty_level: 5,
                energy_level: pomodoroState.energyLevel,
                previous_sessions_today: pomodoroState.sessionsCompleted
            })
        }).catch(error => {
            console.log('N8n workflow not available, using local timer');
        });
        
    } catch (error) {
        console.error('Failed to start Pomodoro session:', error);
        // Still start local timer even if n8n fails
        pomodoroState.active = true;
        pomodoroState.sessionType = sessionType;
        pomodoroState.timeRemaining = 25 * 60;
        startPomodoroTimer();
        updatePomodoroUI();
    }
}

function startPomodoroTimer() {
    if (pomodoroState.active && !timerInterval) {
        timerInterval = setInterval(() => {
            pomodoroState.timeRemaining--;
            updateTimerDisplay();
            
            if (pomodoroState.timeRemaining <= 0) {
                completePomodoroSession();
            }
        }, 1000);
    }
}

async function completePomodoroSession() {
    clearInterval(timerInterval);
    timerInterval = null;
    
    // Play completion sound
    playSound('complete');
    
    try {
        const response = await fetch(`${N8N_URL}/timer/pomodoro`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                user_id: 'default-user',
                subject_id: currentSubject,
                action: 'complete',
                session_type: pomodoroState.sessionType
            })
        });
        
        const data = await response.json();
        
        // Update sessions completed
        if (pomodoroState.sessionType === 'study') {
            pomodoroState.sessionsCompleted++;
        }
        
        // Auto-start next session if enabled
        if (data.next_session && data.next_session.auto_start) {
            setTimeout(() => {
                startPomodoroSession(data.next_session.type);
            }, 3000);
        } else {
            // Show break suggestion
            showBreakSuggestion(data.next_session);
        }
        
        pomodoroState.active = false;
        updatePomodoroUI();
        
    } catch (error) {
        console.error('Failed to complete Pomodoro session:', error);
    }
}

function pausePomodoroSession() {
    if (pomodoroState.active) {
        clearInterval(timerInterval);
        timerInterval = null;
        pomodoroState.active = false;
        updatePomodoroUI();
    }
}

function resumePomodoroSession() {
    if (!pomodoroState.active && pomodoroState.timeRemaining > 0) {
        pomodoroState.active = true;
        startPomodoroTimer();
        updatePomodoroUI();
    }
}

function updateTimerDisplay() {
    const minutes = Math.floor(pomodoroState.timeRemaining / 60);
    const seconds = pomodoroState.timeRemaining % 60;
    const display = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    
    const timerElement = document.getElementById('timer');
    if (timerElement) {
        timerElement.textContent = display;
        
        // Add visual cues for different session types
        if (pomodoroState.sessionType === 'study') {
            timerElement.style.color = '#7bb68e';
        } else if (pomodoroState.sessionType === 'short_break') {
            timerElement.style.color = '#f9d978';
        } else if (pomodoroState.sessionType === 'long_break') {
            timerElement.style.color = '#d4a5c8';
        }
        
        // Add urgency animation for last minute
        if (pomodoroState.timeRemaining <= 60 && pomodoroState.timeRemaining > 0) {
            timerElement.classList.add('pulse');
        } else {
            timerElement.classList.remove('pulse');
        }
    }
}

function updatePomodoroUI() {
    // Update timer control buttons
    const timerCard = document.querySelector('.study-timer');
    if (timerCard) {
        if (!timerCard.querySelector('.pomodoro-controls')) {
            const controls = document.createElement('div');
            controls.className = 'pomodoro-controls';
            controls.innerHTML = `
                <button class="pomodoro-btn" id="pomodoro-toggle">
                    ${pomodoroState.active ? '⏸️' : '▶️'}
                </button>
                <button class="pomodoro-btn" id="pomodoro-skip">⏭️</button>
                <button class="pomodoro-btn" id="pomodoro-reset">🔄</button>
            `;
            timerCard.appendChild(controls);
            
            // Add event listeners
            document.getElementById('pomodoro-toggle').addEventListener('click', () => {
                if (pomodoroState.active) {
                    pausePomodoroSession();
                } else if (pomodoroState.timeRemaining > 0) {
                    resumePomodoroSession();
                } else {
                    startPomodoroSession('study');
                }
            });
            
            document.getElementById('pomodoro-skip').addEventListener('click', () => {
                if (confirm('Skip current session?')) {
                    completePomodoroSession();
                }
            });
            
            document.getElementById('pomodoro-reset').addEventListener('click', () => {
                if (confirm('Reset timer?')) {
                    clearInterval(timerInterval);
                    timerInterval = null;
                    pomodoroState.active = false;
                    pomodoroState.timeRemaining = 25 * 60;
                    updateTimerDisplay();
                }
            });
        }
        
        // Update button states
        const toggleBtn = document.getElementById('pomodoro-toggle');
        if (toggleBtn) {
            toggleBtn.innerHTML = pomodoroState.active ? '⏸️' : '▶️';
        }
    }
}

function showBreakSuggestion(nextSession) {
    const notification = document.createElement('div');
    notification.className = 'break-suggestion';
    notification.innerHTML = `
        <div class="suggestion-content">
            <h3>Great work! 🎉</h3>
            <p>Time for a ${nextSession.type.replace('_', ' ')}!</p>
            <p>Duration: ${nextSession.duration_minutes} minutes</p>
            <button onclick="startPomodoroSession('${nextSession.type}')">Start Break</button>
            <button onclick="this.parentElement.parentElement.remove()">Skip</button>
        </div>
    `;
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.classList.add('show');
    }, 100);
    
    setTimeout(() => {
        notification.remove();
    }, 10000);
}

function showNotification(message) {
    const notification = document.createElement('div');
    notification.className = 'notification';
    notification.textContent = message;
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.classList.add('show');
    }, 100);
    
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

function playSound(type) {
    // Simple sound effects using Web Audio API
    const audioContext = new (window.AudioContext || window.webkitAudioContext)();
    const oscillator = audioContext.createOscillator();
    const gainNode = audioContext.createGain();
    
    oscillator.connect(gainNode);
    gainNode.connect(audioContext.destination);
    
    if (type === 'complete') {
        // Pleasant completion chime
        oscillator.frequency.setValueAtTime(523.25, audioContext.currentTime); // C5
        oscillator.frequency.setValueAtTime(659.25, audioContext.currentTime + 0.1); // E5
        oscillator.frequency.setValueAtTime(783.99, audioContext.currentTime + 0.2); // G5
        gainNode.gain.setValueAtTime(0.3, audioContext.currentTime);
        gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.5);
    }
    
    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.5);
}

// Spaced Repetition Handler
async function handleSpacedRepetition(quality) {
    // Convert button quality to SM-2 scale (0-5)
    const qualityMap = {
        'again': 0,
        'hard': 2,
        'good': 3,
        'easy': 5
    };
    
    const sm2Quality = qualityMap[quality] || 3;
    const currentCard = flashcards[currentCardIndex - 1]; // -1 because we already incremented
    
    if (!currentCard) return;
    
    try {
        const response = await fetch(`${N8N_URL}/flashcards/spaced-repetition`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                flashcard_id: currentCard.id,
                user_id: getUserId(),
                quality: sm2Quality,
                ease_factor: currentCard.ease_factor || 2.5,
                interval: currentCard.interval || 1,
                repetitions: currentCard.repetitions || 0,
                last_reviewed: currentCard.last_reviewed || new Date().toISOString()
            })
        });
        
        if (response.ok) {
            const data = await response.json();
            
            // Update the flashcard with new spaced repetition values
            Object.assign(currentCard, data.updated_values);
            
            // Show learning insight if available
            if (data.learning_insight) {
                showLearningInsight(data.learning_insight);
            }
            
            // Update progress display
            updateSpacedRepetitionStats(data.statistics);
            
            // Save updated flashcards to localStorage
            localStorage.setItem(`flashcards_${currentSubject}`, JSON.stringify(flashcards));
        }
    } catch (error) {
        console.log('Spaced repetition tracking offline, saving locally');
        // Fallback: Save locally if n8n is not available
        currentCard.last_reviewed = new Date().toISOString();
        currentCard.review_count = (currentCard.review_count || 0) + 1;
        localStorage.setItem(`flashcards_${currentSubject}`, JSON.stringify(flashcards));
    }
}

function showLearningInsight(insight) {
    const insightDiv = document.createElement('div');
    insightDiv.className = 'learning-insight';
    insightDiv.innerHTML = `
        <div class="insight-emoji">${insight.emoji || '🌟'}</div>
        <div class="insight-message">${insight.message}</div>
        <div class="insight-tip">💡 ${insight.tip}</div>
    `;
    
    // Add styles for learning insight
    const style = `
        .learning-insight {
            position: fixed;
            bottom: 100px;
            left: 50%;
            transform: translateX(-50%) translateY(100px);
            background: white;
            padding: 1.5rem;
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            max-width: 400px;
            animation: slideUpInsight 0.5s forwards;
            z-index: 1000;
        }
        @keyframes slideUpInsight {
            to { transform: translateX(-50%) translateY(0); }
        }
        .insight-emoji {
            font-size: 2rem;
            text-align: center;
            margin-bottom: 0.5rem;
        }
        .insight-message {
            color: var(--text-dark);
            font-weight: 600;
            margin-bottom: 0.5rem;
            text-align: center;
        }
        .insight-tip {
            color: var(--text-medium);
            font-size: 0.9rem;
            padding-top: 0.5rem;
            border-top: 1px solid var(--notebook-lines);
        }
    `;
    
    if (!document.querySelector('#insight-styles')) {
        const styleEl = document.createElement('style');
        styleEl.id = 'insight-styles';
        styleEl.textContent = style;
        document.head.appendChild(styleEl);
    }
    
    document.body.appendChild(insightDiv);
    
    setTimeout(() => {
        insightDiv.style.animation = 'slideDownInsight 0.5s forwards';
        setTimeout(() => insightDiv.remove(), 500);
    }, 4000);
}

function updateSpacedRepetitionStats(stats) {
    // Update UI elements with spaced repetition statistics
    const streakElement = document.querySelector('.streak-count');
    if (streakElement) {
        streakElement.textContent = stats.streak;
    }
    
    // Show performance indicator
    if (stats.performance === 'excellent') {
        showAchievement('Perfect Recall!', 'Your memory is on fire! 🔥');
    } else if (stats.performance === 'again' && stats.streak > 5) {
        showNotification('Streak broken, but keep going! You\'ve got this! 💪');
    }
}

function showAchievement(title, description) {
    const achievement = document.createElement('div');
    achievement.className = 'achievement-popup show';
    achievement.innerHTML = `
        <div class="achievement-icon">🏆</div>
        <div class="achievement-title">${title}</div>
        <div class="achievement-desc">${description}</div>
    `;
    
    document.body.appendChild(achievement);
    
    setTimeout(() => {
        achievement.classList.remove('show');
        setTimeout(() => achievement.remove(), 500);
    }, 3000);
}

// Add CSS animations
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
    @keyframes pulse {
        0% { transform: scale(1); }
        50% { transform: scale(1.05); }
        100% { transform: scale(1); }
    }
    .pulse {
        animation: pulse 1s infinite;
    }
    .pomodoro-controls {
        display: flex;
        gap: 0.5rem;
        margin-top: 1rem;
        justify-content: center;
    }
    .pomodoro-btn {
        background: rgba(255, 255, 255, 0.2);
        border: none;
        border-radius: 50%;
        width: 40px;
        height: 40px;
        font-size: 1.2rem;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    .pomodoro-btn:hover {
        background: rgba(255, 255, 255, 0.3);
        transform: scale(1.1);
    }
    .break-suggestion {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%) scale(0);
        background: var(--card-bg);
        padding: 2rem;
        border-radius: 20px;
        box-shadow: 0 20px 60px rgba(0,0,0,0.3);
        z-index: 1000;
        transition: transform 0.3s ease;
    }
    .break-suggestion.show {
        transform: translate(-50%, -50%) scale(1);
    }
    .suggestion-content {
        text-align: center;
    }
    .suggestion-content button {
        margin: 0.5rem;
        padding: 0.8rem 1.5rem;
        border: none;
        border-radius: 25px;
        background: var(--accent-purple);
        color: white;
        cursor: pointer;
        font-weight: 600;
        transition: all 0.3s ease;
    }
    .suggestion-content button:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(155, 114, 170, 0.4);
    }
    .notification {
        position: fixed;
        top: 20px;
        right: -300px;
        background: var(--card-bg);
        padding: 1rem 1.5rem;
        border-radius: 12px;
        box-shadow: 0 5px 20px rgba(0,0,0,0.2);
        transition: right 0.3s ease;
        z-index: 999;
    }
    .notification.show {
        right: 20px;
    }
`;
document.head.appendChild(style);