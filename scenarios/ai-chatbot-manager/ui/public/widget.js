/**
 * AI Chatbot Manager - Embeddable Widget SDK
 * Provides an embeddable chatbot widget for websites
 */
(function() {
  'use strict';

  // Configuration
  const CHATBOT_API_BASE = window.location.origin;
  const WIDGET_VERSION = '1.0.0';
  
  // Widget state
  let widgetConfig = {};
  let chatbotId = null;
  let sessionId = null;
  let isOpen = false;
  let isLoading = false;
  let messages = [];

  // DOM elements
  let widgetContainer = null;
  let chatWindow = null;
  let messagesContainer = null;
  let inputField = null;
  let toggleButton = null;

  /**
   * Initialize the widget
   */
  function init() {
    // Get configuration from script tag
    const scriptTag = document.querySelector('script[data-chatbot-id]');
    if (!scriptTag) {
      console.error('AI Chatbot Manager: No script tag with data-chatbot-id found');
      return;
    }

    chatbotId = scriptTag.getAttribute('data-chatbot-id');
    if (!chatbotId) {
      console.error('AI Chatbot Manager: No chatbot ID specified');
      return;
    }

    // Generate session ID
    sessionId = 'widget-session-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);

    // Load chatbot configuration
    loadChatbotConfig().then(() => {
      createWidget();
      console.log('AI Chatbot Manager: Widget initialized for chatbot', chatbotId);
    }).catch(error => {
      console.error('AI Chatbot Manager: Failed to initialize widget', error);
    });
  }

  /**
   * Load chatbot configuration from API
   */
  async function loadChatbotConfig() {
    try {
      const response = await fetch(`${CHATBOT_API_BASE}/api/v1/chatbots/${chatbotId}`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const chatbot = await response.json();
      widgetConfig = {
        name: chatbot.name,
        greeting: chatbot.widget_config?.greeting || 'Hi! How can I help you today?',
        theme: chatbot.widget_config?.theme || 'light',
        position: chatbot.widget_config?.position || 'bottom-right',
        primaryColor: chatbot.widget_config?.primaryColor || '#3b82f6',
        personality: chatbot.personality
      };
    } catch (error) {
      console.error('Failed to load chatbot configuration:', error);
      // Use default configuration
      widgetConfig = {
        name: 'AI Assistant',
        greeting: 'Hi! How can I help you today?',
        theme: 'light',
        position: 'bottom-right',
        primaryColor: '#3b82f6'
      };
    }
  }

  /**
   * Create the widget DOM elements
   */
  function createWidget() {
    // Create main container
    widgetContainer = document.createElement('div');
    widgetContainer.id = 'chatbot-widget-container';
    widgetContainer.innerHTML = getWidgetHTML();
    document.body.appendChild(widgetContainer);

    // Apply styles
    injectStyles();

    // Get DOM references
    toggleButton = widgetContainer.querySelector('#chatbot-toggle-btn');
    chatWindow = widgetContainer.querySelector('#chatbot-window');
    messagesContainer = widgetContainer.querySelector('#chatbot-messages');
    inputField = widgetContainer.querySelector('#chatbot-input');

    // Attach event listeners
    attachEventListeners();

    // Add initial greeting message
    addMessage('assistant', widgetConfig.greeting);
  }

  /**
   * Get the widget HTML structure
   */
  function getWidgetHTML() {
    return `
      <!-- Toggle Button -->
      <div id="chatbot-toggle-btn" class="chatbot-toggle-btn">
        <div class="chatbot-icon">💬</div>
        <div class="chatbot-close-icon" style="display: none;">✕</div>
      </div>

      <!-- Chat Window -->
      <div id="chatbot-window" class="chatbot-window" style="display: none;">
        <!-- Header -->
        <div class="chatbot-header">
          <div class="chatbot-title">
            <div class="chatbot-avatar">🤖</div>
            <div class="chatbot-info">
              <div class="chatbot-name">${widgetConfig.name}</div>
              <div class="chatbot-status">Online</div>
            </div>
          </div>
          <div class="chatbot-actions">
            <button id="chatbot-minimize" class="chatbot-action-btn">−</button>
          </div>
        </div>

        <!-- Messages -->
        <div id="chatbot-messages" class="chatbot-messages">
          <!-- Messages will be inserted here -->
        </div>

        <!-- Input -->
        <div class="chatbot-input-container">
          <input 
            id="chatbot-input" 
            type="text" 
            class="chatbot-input" 
            placeholder="Type your message..."
            maxlength="500"
          />
          <button id="chatbot-send" class="chatbot-send-btn">
            <span class="send-icon">➤</span>
            <div class="loading-spinner" style="display: none;"></div>
          </button>
        </div>

        <!-- Footer -->
        <div class="chatbot-footer">
          <div class="chatbot-branding">
            Powered by AI Chatbot Manager
          </div>
        </div>
      </div>
    `;
  }

  /**
   * Inject CSS styles for the widget
   */
  function injectStyles() {
    const styles = `
      #chatbot-widget-container {
        position: fixed;
        ${widgetConfig.position.includes('right') ? 'right: 20px;' : 'left: 20px;'}
        ${widgetConfig.position.includes('bottom') ? 'bottom: 20px;' : 'top: 20px;'}
        z-index: 999999;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Arial, sans-serif;
      }

      .chatbot-toggle-btn {
        width: 60px;
        height: 60px;
        border-radius: 50%;
        background-color: ${widgetConfig.primaryColor};
        color: white;
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1.5rem;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        transition: all 0.3s ease;
        position: relative;
      }

      .chatbot-toggle-btn:hover {
        transform: scale(1.05);
        box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
      }

      .chatbot-window {
        width: 350px;
        height: 500px;
        background-color: white;
        border-radius: 12px;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        display: flex;
        flex-direction: column;
        overflow: hidden;
        margin-bottom: 10px;
        animation: slideUp 0.3s ease;
      }

      @keyframes slideUp {
        from {
          opacity: 0;
          transform: translateY(20px);
        }
        to {
          opacity: 1;
          transform: translateY(0);
        }
      }

      .chatbot-header {
        background-color: ${widgetConfig.primaryColor};
        color: white;
        padding: 16px;
        display: flex;
        align-items: center;
        justify-content: space-between;
      }

      .chatbot-title {
        display: flex;
        align-items: center;
        gap: 12px;
      }

      .chatbot-avatar {
        font-size: 1.5rem;
      }

      .chatbot-name {
        font-weight: 600;
        font-size: 1rem;
      }

      .chatbot-status {
        font-size: 0.75rem;
        opacity: 0.9;
      }

      .chatbot-action-btn {
        background: none;
        border: none;
        color: white;
        font-size: 1.25rem;
        cursor: pointer;
        padding: 4px;
        border-radius: 4px;
        transition: background-color 0.2s;
      }

      .chatbot-action-btn:hover {
        background-color: rgba(255, 255, 255, 0.1);
      }

      .chatbot-messages {
        flex: 1;
        padding: 16px;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 12px;
        background-color: #f8fafc;
      }

      .chatbot-message {
        display: flex;
        align-items: flex-end;
        gap: 8px;
        max-width: 85%;
      }

      .chatbot-message.user {
        align-self: flex-end;
        flex-direction: row-reverse;
      }

      .chatbot-message-bubble {
        padding: 10px 14px;
        border-radius: 16px;
        font-size: 0.875rem;
        line-height: 1.4;
        word-wrap: break-word;
      }

      .chatbot-message.user .chatbot-message-bubble {
        background-color: ${widgetConfig.primaryColor};
        color: white;
        border-bottom-right-radius: 4px;
      }

      .chatbot-message.assistant .chatbot-message-bubble {
        background-color: white;
        color: #374151;
        border: 1px solid #e5e7eb;
        border-bottom-left-radius: 4px;
      }

      .chatbot-message-avatar {
        width: 24px;
        height: 24px;
        border-radius: 50%;
        background-color: #e5e7eb;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.75rem;
        flex-shrink: 0;
      }

      .chatbot-message.assistant .chatbot-message-avatar {
        background-color: ${widgetConfig.primaryColor};
        color: white;
      }

      .chatbot-input-container {
        padding: 16px;
        border-top: 1px solid #e5e7eb;
        display: flex;
        gap: 8px;
        background-color: white;
      }

      .chatbot-input {
        flex: 1;
        padding: 10px 12px;
        border: 1px solid #d1d5db;
        border-radius: 20px;
        font-size: 0.875rem;
        outline: none;
        transition: border-color 0.2s;
      }

      .chatbot-input:focus {
        border-color: ${widgetConfig.primaryColor};
      }

      .chatbot-send-btn {
        width: 36px;
        height: 36px;
        border-radius: 50%;
        background-color: ${widgetConfig.primaryColor};
        color: white;
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
        font-size: 1rem;
      }

      .chatbot-send-btn:hover:not(:disabled) {
        background-color: ${adjustColor(widgetConfig.primaryColor, -20)};
        transform: scale(1.05);
      }

      .chatbot-send-btn:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      .chatbot-footer {
        padding: 8px 16px;
        background-color: #f8fafc;
        border-top: 1px solid #e5e7eb;
      }

      .chatbot-branding {
        font-size: 0.75rem;
        color: #9ca3af;
        text-align: center;
      }

      .loading-spinner {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top: 2px solid white;
        border-radius: 50%;
        animation: spin 1s linear infinite;
      }

      @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
      }

      .typing-indicator {
        display: flex;
        gap: 4px;
        padding: 10px 14px;
      }

      .typing-dot {
        width: 6px;
        height: 6px;
        background-color: #9ca3af;
        border-radius: 50%;
        animation: typing 1.4s infinite ease-in-out;
      }

      .typing-dot:nth-child(1) { animation-delay: -0.32s; }
      .typing-dot:nth-child(2) { animation-delay: -0.16s; }

      @keyframes typing {
        0%, 80%, 100% { opacity: 0.3; }
        40% { opacity: 1; }
      }

      /* Mobile responsive */
      @media (max-width: 480px) {
        .chatbot-window {
          width: calc(100vw - 40px);
          height: calc(100vh - 100px);
          max-height: 600px;
        }
      }
    `;

    const styleElement = document.createElement('style');
    styleElement.textContent = styles;
    document.head.appendChild(styleElement);
  }

  /**
   * Attach event listeners
   */
  function attachEventListeners() {
    // Toggle button
    toggleButton.addEventListener('click', toggleWidget);

    // Minimize button
    const minimizeBtn = widgetContainer.querySelector('#chatbot-minimize');
    minimizeBtn.addEventListener('click', closeWidget);

    // Send button
    const sendBtn = widgetContainer.querySelector('#chatbot-send');
    sendBtn.addEventListener('click', sendMessage);

    // Input field
    inputField.addEventListener('keypress', (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });

    inputField.addEventListener('input', (e) => {
      // Auto-resize input (basic)
      e.target.style.height = 'auto';
      e.target.style.height = e.target.scrollHeight + 'px';
    });
  }

  /**
   * Toggle widget open/closed
   */
  function toggleWidget() {
    if (isOpen) {
      closeWidget();
    } else {
      openWidget();
    }
  }

  /**
   * Open widget
   */
  function openWidget() {
    chatWindow.style.display = 'flex';
    toggleButton.querySelector('.chatbot-icon').style.display = 'none';
    toggleButton.querySelector('.chatbot-close-icon').style.display = 'block';
    isOpen = true;
    inputField.focus();
  }

  /**
   * Close widget
   */
  function closeWidget() {
    chatWindow.style.display = 'none';
    toggleButton.querySelector('.chatbot-icon').style.display = 'block';
    toggleButton.querySelector('.chatbot-close-icon').style.display = 'none';
    isOpen = false;
  }

  /**
   * Add message to chat
   */
  function addMessage(role, content, options = {}) {
    const message = {
      id: Date.now() + Math.random(),
      role,
      content,
      timestamp: new Date()
    };

    messages.push(message);

    const messageElement = document.createElement('div');
    messageElement.className = `chatbot-message ${role}`;
    messageElement.innerHTML = `
      <div class="chatbot-message-avatar">
        ${role === 'user' ? '👤' : '🤖'}
      </div>
      <div class="chatbot-message-bubble">
        ${content}
      </div>
    `;

    messagesContainer.appendChild(messageElement);
    scrollToBottom();
  }

  /**
   * Send message to chatbot
   */
  async function sendMessage() {
    const message = inputField.value.trim();
    if (!message || isLoading) return;

    // Clear input
    inputField.value = '';
    inputField.style.height = 'auto';

    // Add user message
    addMessage('user', message);

    // Show loading state
    setLoadingState(true);
    showTypingIndicator();

    try {
      const response = await fetch(`${CHATBOT_API_BASE}/api/v1/chat/${chatbotId}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          message: message,
          session_id: sessionId,
          context: { 
            source: 'widget',
            user_agent: navigator.userAgent,
            url: window.location.href
          }
        })
      });

      hideTypingIndicator();

      if (response.ok) {
        const data = await response.json();
        addMessage('assistant', data.response);

        // Handle lead qualification
        if (data.lead_qualification && Object.keys(data.lead_qualification).length > 0) {
          console.log('Lead qualification data:', data.lead_qualification);
        }
      } else {
        addMessage('assistant', 'I apologize, but I\'m having trouble processing your message right now. Please try again in a moment.');
      }
    } catch (error) {
      console.error('Failed to send message:', error);
      hideTypingIndicator();
      addMessage('assistant', 'I\'m having trouble connecting right now. Please check your internet connection and try again.');
    } finally {
      setLoadingState(false);
    }
  }

  /**
   * Show typing indicator
   */
  function showTypingIndicator() {
    const typingElement = document.createElement('div');
    typingElement.className = 'chatbot-message assistant';
    typingElement.id = 'typing-indicator';
    typingElement.innerHTML = `
      <div class="chatbot-message-avatar">🤖</div>
      <div class="chatbot-message-bubble">
        <div class="typing-indicator">
          <div class="typing-dot"></div>
          <div class="typing-dot"></div>
          <div class="typing-dot"></div>
        </div>
      </div>
    `;

    messagesContainer.appendChild(typingElement);
    scrollToBottom();
  }

  /**
   * Hide typing indicator
   */
  function hideTypingIndicator() {
    const typingElement = document.getElementById('typing-indicator');
    if (typingElement) {
      typingElement.remove();
    }
  }

  /**
   * Set loading state
   */
  function setLoadingState(loading) {
    isLoading = loading;
    const sendBtn = widgetContainer.querySelector('#chatbot-send');
    const sendIcon = sendBtn.querySelector('.send-icon');
    const spinner = sendBtn.querySelector('.loading-spinner');

    if (loading) {
      sendIcon.style.display = 'none';
      spinner.style.display = 'block';
      sendBtn.disabled = true;
      inputField.disabled = true;
    } else {
      sendIcon.style.display = 'block';
      spinner.style.display = 'none';
      sendBtn.disabled = false;
      inputField.disabled = false;
    }
  }

  /**
   * Scroll messages to bottom
   */
  function scrollToBottom() {
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }

  /**
   * Adjust color brightness
   */
  function adjustColor(color, amount) {
    const usePound = color[0] === '#';
    const col = usePound ? color.slice(1) : color;
    const num = parseInt(col, 16);
    let r = (num >> 16) + amount;
    let g = (num >> 8 & 0x00FF) + amount;
    let b = (num & 0x0000FF) + amount;
    r = r > 255 ? 255 : r < 0 ? 0 : r;
    g = g > 255 ? 255 : g < 0 ? 0 : g;
    b = b > 255 ? 255 : b < 0 ? 0 : b;
    return (usePound ? '#' : '') + (r << 16 | g << 8 | b).toString(16);
  }

  /**
   * Public API
   */
  window.ChatbotWidget = {
    open: openWidget,
    close: closeWidget,
    toggle: toggleWidget,
    sendMessage: (message) => {
      inputField.value = message;
      sendMessage();
    }
  };

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

})();