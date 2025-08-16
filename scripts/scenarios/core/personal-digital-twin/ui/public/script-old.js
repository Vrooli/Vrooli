// Digital Twin Console - Neural Interface Controller
// Consciousness Transfer Protocol v2.0

let config = null;
let currentPersona = null;
let currentSessionId = null;
let personas = [];

// Initialize on page load
document.addEventListener('DOMContentLoaded', async () => {
    await loadConfig();
    await loadPersonas();
    updateStats();
    
    // Add typing animation to initial message
    animateTyping();
    
    // Add ambient sound effect simulation
    setInterval(updateAmbientEffects, 3000);
});

// Load configuration from server
async function loadConfig() {
    try {
        const response = await fetch('/config');
        config = await response.json();
        console.log('Neural link established:', config);
    } catch (error) {
        console.error('Failed to establish neural link:', error);
        showError('Neural link connection failed. Check system integrity.');
    }
}

// Load all personas
async function loadPersonas() {
    if (!config) return;
    
    showLoading(true);
    try {
        const response = await fetch(`${config.apiUrl}/api/personas`);
        personas = await response.json();
        displayPersonas();
        updateStats();
    } catch (error) {
        console.error('Failed to load personas:', error);
        showError('Persona retrieval failed. Neural database may be offline.');
    } finally {
        showLoading(false);
    }
}

// Display personas in the control panel
function displayPersonas() {
    const personaList = document.getElementById('personaList');
    
    if (personas.length === 0) {
        personaList.innerHTML = `
            <div style="text-align: center; padding: 20px; color: var(--terminal-amber);">
                No personas detected in neural matrix.
                Initialize a new consciousness to begin.
            </div>
        `;
        return;
    }
    
    personaList.innerHTML = personas.map(persona => `
        <div class="persona-item ${currentPersona?.id === persona.id ? 'active' : ''}" 
             onclick="selectPersona('${persona.id}')"
             data-persona-id="${persona.id}">
            <div class="persona-name">${persona.name}</div>
            <div class="persona-status">
                <span class="status-indicator ${persona.training_status === 'completed' ? 'ready' : 'training'}"></span>
                <span>${persona.training_status || 'Initializing'}</span>
                <span style="margin-left: auto;">
                    ${persona.document_count || 0} knowledge units
                </span>
            </div>
        </div>
    `).join('');
}

// Select a persona
async function selectPersona(personaId) {
    const persona = personas.find(p => p.id === personaId);
    if (!persona) return;
    
    currentPersona = persona;
    currentSessionId = generateSessionId();
    
    // Update UI
    document.querySelectorAll('.persona-item').forEach(item => {
        item.classList.toggle('active', item.dataset.personaId === personaId);
    });
    
    // Clear chat and add activation message
    const chatMessages = document.getElementById('chatMessages');
    chatMessages.innerHTML = `
        <div class="message twin">
            <div class="message-author">System</div>
            <div class="message-content">Neural link established with ${persona.name}. Consciousness transfer complete.</div>
        </div>
        <div class="message twin">
            <div class="message-author">${persona.name}</div>
            <div class="message-content">${persona.description || 'Digital twin activated. Ready for interaction.'}</div>
        </div>
    `;
    
    updateStats();
    
    // Visual feedback
    flashEffect();
}

// Send chat message
async function sendMessage() {
    const input = document.getElementById('chatInput');
    const message = input.value.trim();
    
    if (!message || !currentPersona) {
        if (!currentPersona) {
            showError('No persona selected. Initialize neural link first.');
        }
        return;
    }
    
    // Add user message
    addMessage('User', message, 'user');
    input.value = '';
    
    // Show typing indicator
    const typingId = addTypingIndicator(currentPersona.name);
    
    try {
        const response = await fetch(`${config.chatUrl}/api/chat`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                persona_id: currentPersona.id,
                message: message,
                session_id: currentSessionId
            })
        });
        
        const data = await response.json();
        
        // Remove typing indicator
        removeTypingIndicator(typingId);
        
        // Add twin response
        addMessage(currentPersona.name, data.response, 'twin');
        
    } catch (error) {
        removeTypingIndicator(typingId);
        console.error('Chat transmission failed:', error);
        addMessage('System', 'Neural transmission interrupted. Please retry.', 'twin');
    }
}

// Add message to chat
function addMessage(author, content, type) {
    const chatMessages = document.getElementById('chatMessages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${type}`;
    messageDiv.innerHTML = `
        <div class="message-author">${author}</div>
        <div class="message-content">${content}</div>
    `;
    chatMessages.appendChild(messageDiv);
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Add typing indicator
function addTypingIndicator(name) {
    const id = 'typing-' + Date.now();
    const chatMessages = document.getElementById('chatMessages');
    const indicator = document.createElement('div');
    indicator.id = id;
    indicator.className = 'message twin';
    indicator.innerHTML = `
        <div class="message-author">${name}</div>
        <div class="message-content">
            <span class="typing-dots">
                <span>.</span><span>.</span><span>.</span>
            </span>
        </div>
    `;
    chatMessages.appendChild(indicator);
    chatMessages.scrollTop = chatMessages.scrollHeight;
    return id;
}

// Remove typing indicator
function removeTypingIndicator(id) {
    const indicator = document.getElementById(id);
    if (indicator) {
        indicator.remove();
    }
}

// Handle chat input keypress
function handleChatKeypress(event) {
    if (event.key === 'Enter') {
        sendMessage();
    }
}

// Show create persona modal
function showCreatePersonaModal() {
    document.getElementById('createPersonaModal').classList.add('active');
    // Add glitch effect
    glitchEffect();
}

// Close modal
function closeModal() {
    document.getElementById('createPersonaModal').classList.remove('active');
}

// Create new persona
async function createPersona(event) {
    event.preventDefault();
    
    const name = document.getElementById('personaName').value;
    const description = document.getElementById('personaDescription').value;
    const traitsText = document.getElementById('personalityTraits').value;
    
    let personalityTraits = {};
    if (traitsText) {
        try {
            personalityTraits = JSON.parse(traitsText);
        } catch (e) {
            showError('Invalid personality matrix format. Use valid JSON.');
            return;
        }
    }
    
    showLoading(true);
    
    try {
        const response = await fetch(`${config.apiUrl}/api/persona/create`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                name,
                description,
                personality_traits: personalityTraits
            })
        });
        
        if (response.ok) {
            closeModal();
            await loadPersonas();
            showSuccess('Persona initialized successfully. Neural patterns established.');
        } else {
            throw new Error('Persona initialization failed');
        }
    } catch (error) {
        console.error('Failed to create persona:', error);
        showError('Persona initialization failed. Check neural matrix configuration.');
    } finally {
        showLoading(false);
    }
}

// Connect data source
async function connectDataSource() {
    if (!currentPersona) {
        showError('Select a persona before connecting data sources.');
        return;
    }
    
    // For demo, show a simple prompt
    const sourceType = prompt('Enter data source type (google_drive, local_files, github):');
    if (!sourceType) return;
    
    showLoading(true);
    
    try {
        const response = await fetch(`${config.apiUrl}/api/datasource/connect`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                persona_id: currentPersona.id,
                source_type: sourceType,
                source_config: {}
            })
        });
        
        if (response.ok) {
            showSuccess('Data source connected. Knowledge assimilation initiated.');
            await loadPersonas();
        }
    } catch (error) {
        console.error('Failed to connect data source:', error);
        showError('Data source connection failed.');
    } finally {
        showLoading(false);
    }
}

// Start training
async function startTraining() {
    if (!currentPersona) {
        showError('Select a persona before initiating training.');
        return;
    }
    
    showLoading(true);
    
    try {
        const response = await fetch(`${config.apiUrl}/api/train/start`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                persona_id: currentPersona.id,
                model: 'llama3.2',
                technique: 'fine-tuning'
            })
        });
        
        if (response.ok) {
            showSuccess('Neural training initiated. Consciousness evolution in progress.');
            await loadPersonas();
        }
    } catch (error) {
        console.error('Failed to start training:', error);
        showError('Training initialization failed.');
    } finally {
        showLoading(false);
    }
}

// Generate API token
async function generateAPIToken() {
    if (!currentPersona) {
        showError('Select a persona before generating access tokens.');
        return;
    }
    
    const tokenName = prompt('Enter token designation:');
    if (!tokenName) return;
    
    showLoading(true);
    
    try {
        const response = await fetch(`${config.apiUrl}/api/tokens/create`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                persona_id: currentPersona.id,
                name: tokenName,
                permissions: ['read', 'write', 'chat']
            })
        });
        
        if (response.ok) {
            const token = await response.json();
            alert(`Access token generated:\n${token.token}\n\nStore this securely. It will not be shown again.`);
            showSuccess('Neural access token generated successfully.');
        }
    } catch (error) {
        console.error('Failed to generate token:', error);
        showError('Token generation failed.');
    } finally {
        showLoading(false);
    }
}

// Update statistics
function updateStats() {
    document.getElementById('totalPersonas').textContent = personas.length;
    
    const totalDocs = personas.reduce((sum, p) => sum + (p.document_count || 0), 0);
    document.getElementById('totalDocuments').textContent = formatNumber(totalDocs);
    
    const totalTokens = personas.reduce((sum, p) => sum + (p.total_tokens || 0), 0);
    document.getElementById('totalTokens').textContent = formatNumber(totalTokens);
    
    const activeTraining = personas.filter(p => p.training_status === 'training').length;
    document.getElementById('activeTraining').textContent = activeTraining;
}

// Utility functions
function generateSessionId() {
    return 'session-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
}

function formatNumber(num) {
    if (num > 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num > 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function showLoading(show) {
    document.getElementById('loading').classList.toggle('active', show);
}

function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error';
    errorDiv.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: rgba(255, 0, 68, 0.2);
        border: 1px solid var(--glitch-red);
        padding: 15px 20px;
        border-radius: 5px;
        z-index: 10000;
        animation: glitch 0.3s infinite;
    `;
    errorDiv.textContent = '⚠ ' + message;
    document.body.appendChild(errorDiv);
    
    setTimeout(() => errorDiv.remove(), 5000);
}

function showSuccess(message) {
    const successDiv = document.createElement('div');
    successDiv.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: rgba(0, 255, 0, 0.2);
        border: 1px solid var(--neon-green);
        color: var(--neon-green);
        padding: 15px 20px;
        border-radius: 5px;
        z-index: 10000;
    `;
    successDiv.textContent = '✓ ' + message;
    document.body.appendChild(successDiv);
    
    setTimeout(() => successDiv.remove(), 3000);
}

// Visual effects
function flashEffect() {
    document.body.style.animation = 'flash 0.3s';
    setTimeout(() => {
        document.body.style.animation = '';
    }, 300);
}

function glitchEffect() {
    const elements = document.querySelectorAll('.modal-content *');
    elements.forEach((el, i) => {
        setTimeout(() => {
            el.style.animation = 'glitch 0.1s';
            setTimeout(() => {
                el.style.animation = '';
            }, 100);
        }, i * 20);
    });
}

function animateTyping() {
    const style = document.createElement('style');
    style.textContent = `
        .typing-dots span {
            animation: typing-bounce 1.4s infinite;
            display: inline-block;
        }
        .typing-dots span:nth-child(2) {
            animation-delay: 0.2s;
        }
        .typing-dots span:nth-child(3) {
            animation-delay: 0.4s;
        }
        @keyframes typing-bounce {
            0%, 60%, 100% {
                transform: translateY(0);
            }
            30% {
                transform: translateY(-10px);
            }
        }
        @keyframes flash {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.7; }
        }
    `;
    document.head.appendChild(style);
}

function updateAmbientEffects() {
    // Randomly update some stats to simulate activity
    if (Math.random() > 0.7) {
        const tokens = document.getElementById('totalTokens');
        const current = parseInt(tokens.textContent.replace(/[KM]/, '')) || 0;
        tokens.textContent = formatNumber(current + Math.floor(Math.random() * 100));
    }
}

// Initialize WebSocket for real-time updates (if available)
function initializeWebSocket() {
    if (!config || !config.wsUrl) return;
    
    const ws = new WebSocket(config.wsUrl);
    
    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type === 'persona_update') {
            loadPersonas();
        }
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
}

console.log(`
╔══════════════════════════════════════════════════╗
║     DIGITAL TWIN CONSOLE - NEURAL INTERFACE       ║
║     Consciousness Transfer Protocol v2.0          ║
║     © 2025 Vrooli Systems - Mind Upload Division  ║
╚══════════════════════════════════════════════════╝
`);