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

    // Flashcard difficulty buttons
    document.querySelectorAll('.control-btn').forEach(btn => {
        btn.addEventListener('click', handleFlashcardResponse);
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
    document.querySelector('.quiz-container').style.display = 'none';
    document.querySelector('.notes-container').style.display = 'none';

    // Show selected mode
    switch(mode) {
        case 'flashcards':
            document.querySelector('.flashcard-container').style.display = 'flex';
            loadFlashcards();
            break;
        case 'quiz':
            document.querySelector('.quiz-container').style.display = 'block';
            loadQuizQuestions();
            break;
        case 'notes':
            document.querySelector('.notes-container').style.display = 'flex';
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
        document.querySelector('.flashcard-front > div').textContent = card.front;
        document.querySelector('.flashcard-back > div').textContent = card.back;
        document.getElementById('flashcard').classList.remove('flipped');
        updateProgress();
    } else {
        // All cards reviewed
        showCompletionMessage();
    }
}

function handleFlashcardResponse(e) {
    const difficulty = e.target.textContent.toLowerCase();
    
    // Animate button press
    e.target.style.transform = 'scale(0.95)';
    setTimeout(() => e.target.style.transform = '', 100);
    
    // Update progress
    if (difficulty !== 'skip') {
        updateStats(difficulty);
    }
    
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
    const noteContent = document.querySelector('.note-input').value;
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
    document.querySelector('.note-input').value = '';
    showNotification('Note saved successfully!');
}

function updateProgress() {
    let progress = 0;
    if (currentMode === 'flashcards') {
        progress = (currentCardIndex / flashcards.length) * 100;
    } else if (currentMode === 'quiz') {
        progress = (currentQuestionIndex / quizQuestions.length) * 100;
    }
    document.querySelector('.progress-fill').style.width = progress + '%';
}

function updateStats(difficulty) {
    // Update mastered cards count
    if (difficulty === 'easy') {
        const masteredEl = document.getElementById('mastered');
        masteredEl.textContent = parseInt(masteredEl.textContent) + 1;
    }
    
    // Update XP
    const xpGain = difficulty === 'easy' ? 5 : difficulty === 'medium' ? 10 : 15;
    updateXP(xpGain);
}

function updateXP(amount) {
    const xpEl = document.getElementById('xp');
    const currentXP = parseInt(xpEl.textContent.replace(',', ''));
    xpEl.textContent = (currentXP + amount).toLocaleString();
    
    // Animate XP gain
    xpEl.style.transform = 'scale(1.2)';
    xpEl.style.color = '#4ade80';
    setTimeout(() => {
        xpEl.style.transform = '';
        xpEl.style.color = '#667eea';
    }, 300);
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
    const list = document.querySelector('.subject-list');
    const newItem = document.createElement('li');
    newItem.className = 'subject-item';
    newItem.dataset.subject = name.toLowerCase().replace(/\s+/g, '-');
    newItem.textContent = '📚 ' + name;
    newItem.addEventListener('click', (e) => {
        document.querySelector('.subject-item.active')?.classList.remove('active');
        e.target.classList.add('active');
        currentSubject = e.target.dataset.subject;
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
`;
document.head.appendChild(style);