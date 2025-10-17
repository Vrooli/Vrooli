import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

if (typeof window !== 'undefined' && window.parent !== window && !window.__personalRelationshipBridgeInitialized) {
    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[PersonalRelationshipManager] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'personal-relationship-manager' });
    window.__personalRelationshipBridgeInitialized = true;
}

// API Configuration
const API_BASE_URL = window.location.hostname === 'localhost' 
    ? 'http://localhost:39001/api' 
    : '/api';

// State Management
const state = {
    currentView: 'dashboard',
    contacts: [],
    interactions: [],
    gifts: [],
    reminders: [],
    selectedContact: null
};

// Initialize App
document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    loadDashboard();
    fetchContacts();
});

// Event Listeners
function initializeEventListeners() {
    // Navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', (e) => {
            const view = e.currentTarget.dataset.view;
            switchView(view);
        });
    });
    
    // Add Person Button
    document.querySelector('.add-person-btn').addEventListener('click', () => {
        openModal('add-person-modal');
    });
    
    // Quick Actions
    document.getElementById('log-interaction')?.addEventListener('click', () => {
        openModal('log-interaction-modal');
        populateContactSelect();
    });
    
    document.getElementById('find-gift')?.addEventListener('click', () => {
        switchView('gifts');
    });
    
    document.getElementById('set-reminder')?.addEventListener('click', () => {
        alert('Reminder feature coming soon! 🔔');
    });
    
    // Modal Close Buttons
    document.querySelectorAll('.modal-close, .modal-cancel').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const modal = e.target.closest('.modal');
            closeModal(modal.id);
        });
    });
    
    // Form Submissions
    document.getElementById('add-person-form')?.addEventListener('submit', handleAddPerson);
    document.getElementById('log-interaction-form')?.addEventListener('submit', handleLogInteraction);
    document.getElementById('gift-finder-form')?.addEventListener('submit', handleFindGifts);
    
    // Search and Filters
    document.querySelector('.search-input')?.addEventListener('input', handleSearch);
    document.querySelectorAll('.chip').forEach(chip => {
        chip.addEventListener('click', handleFilter);
    });
}

// View Management
function switchView(viewName) {
    // Update navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.dataset.view === viewName) {
            item.classList.add('active');
        }
    });
    
    // Update views
    document.querySelectorAll('.view').forEach(view => {
        view.classList.remove('active');
    });
    document.getElementById(`${viewName}-view`)?.classList.add('active');
    
    // Load view-specific data
    state.currentView = viewName;
    switch(viewName) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'contacts':
            loadContacts();
            break;
        case 'upcoming':
            loadUpcomingEvents();
            break;
        case 'gifts':
            loadGiftIdeas();
            break;
        case 'memories':
            loadMemories();
            break;
    }
}

// Dashboard Functions
async function loadDashboard() {
    // Load today's reminders
    loadTodayReminders();
    
    // Load recent interactions
    loadRecentInteractions();
    
    // Load insights
    loadRelationshipInsights();
}

function loadTodayReminders() {
    const remindersContainer = document.getElementById('today-reminders');
    if (!remindersContainer) return;
    
    // Mock data for demonstration
    const mockReminders = [
        { icon: '🎂', title: "Sarah's Birthday!", description: "Don't forget to call her today" },
        { icon: '💐', title: "Mom's Anniversary", description: "Send flowers - she loves tulips" }
    ];
    
    remindersContainer.innerHTML = mockReminders.map(reminder => `
        <div class="reminder-item">
            <div class="reminder-icon">${reminder.icon}</div>
            <div class="reminder-details">
                <strong>${reminder.title}</strong>
                <p>${reminder.description}</p>
            </div>
        </div>
    `).join('');
}

function loadRecentInteractions() {
    const interactionsContainer = document.getElementById('recent-interactions');
    if (!interactionsContainer) return;
    
    // Mock data
    const mockInteractions = [
        { name: 'Dad', type: 'call', time: '2 days ago', description: 'Talked about the fishing trip' },
        { name: 'Best Friend', type: 'visit', time: '1 week ago', description: 'Coffee and catch-up' },
        { name: 'Sister', type: 'message', time: '3 days ago', description: 'Shared photos from vacation' }
    ];
    
    interactionsContainer.innerHTML = mockInteractions.map(interaction => `
        <div class="timeline-item">
            <div style="flex: 1;">
                <strong>${interaction.name}</strong> - ${interaction.type}
                <p style="font-size: 0.9rem; color: var(--text-secondary); margin-top: 0.25rem;">
                    ${interaction.description}
                </p>
                <small style="color: var(--text-light);">${interaction.time}</small>
            </div>
        </div>
    `).join('');
}

function loadRelationshipInsights() {
    // Insights are already in HTML for now
}

// Contacts Functions
async function fetchContacts() {
    try {
        const response = await fetch(`${API_BASE_URL}/contacts`);
        if (response.ok) {
            state.contacts = await response.json();
        }
    } catch (error) {
        console.error('Error fetching contacts:', error);
        // Use mock data if API fails
        state.contacts = getMockContacts();
    }
}

function getMockContacts() {
    return [
        {
            id: 1,
            name: 'Sarah Johnson',
            nickname: 'SJ',
            relationship_type: 'friend',
            birthday: '1995-03-15',
            interests: ['reading', 'yoga', 'coffee'],
            tags: ['college', 'book club']
        },
        {
            id: 2,
            name: 'Mom',
            nickname: 'Mom',
            relationship_type: 'family',
            birthday: '1965-07-22',
            interests: ['gardening', 'cooking', 'photography'],
            tags: ['family', 'weekly calls']
        },
        {
            id: 3,
            name: 'Alex Chen',
            nickname: 'AC',
            relationship_type: 'colleague',
            birthday: '1990-11-08',
            interests: ['tech', 'gaming', 'basketball'],
            tags: ['work', 'mentor']
        }
    ];
}

function loadContacts() {
    const contactsGrid = document.getElementById('contacts-grid');
    if (!contactsGrid) return;
    
    contactsGrid.innerHTML = state.contacts.map(contact => `
        <div class="contact-card" onclick="showContactDetail(${contact.id})">
            <div class="contact-header">
                <div class="contact-avatar">
                    ${contact.nickname ? contact.nickname.substring(0, 2).toUpperCase() : contact.name.substring(0, 2).toUpperCase()}
                </div>
                <div class="contact-info">
                    <h4>${contact.name}</h4>
                    <div class="contact-meta">
                        <span class="meta-tag">${contact.relationship_type}</span>
                        ${contact.birthday ? `<span class="meta-tag">🎂 ${formatDate(contact.birthday)}</span>` : ''}
                    </div>
                </div>
            </div>
            ${contact.interests ? `
                <div style="margin: 1rem 0;">
                    <small style="color: var(--text-secondary);">Interests:</small>
                    <div style="margin-top: 0.5rem;">
                        ${contact.interests.map(interest => 
                            `<span class="meta-tag">${interest}</span>`
                        ).join(' ')}
                    </div>
                </div>
            ` : ''}
            <div class="contact-actions">
                <button class="contact-action-btn" onclick="event.stopPropagation(); callContact(${contact.id})">
                    📞 Call
                </button>
                <button class="contact-action-btn" onclick="event.stopPropagation(); messageContact(${contact.id})">
                    💬 Message
                </button>
                <button class="contact-action-btn" onclick="event.stopPropagation(); giftForContact(${contact.id})">
                    🎁 Gift
                </button>
            </div>
        </div>
    `).join('');
}

function showContactDetail(contactId) {
    const contact = state.contacts.find(c => c.id === contactId);
    if (!contact) return;
    
    state.selectedContact = contact;
    const detailContent = document.getElementById('contact-detail-content');
    
    detailContent.innerHTML = `
        <div style="display: flex; align-items: center; gap: 2rem; margin-bottom: 2rem;">
            <div class="contact-avatar" style="width: 100px; height: 100px; font-size: 2rem;">
                ${contact.nickname ? contact.nickname.substring(0, 2).toUpperCase() : contact.name.substring(0, 2).toUpperCase()}
            </div>
            <div>
                <h2>${contact.name}</h2>
                <p style="color: var(--text-secondary);">${contact.relationship_type}</p>
                ${contact.birthday ? `<p>🎂 Birthday: ${formatDate(contact.birthday)}</p>` : ''}
            </div>
        </div>
        
        <div style="display: grid; gap: 1.5rem;">
            <div class="card">
                <h3>Contact Information</h3>
                <p>📧 ${contact.email || 'No email added'}</p>
                <p>📱 ${contact.phone || 'No phone added'}</p>
                <p>🏠 ${contact.address || 'No address added'}</p>
            </div>
            
            <div class="card">
                <h3>Interests & Preferences</h3>
                <div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
                    ${contact.interests ? contact.interests.map(interest => 
                        `<span class="chip">${interest}</span>`
                    ).join('') : 'No interests added'}
                </div>
            </div>
            
            <div class="card">
                <h3>Notes</h3>
                <p>${contact.notes || 'No notes added yet. Add some memories!'}</p>
            </div>
            
            <div class="form-actions">
                <button class="btn-secondary" onclick="editContact(${contact.id})">Edit</button>
                <button class="btn-primary" onclick="closeModal('contact-detail-modal')">Close</button>
            </div>
        </div>
    `;
    
    openModal('contact-detail-modal');
}

// Contact Actions
function callContact(contactId) {
    const contact = state.contacts.find(c => c.id === contactId);
    alert(`Calling ${contact.name}... 📞`);
}

function messageContact(contactId) {
    const contact = state.contacts.find(c => c.id === contactId);
    alert(`Opening message to ${contact.name}... 💬`);
}

function giftForContact(contactId) {
    const contact = state.contacts.find(c => c.id === contactId);
    state.selectedContact = contact;
    switchView('gifts');
    // Pre-fill the gift finder form
    setTimeout(() => {
        const select = document.getElementById('gift-person');
        if (select) {
            select.value = contactId;
        }
    }, 100);
}

function editContact(contactId) {
    alert('Edit feature coming soon! ✏️');
}

// Upcoming Events
function loadUpcomingEvents() {
    const eventsContainer = document.getElementById('events-timeline');
    if (!eventsContainer) return;
    
    // Mock upcoming events
    const events = [
        { date: 'Tomorrow', icon: '🎂', title: "John's Birthday", action: 'Send a card' },
        { date: 'March 20', icon: '💑', title: 'Anniversary with Partner', action: 'Book restaurant' },
        { date: 'March 25', icon: '🎉', title: "Sister's Graduation", action: 'Buy gift' },
        { date: 'April 1', icon: '🌸', title: "Mom's Birthday", action: 'Order flowers' }
    ];
    
    eventsContainer.innerHTML = `
        <div class="events-list">
            ${events.map(event => `
                <div class="card" style="margin-bottom: 1rem;">
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <div style="display: flex; gap: 1rem; align-items: center;">
                            <div style="font-size: 2rem;">${event.icon}</div>
                            <div>
                                <h4>${event.title}</h4>
                                <p style="color: var(--text-secondary);">${event.date}</p>
                                <small style="color: var(--primary-color);">💡 ${event.action}</small>
                            </div>
                        </div>
                        <button class="btn-primary">Set Reminder</button>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
}

// Gift Ideas
function loadGiftIdeas() {
    populateGiftPersonSelect();
}

function populateGiftPersonSelect() {
    const select = document.getElementById('gift-person');
    if (!select) return;
    
    select.innerHTML = '<option value="">Select a person...</option>' +
        state.contacts.map(contact => 
            `<option value="${contact.id}">${contact.name}</option>`
        ).join('');
}

async function handleFindGifts(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const contactId = formData.get('gift-person');
    const occasion = formData.get('gift-occasion');
    const budget = formData.get('gift-budget');
    
    if (!contactId) {
        alert('Please select a person');
        return;
    }
    
    const contact = state.contacts.find(c => c.id == contactId);
    
    // Show loading state
    const suggestionsContainer = document.getElementById('gift-suggestions');
    suggestionsContainer.innerHTML = '<div class="card"><p>Finding perfect gifts... 🎁✨</p></div>';
    
    try {
        const response = await fetch(`${API_BASE_URL}/gifts/suggest`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                contact_id: contactId,
                name: contact.name,
                interests: contact.interests?.join(', ') || 'general',
                occasion: occasion || 'any',
                budget: budget || '50-100'
            })
        });
        
        if (response.ok) {
            const suggestions = await response.json();
            displayGiftSuggestions(suggestions);
        } else {
            throw new Error('Failed to get suggestions');
        }
    } catch (error) {
        console.error('Error getting gift suggestions:', error);
        // Show mock suggestions
        displayMockGiftSuggestions(contact);
    }
}

function displayGiftSuggestions(suggestions) {
    const container = document.getElementById('gift-suggestions');
    
    container.innerHTML = suggestions.map(gift => `
        <div class="gift-card">
            <div class="gift-header">
                <h4>${gift.name}</h4>
                <span class="gift-price">$${gift.price}</span>
            </div>
            <p class="gift-description">${gift.description}</p>
            <div class="gift-actions">
                <button class="gift-action-btn" onclick="saveGiftIdea(${JSON.stringify(gift).replace(/"/g, '&quot;')})">
                    💾 Save Idea
                </button>
                <button class="gift-action-btn" onclick="markAsPurchased(${JSON.stringify(gift).replace(/"/g, '&quot;')})">
                    ✅ Mark Purchased
                </button>
            </div>
        </div>
    `).join('');
}

function displayMockGiftSuggestions(contact) {
    const mockGifts = [
        {
            name: 'Personalized Journal',
            price: 35,
            description: 'A beautiful leather-bound journal with their initials embossed on the cover. Perfect for capturing thoughts and memories.',
            store: 'Stationery store'
        },
        {
            name: 'Artisan Coffee Subscription',
            price: 25,
            description: 'Monthly delivery of carefully selected coffee beans from small roasters around the world.',
            store: 'Online subscription'
        },
        {
            name: 'Indoor Plant Collection',
            price: 45,
            description: 'A set of low-maintenance succulents in decorative pots to brighten their space.',
            store: 'Garden center'
        }
    ];
    
    displayGiftSuggestions(mockGifts);
}

function saveGiftIdea(gift) {
    alert(`Gift idea "${gift.name}" saved! 💾`);
}

function markAsPurchased(gift) {
    alert(`Marked "${gift.name}" as purchased! ✅`);
}

// Memories
function loadMemories() {
    const memoriesContainer = document.getElementById('memories-gallery');
    if (!memoriesContainer) return;
    
    memoriesContainer.innerHTML = `
        <div class="card" style="text-align: center; padding: 3rem;">
            <div style="font-size: 4rem; margin-bottom: 1rem;">📸</div>
            <h3>Memory Lane Coming Soon!</h3>
            <p style="color: var(--text-secondary);">
                Soon you'll be able to add photos and memories with your loved ones.
            </p>
        </div>
    `;
}

// Form Handlers
async function handleAddPerson(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const personData = {
        name: formData.get('name'),
        nickname: formData.get('nickname'),
        relationship_type: formData.get('relationship_type'),
        birthday: formData.get('birthday'),
        anniversary: formData.get('anniversary'),
        interests: formData.get('interests')?.split(',').map(i => i.trim()),
        notes: formData.get('notes')
    };
    
    try {
        const response = await fetch(`${API_BASE_URL}/contacts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(personData)
        });
        
        if (response.ok) {
            const newContact = await response.json();
            state.contacts.push(newContact);
            closeModal('add-person-modal');
            e.target.reset();
            
            // Show success message
            showNotification(`${personData.name} added to your circle! 💕`);
            
            // Reload contacts if on that view
            if (state.currentView === 'contacts') {
                loadContacts();
            }
        }
    } catch (error) {
        console.error('Error adding contact:', error);
        // Add to local state anyway for demo
        const newContact = { ...personData, id: Date.now() };
        state.contacts.push(newContact);
        closeModal('add-person-modal');
        e.target.reset();
        showNotification(`${personData.name} added to your circle! 💕`);
        
        if (state.currentView === 'contacts') {
            loadContacts();
        }
    }
}

async function handleLogInteraction(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const interactionData = {
        contact_id: formData.get('contact_id'),
        interaction_type: formData.get('interaction_type'),
        description: formData.get('description'),
        sentiment: formData.get('sentiment'),
        interaction_date: new Date().toISOString()
    };
    
    try {
        const response = await fetch(`${API_BASE_URL}/interactions`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(interactionData)
        });
        
        if (response.ok) {
            closeModal('log-interaction-modal');
            e.target.reset();
            showNotification('Interaction logged! 💬');
            loadRecentInteractions();
        }
    } catch (error) {
        console.error('Error logging interaction:', error);
        // Still close and show success for demo
        closeModal('log-interaction-modal');
        e.target.reset();
        showNotification('Interaction logged! 💬');
    }
}

function populateContactSelect() {
    const select = document.querySelector('#log-interaction-form select[name="contact_id"]');
    if (!select) return;
    
    select.innerHTML = '<option value="">Select person...</option>' +
        state.contacts.map(contact => 
            `<option value="${contact.id}">${contact.name}</option>`
        ).join('');
}

// Search and Filter
function handleSearch(e) {
    const searchTerm = e.target.value.toLowerCase();
    const filteredContacts = state.contacts.filter(contact => 
        contact.name.toLowerCase().includes(searchTerm) ||
        contact.notes?.toLowerCase().includes(searchTerm) ||
        contact.interests?.some(i => i.toLowerCase().includes(searchTerm)) ||
        contact.tags?.some(t => t.toLowerCase().includes(searchTerm))
    );
    
    displayFilteredContacts(filteredContacts);
}

function handleFilter(e) {
    // Remove active from all chips
    document.querySelectorAll('.chip').forEach(chip => chip.classList.remove('active'));
    e.target.classList.add('active');
    
    const filterType = e.target.textContent.toLowerCase();
    
    if (filterType === 'all') {
        loadContacts();
        return;
    }
    
    const filteredContacts = state.contacts.filter(contact => 
        contact.relationship_type === filterType
    );
    
    displayFilteredContacts(filteredContacts);
}

function displayFilteredContacts(contacts) {
    const contactsGrid = document.getElementById('contacts-grid');
    if (!contactsGrid) return;
    
    if (contacts.length === 0) {
        contactsGrid.innerHTML = `
            <div class="card" style="grid-column: 1 / -1; text-align: center; padding: 2rem;">
                <p>No contacts found matching your search.</p>
            </div>
        `;
        return;
    }
    
    contactsGrid.innerHTML = contacts.map(contact => `
        <div class="contact-card" onclick="showContactDetail(${contact.id})">
            <!-- Same contact card HTML as in loadContacts -->
            <div class="contact-header">
                <div class="contact-avatar">
                    ${contact.nickname ? contact.nickname.substring(0, 2).toUpperCase() : contact.name.substring(0, 2).toUpperCase()}
                </div>
                <div class="contact-info">
                    <h4>${contact.name}</h4>
                    <div class="contact-meta">
                        <span class="meta-tag">${contact.relationship_type}</span>
                    </div>
                </div>
            </div>
            <div class="contact-actions">
                <button class="contact-action-btn" onclick="event.stopPropagation(); callContact(${contact.id})">
                    📞 Call
                </button>
                <button class="contact-action-btn" onclick="event.stopPropagation(); messageContact(${contact.id})">
                    💬 Message
                </button>
                <button class="contact-action-btn" onclick="event.stopPropagation(); giftForContact(${contact.id})">
                    🎁 Gift
                </button>
            </div>
        </div>
    `).join('');
}

// Modal Functions
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.add('active');
        document.body.style.overflow = 'hidden';
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('active');
        document.body.style.overflow = 'auto';
    }
}

// Notification System
function showNotification(message) {
    const notification = document.createElement('div');
    notification.className = 'notification';
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 2rem;
        right: 2rem;
        background: linear-gradient(135deg, var(--primary-color), var(--primary-dark));
        color: white;
        padding: 1rem 1.5rem;
        border-radius: var(--radius-md);
        box-shadow: var(--shadow-lg);
        z-index: 2000;
        animation: slideInRight 0.3s ease;
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.animation = 'slideOutRight 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Utility Functions
function formatDate(dateString) {
    const date = new Date(dateString);
    const options = { month: 'short', day: 'numeric' };
    return date.toLocaleDateString('en-US', options);
}

// Add animation keyframes
const style = document.createElement('style');
style.textContent = `
    @keyframes slideInRight {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    
    @keyframes slideOutRight {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);
