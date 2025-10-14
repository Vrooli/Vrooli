import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_STATE_KEY = '__gameDialogBridgeInitialized';

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_STATE_KEY]) {
    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[GameDialogGenerator] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'game-dialog-generator' });
    window[BRIDGE_STATE_KEY] = true;
}

// 🌿 Game Dialog Generator JavaScript 🎮
// Jungle Platformer Adventure Theme

let currentSection = 'welcome';
let characters = [];
let projects = [];

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    console.log('🌿 Game Dialog Generator loaded! 🎮');
    initializeFloatingLeaves();
    loadCharacters();
    loadProjects();

    // Set up trait sliders
    const sliders = ['brave', 'humorous', 'loyal'];
    sliders.forEach(trait => {
        const slider = document.getElementById(`trait-${trait}`);
        const valueSpan = document.getElementById(`${trait}-value`);
        slider.addEventListener('input', () => {
            valueSpan.textContent = slider.value;
        });
    });
});

// Floating leaves animation
function initializeFloatingLeaves() {
    const leavesContainer = document.getElementById('floatingLeaves');

    function createLeaf() {
        const leaf = document.createElement('div');
        leaf.className = 'leaf';
        leaf.style.left = Math.random() * 100 + '%';
        leaf.style.animationDelay = Math.random() * 8 + 's';
        leaf.style.animationDuration = (8 + Math.random() * 4) + 's';
        leavesContainer.appendChild(leaf);

        // Remove leaf after animation
        setTimeout(() => {
            if (leaf.parentNode) {
                leaf.parentNode.removeChild(leaf);
            }
        }, 12000);
    }

    // Create leaves periodically
    setInterval(createLeaf, 2000);

    // Create initial leaves
    for (let i = 0; i < 3; i++) {
        setTimeout(createLeaf, i * 1000);
    }
}

// Section navigation
function showSection(sectionName) {
    // Hide all sections
    document.querySelectorAll('[id$="-section"]').forEach(section => {
        section.classList.add('hidden');
    });

    // Show selected section
    document.getElementById(sectionName + '-section').classList.remove('hidden');
    currentSection = sectionName;

    // Update nav buttons
    document.querySelectorAll('.nav-buttons .btn').forEach(btn => {
        btn.style.opacity = '0.7';
    });
    event.target.style.opacity = '1';

    // Load section-specific data
    if (sectionName === 'characters') {
        loadCharacters();
    } else if (sectionName === 'dialog') {
        loadCharactersForDialog();
    } else if (sectionName === 'projects') {
        loadProjects();
    }
}

// API helper functions
async function apiCall(endpoint, options = {}) {
    try {
        const response = await fetch(`/api/v1${endpoint}`, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });

        if (!response.ok) {
            throw new Error(`API Error: ${response.status}`);
        }

        return await response.json();
    } catch (error) {
        console.error('API Call Error:', error);
        showMessage('Failed to communicate with the jungle spirits! 🌿', 'error');
        throw error;
    }
}

// Message system
function showMessage(message, type = 'success') {
    const container = document.getElementById('messages-container');
    const messageEl = document.createElement('div');
    messageEl.className = `message message-${type}`;
    messageEl.textContent = message;

    container.appendChild(messageEl);

    // Auto-remove after 5 seconds
    setTimeout(() => {
        if (messageEl.parentNode) {
            messageEl.parentNode.removeChild(messageEl);
        }
    }, 5000);
}

// Character management
async function loadCharacters() {
    try {
        const response = await apiCall('/characters');
        characters = response.characters || [];
        displayCharacters();
    } catch (error) {
        console.error('Failed to load characters:', error);
        showMessage('Failed to load jungle characters! 🐒', 'error');
    }
}

function displayCharacters() {
    const grid = document.getElementById('character-grid');

    if (characters.length === 0) {
        grid.innerHTML = `
            <div style="grid-column: 1 / -1; text-align: center; padding: 3rem; color: var(--jungle-secondary);">
                <div style="font-size: 4rem; margin-bottom: 1rem;">🌿</div>
                <h3>No characters in the jungle yet!</h3>
                <p>Create your first character to start the adventure!</p>
                <button class="btn btn-primary" onclick="showCharacterCreation()" style="margin-top: 1rem;">
                    🐒 Create First Character
                </button>
            </div>
        `;
        return;
    }

    grid.innerHTML = characters.map(character => `
        <div class="character-card" onclick="selectCharacter('${character.id}')">
            <div class="character-avatar">
                ${getCharacterEmoji(character.personality_traits)}
            </div>
            <div class="character-name">${character.name}</div>
            <p style="text-align: center; color: var(--jungle-secondary); font-size: 0.9rem; margin: 0.5rem 0;">
                ${character.background_story ? character.background_story.substring(0, 80) + '...' : 'A jungle adventurer'}
            </p>
            <div class="character-traits">
                ${Object.entries(character.personality_traits || {}).map(([trait, value]) => 
                    `<span class="trait-badge">${trait}: ${value}</span>`
                ).join('')}
            </div>
        </div>
    `).join('');
}

function getCharacterEmoji(traits) {
    if (!traits) return '🐒';

    if (traits.brave > 0.7) return '🦁';
    if (traits.humorous > 0.7) return '🐒';
    if (traits.mysterious > 0.7) return '🐆';
    if (traits.wise > 0.7) return '🦉';
    if (traits.cheerful > 0.7) return '🦜';
    if (traits.tough > 0.7) return '🦏';

    return '🐒';
}

function showCharacterCreation() {
    document.getElementById('character-creation-form').classList.remove('hidden');
    document.getElementById('characters-list').classList.add('hidden');
}

function hideCharacterCreation() {
    document.getElementById('character-creation-form').classList.add('hidden');
    document.getElementById('characters-list').classList.remove('hidden');
}

async function createCharacter() {
    const name = document.getElementById('character-name').value;
    const background = document.getElementById('character-background').value;
    const brave = parseFloat(document.getElementById('trait-brave').value);
    const humorous = parseFloat(document.getElementById('trait-humorous').value);
    const loyal = parseFloat(document.getElementById('trait-loyal').value);
    const voicePitch = document.getElementById('voice-pitch').value;

    if (!name.trim()) {
        showMessage('Character name is required! 🌿', 'error');
        return;
    }

    try {
        const characterData = {
            name: name.trim(),
            background_story: background.trim(),
            personality_traits: {
                brave,
                humorous,
                loyal
            },
            voice_profile: {
                pitch: voicePitch
            }
        };

        await apiCall('/characters', {
            method: 'POST',
            body: JSON.stringify(characterData)
        });

        showMessage(`🌿 ${name} joined the jungle adventure! 🎮`, 'success');

        // Reset form
        document.getElementById('character-name').value = '';
        document.getElementById('character-background').value = '';
        document.getElementById('trait-brave').value = '0.5';
        document.getElementById('trait-humorous').value = '0.5';
        document.getElementById('trait-loyal').value = '0.5';
        document.getElementById('voice-pitch').value = 'medium';

        // Update slider displays
        document.getElementById('brave-value').textContent = '0.5';
        document.getElementById('humorous-value').textContent = '0.5';
        document.getElementById('loyal-value').textContent = '0.5';

        hideCharacterCreation();
        loadCharacters();

    } catch (error) {
        showMessage('Failed to create character! 🐒', 'error');
    }
}

function selectCharacter(characterId) {
    // For now, just show character details
    const character = characters.find(c => c.id === characterId);
    if (character) {
        showMessage(`Selected ${character.name} for the adventure! 🌿`, 'success');
    }
}

// Dialog generation
async function loadCharactersForDialog() {
    if (characters.length === 0) {
        await loadCharacters();
    }

    const select = document.getElementById('dialog-character');
    select.innerHTML = '<option value="">Choose a character...</option>' + 
        characters.map(character => 
            `<option value="${character.id}">${character.name}</option>`
        ).join('');
}

async function generateDialog() {
    const characterId = document.getElementById('dialog-character').value;
    const sceneContext = document.getElementById('scene-context').value;
    const emotion = document.getElementById('character-emotion').value;

    if (!characterId || !sceneContext.trim()) {
        showMessage('Please select a character and describe the scene! 🎭', 'error');
        return;
    }

    const outputEl = document.getElementById('dialog-output');
    const metadataEl = document.getElementById('dialog-metadata');

    // Show loading state
    outputEl.innerHTML = '<div class="dialog-text generating-text">🌿 The jungle spirits are crafting your dialog... <div class="loading"></div></div>';

    try {
        const dialogData = {
            character_id: characterId,
            scene_context: sceneContext.trim(),
            emotion_state: emotion
        };

        const response = await apiCall('/dialog/generate', {
            method: 'POST',
            body: JSON.stringify(dialogData)
        });

        // Display generated dialog
        outputEl.innerHTML = `<div class="dialog-text">"${response.dialog}"</div>`;

        // Show metadata
        document.getElementById('generated-emotion').textContent = response.emotion;
        document.getElementById('consistency-score').textContent = 
            (response.character_consistency_score * 100).toFixed(1) + '%';
        metadataEl.classList.remove('hidden');

        showMessage('🌿 Dialog generated successfully! 🎮', 'success');

    } catch (error) {
        outputEl.innerHTML = '<div class="dialog-text dialog-placeholder">🌿 Failed to generate dialog. Try again! 🎮</div>';
        metadataEl.classList.add('hidden');
        showMessage('Failed to generate dialog! 💬', 'error');
    }
}

// Project management
async function loadProjects() {
    try {
        const response = await apiCall('/projects');
        projects = response.projects || [];
        displayProjects();
    } catch (error) {
        console.error('Failed to load projects:', error);
        showMessage('Failed to load game projects! 🎮', 'error');
    }
}

function displayProjects() {
    const grid = document.getElementById('projects-grid');

    if (projects.length === 0) {
        grid.innerHTML = `
            <div style="grid-column: 1 / -1; text-align: center; padding: 3rem; color: var(--jungle-secondary);">
                <div style="font-size: 4rem; margin-bottom: 1rem;">🎮</div>
                <h3>No game projects yet!</h3>
                <p>Create your first project to organize your jungle adventure!</p>
                <button class="btn btn-primary" onclick="showProjectCreation()" style="margin-top: 1rem;">
                    🌿 Create First Project
                </button>
            </div>
        `;
        return;
    }

    grid.innerHTML = projects.map(project => `
        <div class="character-card" onclick="selectProject('${project.id}')">
            <div class="character-avatar">🎮</div>
            <div class="character-name">${project.name}</div>
            <p style="text-align: center; color: var(--jungle-secondary); font-size: 0.9rem; margin: 0.5rem 0;">
                ${project.description ? project.description.substring(0, 80) + '...' : 'A jungle adventure game'}
            </p>
            <div class="character-traits">
                <span class="trait-badge">Characters: ${project.characters ? project.characters.length : 0}</span>
                <span class="trait-badge">Format: ${project.export_format || 'JSON'}</span>
            </div>
        </div>
    `).join('');
}

function showProjectCreation() {
    document.getElementById('project-creation-form').classList.remove('hidden');
    document.getElementById('projects-list').classList.add('hidden');
}

function hideProjectCreation() {
    document.getElementById('project-creation-form').classList.add('hidden');
    document.getElementById('projects-list').classList.remove('hidden');
}

async function createProject() {
    const name = document.getElementById('project-name').value;
    const description = document.getElementById('project-description').value;

    if (!name.trim()) {
        showMessage('Project name is required! 🎮', 'error');
        return;
    }

    try {
        const projectData = {
            name: name.trim(),
            description: description.trim(),
            settings: {
                theme: 'jungle-adventure'
            }
        };

        await apiCall('/projects', {
            method: 'POST',
            body: JSON.stringify(projectData)
        });

        showMessage(`🌿 ${name} project created! Ready for adventure! 🎮`, 'success');

        // Reset form
        document.getElementById('project-name').value = '';
        document.getElementById('project-description').value = '';

        hideProjectCreation();
        loadProjects();

    } catch (error) {
        showMessage('Failed to create project! 🎮', 'error');
    }
}

function selectProject(projectId) {
    const project = projects.find(p => p.id === projectId);
    if (project) {
        showMessage(`Selected ${project.name} project! 🌿`, 'success');
    }
}

// Keyboard shortcuts
document.addEventListener('keydown', function(e) {
    if (e.ctrlKey || e.metaKey) {
        switch(e.key) {
            case '1':
                e.preventDefault();
                showSection('characters');
                break;
            case '2':
                e.preventDefault();
                showSection('dialog');
                break;
            case '3':
                e.preventDefault();
                showSection('projects');
                break;
        }
    }
});

console.log('🌿 Game Dialog Generator initialized! Welcome to the jungle adventure! 🎮');

Object.assign(window, {
    showSection,
    showCharacterCreation,
    createCharacter,
    hideCharacterCreation,
    generateDialog,
    showProjectCreation,
    createProject,
    hideProjectCreation,
});
