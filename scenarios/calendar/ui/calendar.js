// Calendar - Universal Scheduling Intelligence
// Client-side JavaScript for professional calendar interface

class CalendarApp {
    constructor() {
        this.currentDate = new Date();
        this.currentView = 'week';
        this.events = [];
        // Get API URL from environment or derive from current origin
        this.apiBaseUrl = window.CALENDAR_API_URL || window.location.origin.replace(':35001', ':15001');
        this.authToken = localStorage.getItem('calendar_auth_token');
        
        this.init();
    }

    async init() {
        await this.checkAuth();
        this.setupEventListeners();
        this.renderCurrentView();
        await this.loadEvents();
        this.startPolling();
    }

    // Authentication
    async checkAuth() {
        if (!this.authToken) {
            // For development, use a mock token
            // In production, redirect to scenario-authenticator
            this.authToken = 'mock-token-for-development';
            localStorage.setItem('calendar_auth_token', this.authToken);
        }

        try {
            // Validate token with API
            const response = await this.apiCall('/health');
            if (response.status === 'healthy') {
                this.showToast('Calendar loaded successfully', 'success');
            }
        } catch (error) {
            console.error('Auth check failed:', error);
            this.showToast('Unable to connect to calendar service', 'error');
        }
    }

    // API calls
    async apiCall(endpoint, options = {}) {
        const url = `${this.apiBaseUrl}/api/v1${endpoint}`;
        const headers = {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${this.authToken}`,
            ...options.headers
        };

        try {
            const response = await fetch(url, {
                ...options,
                headers
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API call failed:', error);
            throw error;
        }
    }

    // Event listeners
    setupEventListeners() {
        // View switchers
        document.getElementById('week-view-btn').addEventListener('click', () => this.switchView('week'));
        document.getElementById('month-view-btn').addEventListener('click', () => this.switchView('month'));
        document.getElementById('agenda-view-btn').addEventListener('click', () => this.switchView('agenda'));

        // Navigation
        document.getElementById('prev-btn').addEventListener('click', () => this.navigate(-1));
        document.getElementById('next-btn').addEventListener('click', () => this.navigate(1));
        document.getElementById('today-nav-btn').addEventListener('click', () => this.goToToday());
        document.getElementById('today-btn').addEventListener('click', () => this.goToToday());

        // Modals
        document.getElementById('new-event-btn').addEventListener('click', () => this.showModal('new-event-modal'));
        document.getElementById('chat-btn').addEventListener('click', () => this.showModal('chat-modal'));

        // Modal close buttons
        document.querySelectorAll('[data-modal]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const modalId = e.target.getAttribute('data-modal') || e.target.closest('[data-modal]').getAttribute('data-modal');
                this.hideModal(modalId);
            });
        });

        // Form submissions
        document.getElementById('new-event-form').addEventListener('submit', this.handleCreateEvent.bind(this));
        document.getElementById('chat-send-btn').addEventListener('click', this.handleChatSend.bind(this));
        document.getElementById('chat-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.handleChatSend();
        });

        // Search
        document.getElementById('search-input').addEventListener('input', this.debounce(this.handleSearch.bind(this), 300));

        // Keyboard shortcuts
        document.addEventListener('keydown', this.handleKeyboard.bind(this));

        // Click outside modal to close
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.hideModal(e.target.id);
            }
        });
    }

    // View management
    switchView(view) {
        if (this.currentView === view) return;

        // Update button states
        document.querySelectorAll('.view-switcher .btn').forEach(btn => btn.classList.remove('active'));
        document.getElementById(`${view}-view-btn`).classList.add('active');

        // Update view content
        document.querySelectorAll('.calendar-view').forEach(view => view.classList.remove('active'));
        document.getElementById(`${view}-view`).classList.add('active');

        this.currentView = view;
        this.renderCurrentView();
    }

    navigate(direction) {
        const date = new Date(this.currentDate);
        
        switch (this.currentView) {
            case 'week':
                date.setDate(date.getDate() + (direction * 7));
                break;
            case 'month':
                date.setMonth(date.getMonth() + direction);
                break;
            case 'agenda':
                date.setDate(date.getDate() + (direction * 7));
                break;
        }

        this.currentDate = date;
        this.renderCurrentView();
    }

    goToToday() {
        this.currentDate = new Date();
        this.renderCurrentView();
    }

    // Rendering
    renderCurrentView() {
        this.updatePeriodTitle();
        
        switch (this.currentView) {
            case 'week':
                this.renderWeekView();
                break;
            case 'month':
                this.renderMonthView();
                break;
            case 'agenda':
                this.renderAgendaView();
                break;
        }
    }

    updatePeriodTitle() {
        const title = document.getElementById('current-period');
        
        switch (this.currentView) {
            case 'week':
                const weekStart = this.getWeekStart(this.currentDate);
                const weekEnd = new Date(weekStart);
                weekEnd.setDate(weekEnd.getDate() + 6);
                
                if (weekStart.getMonth() === weekEnd.getMonth()) {
                    title.textContent = `${weekStart.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })} ${weekStart.getDate()}-${weekEnd.getDate()}`;
                } else {
                    title.textContent = `${weekStart.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} - ${weekEnd.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}`;
                }
                break;
                
            case 'month':
                title.textContent = this.currentDate.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
                break;
                
            case 'agenda':
                title.textContent = `Agenda - ${this.currentDate.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}`;
                break;
        }
    }

    renderWeekView() {
        const weekStart = this.getWeekStart(this.currentDate);
        
        // Render day headers
        const dayHeaders = document.getElementById('day-headers');
        dayHeaders.innerHTML = '';
        
        for (let i = 0; i < 7; i++) {
            const date = new Date(weekStart);
            date.setDate(date.getDate() + i);
            
            const header = document.createElement('div');
            header.className = 'day-header';
            if (this.isToday(date)) header.classList.add('today');
            
            header.innerHTML = `
                <div class="day-name">${date.toLocaleDateString('en-US', { weekday: 'short' })}</div>
                <div class="day-number">${date.getDate()}</div>
            `;
            
            dayHeaders.appendChild(header);
        }

        // Render time column
        const timeColumn = document.getElementById('time-column');
        timeColumn.innerHTML = '';
        
        for (let hour = 0; hour < 24; hour++) {
            const timeSlot = document.createElement('div');
            timeSlot.className = 'time-slot';
            
            const time = new Date();
            time.setHours(hour, 0, 0, 0);
            timeSlot.textContent = time.toLocaleTimeString('en-US', { 
                hour: 'numeric', 
                hour12: true 
            });
            
            timeColumn.appendChild(timeSlot);
        }

        // Render week grid
        const weekGrid = document.getElementById('week-grid');
        weekGrid.innerHTML = '';
        
        for (let i = 0; i < 7; i++) {
            const date = new Date(weekStart);
            date.setDate(date.getDate() + i);
            
            const dayColumn = document.createElement('div');
            dayColumn.className = 'day-column';
            dayColumn.dataset.date = date.toISOString().split('T')[0];
            
            // Add hour lines
            for (let hour = 0; hour < 24; hour++) {
                const hourLine = document.createElement('div');
                hourLine.style.position = 'absolute';
                hourLine.style.top = `${hour * 60}px`;
                hourLine.style.left = '0';
                hourLine.style.right = '0';
                hourLine.style.height = '1px';
                hourLine.style.backgroundColor = 'var(--border-light)';
                dayColumn.appendChild(hourLine);
            }
            
            weekGrid.appendChild(dayColumn);
        }

        // Render events
        this.renderWeekEvents();
    }

    renderWeekEvents() {
        const weekStart = this.getWeekStart(this.currentDate);
        const weekEnd = new Date(weekStart);
        weekEnd.setDate(weekEnd.getDate() + 7);

        // Clear existing events
        document.querySelectorAll('.event-block').forEach(el => el.remove());

        // Filter events for this week
        const weekEvents = this.events.filter(event => {
            const eventDate = new Date(event.start_time);
            return eventDate >= weekStart && eventDate < weekEnd;
        });

        weekEvents.forEach(event => {
            const eventDate = new Date(event.start_time);
            const dayIndex = (eventDate.getDay() + 6) % 7; // Adjust for Monday start
            const dayColumn = document.querySelector(`.day-column[data-date="${eventDate.toISOString().split('T')[0]}"]`);
            
            if (!dayColumn) return;

            const eventBlock = document.createElement('div');
            eventBlock.className = `event-block type-${event.event_type || 'meeting'}`;
            eventBlock.dataset.eventId = event.id;
            
            const startTime = new Date(event.start_time);
            const endTime = new Date(event.end_time);
            const startMinutes = startTime.getHours() * 60 + startTime.getMinutes();
            const duration = (endTime - startTime) / (1000 * 60); // minutes
            
            eventBlock.style.top = `${startMinutes}px`;
            eventBlock.style.height = `${Math.max(duration, 30)}px`;
            
            eventBlock.innerHTML = `
                <div class="event-title">${event.title}</div>
                <div class="event-time">${startTime.toLocaleTimeString('en-US', { 
                    hour: 'numeric', 
                    minute: '2-digit',
                    hour12: true 
                })}</div>
            `;
            
            eventBlock.addEventListener('click', () => this.showEventDetails(event));
            dayColumn.appendChild(eventBlock);
        });
    }

    renderMonthView() {
        const monthStart = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 1);
        const monthEnd = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, 0);
        
        // Render month headers
        const monthHeader = document.getElementById('month-header');
        monthHeader.innerHTML = '';
        
        const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
        dayNames.forEach(name => {
            const header = document.createElement('div');
            header.className = 'month-day-header';
            header.textContent = name;
            monthHeader.appendChild(header);
        });

        // Render month grid
        const monthGrid = document.getElementById('month-grid');
        monthGrid.innerHTML = '';
        
        const firstDayOfWeek = monthStart.getDay();
        const daysInMonth = monthEnd.getDate();
        const totalCells = Math.ceil((firstDayOfWeek + daysInMonth) / 7) * 7;

        for (let i = 0; i < totalCells; i++) {
            const dayNumber = i - firstDayOfWeek + 1;
            const date = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), dayNumber);
            
            const dayCell = document.createElement('div');
            dayCell.className = 'month-day';
            dayCell.dataset.date = date.toISOString().split('T')[0];
            
            if (dayNumber < 1 || dayNumber > daysInMonth) {
                dayCell.classList.add('other-month');
                const otherMonthDay = dayNumber < 1 
                    ? new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() - 1, dayNumber + new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 0).getDate())
                    : new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, dayNumber - daysInMonth);
                dayCell.innerHTML = `
                    <div class="month-day-number">${otherMonthDay.getDate()}</div>
                    <div class="month-events"></div>
                `;
            } else {
                if (this.isToday(date)) dayCell.classList.add('today');
                dayCell.innerHTML = `
                    <div class="month-day-number">${dayNumber}</div>
                    <div class="month-events"></div>
                `;
            }
            
            dayCell.addEventListener('click', () => this.selectDate(date));
            monthGrid.appendChild(dayCell);
        }

        // Render events
        this.renderMonthEvents();
    }

    renderMonthEvents() {
        const monthStart = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 1);
        const monthEnd = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, 0);

        // Filter events for this month
        const monthEvents = this.events.filter(event => {
            const eventDate = new Date(event.start_time);
            return eventDate >= monthStart && eventDate <= monthEnd;
        });

        monthEvents.forEach(event => {
            const eventDate = new Date(event.start_time);
            const dayCell = document.querySelector(`.month-day[data-date="${eventDate.toISOString().split('T')[0]}"]`);
            
            if (!dayCell) return;

            const eventsContainer = dayCell.querySelector('.month-events');
            const eventElement = document.createElement('div');
            eventElement.className = `month-event type-${event.event_type || 'meeting'}`;
            eventElement.textContent = event.title;
            eventElement.addEventListener('click', (e) => {
                e.stopPropagation();
                this.showEventDetails(event);
            });
            
            eventsContainer.appendChild(eventElement);
        });
    }

    renderAgendaView() {
        const agendaList = document.getElementById('agenda-list');
        agendaList.innerHTML = '';

        // Group events by date
        const eventsByDate = this.groupEventsByDate(this.events);
        const sortedDates = Object.keys(eventsByDate).sort();

        if (sortedDates.length === 0) {
            agendaList.innerHTML = '<div class="no-events">No upcoming events</div>';
            return;
        }

        sortedDates.forEach(dateStr => {
            const date = new Date(dateStr + 'T00:00:00');
            const events = eventsByDate[dateStr];

            const agendaDay = document.createElement('div');
            agendaDay.className = 'agenda-day';
            
            const header = document.createElement('div');
            header.className = 'agenda-day-header';
            header.textContent = this.formatDateForAgenda(date);
            agendaDay.appendChild(header);

            const eventsContainer = document.createElement('div');
            eventsContainer.className = 'agenda-events';

            events.forEach(event => {
                const eventElement = document.createElement('div');
                eventElement.className = 'agenda-event';
                eventElement.innerHTML = `
                    <div class="agenda-event-time">
                        ${new Date(event.start_time).toLocaleTimeString('en-US', { 
                            hour: 'numeric', 
                            minute: '2-digit',
                            hour12: true 
                        })}
                    </div>
                    <div class="agenda-event-info">
                        <div class="agenda-event-title">${event.title}</div>
                        <div class="agenda-event-details">
                            ${event.location ? event.location + ' • ' : ''}
                            ${this.formatDuration(new Date(event.start_time), new Date(event.end_time))}
                        </div>
                    </div>
                `;
                
                eventElement.addEventListener('click', () => this.showEventDetails(event));
                eventsContainer.appendChild(eventElement);
            });

            agendaDay.appendChild(eventsContainer);
            agendaList.appendChild(agendaDay);
        });
    }

    // Event management
    async loadEvents() {
        try {
            this.showLoading(true);
            const response = await this.apiCall('/events');
            this.events = response.events || [];
            this.renderCurrentView();
            this.updateUpcomingEvents();
        } catch (error) {
            console.error('Failed to load events:', error);
            this.showToast('Failed to load events', 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async handleCreateEvent(e) {
        e.preventDefault();
        
        const formData = new FormData(e.target);
        const eventData = {
            title: formData.get('title'),
            description: formData.get('description'),
            start_time: new Date(formData.get('start_time')).toISOString(),
            end_time: new Date(formData.get('end_time')).toISOString(),
            location: formData.get('location'),
            event_type: formData.get('event_type')
        };

        // Add reminder if specified
        const reminderMinutes = formData.get('reminder');
        if (reminderMinutes) {
            eventData.reminders = [{
                minutes_before: parseInt(reminderMinutes),
                type: 'email'
            }];
        }

        try {
            this.showLoading(true);
            await this.apiCall('/events', {
                method: 'POST',
                body: JSON.stringify(eventData)
            });

            this.hideModal('new-event-modal');
            e.target.reset();
            this.showToast('Event created successfully', 'success');
            await this.loadEvents();
        } catch (error) {
            console.error('Failed to create event:', error);
            this.showToast('Failed to create event', 'error');
        } finally {
            this.showLoading(false);
        }
    }

    showEventDetails(event) {
        const modal = document.getElementById('event-details-modal');
        const title = document.getElementById('event-details-title');
        const content = document.getElementById('event-details-content');

        title.textContent = event.title;
        
        const startTime = new Date(event.start_time);
        const endTime = new Date(event.end_time);
        
        content.innerHTML = `
            <div class="event-detail-item">
                <strong>Date:</strong> ${startTime.toLocaleDateString('en-US', { 
                    weekday: 'long', 
                    year: 'numeric', 
                    month: 'long', 
                    day: 'numeric' 
                })}
            </div>
            <div class="event-detail-item">
                <strong>Time:</strong> ${startTime.toLocaleTimeString('en-US', { 
                    hour: 'numeric', 
                    minute: '2-digit',
                    hour12: true 
                })} - ${endTime.toLocaleTimeString('en-US', { 
                    hour: 'numeric', 
                    minute: '2-digit',
                    hour12: true 
                })}
            </div>
            ${event.location ? `<div class="event-detail-item"><strong>Location:</strong> ${event.location}</div>` : ''}
            ${event.description ? `<div class="event-detail-item"><strong>Description:</strong> ${event.description}</div>` : ''}
            <div class="event-detail-item">
                <strong>Type:</strong> ${event.event_type || 'Meeting'}
            </div>
        `;

        this.showModal('event-details-modal');
    }

    // AI Chat
    async handleChatSend() {
        const input = document.getElementById('chat-input');
        const message = input.value.trim();
        
        if (!message) return;

        // Add user message to chat
        this.addChatMessage(message, 'user');
        input.value = '';

        try {
            const response = await this.apiCall('/schedule/chat', {
                method: 'POST',
                body: JSON.stringify({ message })
            });

            this.addChatMessage(response.response, 'assistant');

            // Handle suggested actions
            if (response.suggested_actions && response.suggested_actions.length > 0) {
                const actionsHtml = response.suggested_actions.map(action => 
                    `<button class="btn btn-sm btn-secondary" onclick="calendar.handleSuggestedAction('${action.action}', ${JSON.stringify(action.parameters).replace(/"/g, '&quot;')})">
                        ${action.action.replace('_', ' ').toUpperCase()} (${Math.round(action.confidence * 100)}%)
                    </button>`
                ).join(' ');
                
                this.addChatMessage(`Suggested actions: ${actionsHtml}`, 'assistant');
            }
        } catch (error) {
            console.error('Chat failed:', error);
            this.addChatMessage('Sorry, I encountered an error. Please try again.', 'assistant');
        }
    }

    addChatMessage(message, sender) {
        const messagesContainer = document.getElementById('chat-messages');
        const messageElement = document.createElement('div');
        messageElement.className = `chat-message ${sender}`;
        messageElement.innerHTML = `<div class="message-content">${message}</div>`;
        messagesContainer.appendChild(messageElement);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    handleSuggestedAction(action, parameters) {
        console.log('Suggested action:', action, parameters);
        // Handle suggested actions like creating events
        if (action === 'create_event') {
            // Pre-fill the new event form with suggested parameters
            this.showModal('new-event-modal');
            if (parameters.title) {
                document.getElementById('event-title').value = parameters.title;
            }
            if (parameters.start_time) {
                document.getElementById('event-start').value = parameters.start_time.slice(0, 16);
            }
        }
    }

    // Search
    async handleSearch(e) {
        const query = e.target.value.trim();
        
        if (!query) {
            await this.loadEvents();
            return;
        }

        try {
            const response = await this.apiCall(`/events?search=${encodeURIComponent(query)}`);
            this.events = response.events || [];
            this.renderCurrentView();
        } catch (error) {
            console.error('Search failed:', error);
        }
    }

    // Utility functions
    getWeekStart(date) {
        const start = new Date(date);
        const day = start.getDay();
        const diff = start.getDate() - day + (day === 0 ? -6 : 1); // Adjust for Monday start
        return new Date(start.setDate(diff));
    }

    isToday(date) {
        const today = new Date();
        return date.toDateString() === today.toDateString();
    }

    groupEventsByDate(events) {
        return events.reduce((groups, event) => {
            const date = new Date(event.start_time).toISOString().split('T')[0];
            if (!groups[date]) groups[date] = [];
            groups[date].push(event);
            return groups;
        }, {});
    }

    formatDateForAgenda(date) {
        const today = new Date();
        const tomorrow = new Date(today);
        tomorrow.setDate(tomorrow.getDate() + 1);

        if (date.toDateString() === today.toDateString()) {
            return 'Today, ' + date.toLocaleDateString('en-US', { 
                weekday: 'long', 
                month: 'long', 
                day: 'numeric' 
            });
        } else if (date.toDateString() === tomorrow.toDateString()) {
            return 'Tomorrow, ' + date.toLocaleDateString('en-US', { 
                weekday: 'long', 
                month: 'long', 
                day: 'numeric' 
            });
        } else {
            return date.toLocaleDateString('en-US', { 
                weekday: 'long', 
                year: 'numeric', 
                month: 'long', 
                day: 'numeric' 
            });
        }
    }

    formatDuration(start, end) {
        const diff = end - start;
        const hours = Math.floor(diff / (1000 * 60 * 60));
        const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
        
        if (hours > 0) {
            return `${hours}h${minutes > 0 ? ` ${minutes}m` : ''}`;
        } else {
            return `${minutes}m`;
        }
    }

    selectDate(date) {
        this.currentDate = date;
        if (this.currentView !== 'week') {
            this.switchView('week');
        } else {
            this.renderCurrentView();
        }
    }

    // UI helpers
    showModal(modalId) {
        document.getElementById(modalId).classList.add('active');
        document.body.style.overflow = 'hidden';
    }

    hideModal(modalId) {
        document.getElementById(modalId).classList.remove('active');
        document.body.style.overflow = '';
    }

    showLoading(show) {
        const loading = document.getElementById('loading');
        if (show) {
            loading.classList.remove('hidden');
        } else {
            loading.classList.add('hidden');
        }
    }

    showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        
        const icon = {
            success: 'fa-check-circle',
            error: 'fa-exclamation-circle',
            warning: 'fa-exclamation-triangle',
            info: 'fa-info-circle'
        }[type] || 'fa-info-circle';
        
        toast.innerHTML = `
            <i class="fas ${icon}"></i>
            <span>${message}</span>
        `;
        
        container.appendChild(toast);
        
        // Auto remove after 5 seconds
        setTimeout(() => {
            toast.remove();
        }, 5000);
    }

    updateUpcomingEvents() {
        const container = document.getElementById('upcoming-events');
        container.innerHTML = '';

        const upcomingEvents = this.events
            .filter(event => new Date(event.start_time) > new Date())
            .sort((a, b) => new Date(a.start_time) - new Date(b.start_time))
            .slice(0, 5);

        if (upcomingEvents.length === 0) {
            container.innerHTML = '<div class="no-events">No upcoming events</div>';
            return;
        }

        upcomingEvents.forEach(event => {
            const eventElement = document.createElement('div');
            eventElement.className = 'upcoming-event';
            eventElement.innerHTML = `
                <div class="upcoming-event-title">${event.title}</div>
                <div class="upcoming-event-time">
                    ${new Date(event.start_time).toLocaleDateString('en-US', { 
                        month: 'short', 
                        day: 'numeric' 
                    })} at ${new Date(event.start_time).toLocaleTimeString('en-US', { 
                        hour: 'numeric', 
                        minute: '2-digit',
                        hour12: true 
                    })}
                </div>
            `;
            eventElement.addEventListener('click', () => this.showEventDetails(event));
            container.appendChild(eventElement);
        });
    }

    // Keyboard shortcuts
    handleKeyboard(e) {
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

        switch (e.key) {
            case 'n':
                if (e.ctrlKey || e.metaKey) {
                    e.preventDefault();
                    this.showModal('new-event-modal');
                }
                break;
            case 'w':
                this.switchView('week');
                break;
            case 'm':
                this.switchView('month');
                break;
            case 'a':
                this.switchView('agenda');
                break;
            case 't':
                this.goToToday();
                break;
            case 'ArrowLeft':
                this.navigate(-1);
                break;
            case 'ArrowRight':
                this.navigate(1);
                break;
            case '/':
                e.preventDefault();
                document.getElementById('search-input').focus();
                break;
        }
    }

    // Polling for live updates
    startPolling() {
        // Poll for updates every 5 minutes
        setInterval(() => {
            this.loadEvents();
        }, 5 * 60 * 1000);
    }

    // Utility function for debouncing
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }
}

// Initialize calendar when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.calendar = new CalendarApp();
});

// Make calendar available globally for debugging
window.CalendarApp = CalendarApp;