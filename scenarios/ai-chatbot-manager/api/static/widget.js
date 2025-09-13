/**
 * AI Chatbot Manager Widget
 * Embeddable chat widget for website integration
 * 
 * Configuration:
 * - Set window.CHATBOT_API_URL before including this script
 * - Or use <meta name="chatbot-api-url" content="YOUR_API_URL">
 * - Pass CHATBOT_ID and WIDGET_CONFIG when initializing
 */

(function(window, document) {
    'use strict';

    // Widget factory function
    window.AIChatbotWidget = function(options) {
        // Validate required options
        if (!options || !options.chatbotId) {
            console.error('AIChatbotWidget: chatbotId is required');
            return;
        }

        // Configuration
        const CHATBOT_ID = options.chatbotId;
        const WIDGET_CONFIG = options.config || {};
        
        // Determine API URL dynamically
        const API_URL = options.apiUrl || window.CHATBOT_API_URL || (function() {
            // Look for API URL in meta tags
            const metaApiUrl = document.querySelector('meta[name="chatbot-api-url"]');
            if (metaApiUrl && metaApiUrl.content) {
                return metaApiUrl.content;
            }
            
            // Default to same host with standard API port
            const currentProtocol = window.location.protocol;
            const currentHost = window.location.hostname;
            const apiPort = window.CHATBOT_API_PORT || '15000';
            
            console.warn('CHATBOT_API_URL not configured. Set window.CHATBOT_API_URL or add <meta name="chatbot-api-url" content="YOUR_API_URL">');
            return currentProtocol + '//' + currentHost + ':' + apiPort;
        })();
        
        const WS_URL = API_URL.replace('http', 'ws') + '/api/v1/ws/' + CHATBOT_ID;
        
        console.log('Chatbot Widget initializing with API:', API_URL);
        
        // Widget state
        let ws = null;
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;
        const sessionId = 'widget-' + Math.random().toString(36).substr(2, 9);
        
        // Create widget container
        const container = document.createElement('div');
        container.id = 'ai-chatbot-widget-' + CHATBOT_ID;
        container.className = 'ai-chatbot-widget';
        
        // Apply positioning based on config
        const position = WIDGET_CONFIG.position || 'bottom-right';
        const [vPos, hPos] = position.split('-');
        container.style.cssText = `
            position: fixed;
            z-index: 9999;
            ${vPos}: 20px;
            ${hPos}: 20px;
        `;
        
        // Create chat button
        const button = document.createElement('button');
        button.className = 'ai-chatbot-button';
        button.setAttribute('aria-label', 'Open chat');
        button.innerHTML = WIDGET_CONFIG.buttonIcon || '💬';
        button.style.cssText = `
            width: 60px;
            height: 60px;
            border-radius: 50%;
            border: none;
            background: ${WIDGET_CONFIG.primaryColor || '#007bff'};
            color: white;
            font-size: 24px;
            cursor: pointer;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            transition: transform 0.3s, box-shadow 0.3s;
        `;
        
        // Button hover effects
        button.addEventListener('mouseenter', function() {
            this.style.transform = 'scale(1.1)';
            this.style.boxShadow = '0 6px 16px rgba(0,0,0,0.2)';
        });
        
        button.addEventListener('mouseleave', function() {
            this.style.transform = 'scale(1)';
            this.style.boxShadow = '0 4px 12px rgba(0,0,0,0.15)';
        });
        
        // Create chat window
        const chatWindow = document.createElement('div');
        chatWindow.className = 'ai-chatbot-window';
        chatWindow.style.cssText = `
            display: none;
            position: absolute;
            ${vPos === 'bottom' ? 'bottom: 80px' : 'top: 80px'};
            ${hPos}: 0;
            width: 350px;
            height: 500px;
            background: white;
            border-radius: 12px;
            box-shadow: 0 5px 40px rgba(0,0,0,0.16);
            overflow: hidden;
            display: flex;
            flex-direction: column;
        `;
        
        // Create chat header
        const header = document.createElement('div');
        header.className = 'ai-chatbot-header';
        header.style.cssText = `
            padding: 20px;
            background: ${WIDGET_CONFIG.primaryColor || '#007bff'};
            color: white;
            font-weight: bold;
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-shrink: 0;
        `;
        
        const headerTitle = document.createElement('span');
        headerTitle.textContent = WIDGET_CONFIG.title || 'Chat Support';
        
        const closeButton = document.createElement('button');
        closeButton.className = 'ai-chatbot-close';
        closeButton.setAttribute('aria-label', 'Close chat');
        closeButton.innerHTML = '×';
        closeButton.style.cssText = `
            background: none;
            border: none;
            color: white;
            font-size: 24px;
            cursor: pointer;
            padding: 0;
            width: 30px;
            height: 30px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 4px;
            transition: background 0.2s;
        `;
        
        closeButton.addEventListener('mouseenter', function() {
            this.style.background = 'rgba(255,255,255,0.2)';
        });
        
        closeButton.addEventListener('mouseleave', function() {
            this.style.background = 'none';
        });
        
        header.appendChild(headerTitle);
        header.appendChild(closeButton);
        
        // Create messages container
        const messagesContainer = document.createElement('div');
        messagesContainer.className = 'ai-chatbot-messages';
        messagesContainer.style.cssText = `
            flex: 1;
            overflow-y: auto;
            padding: 20px;
            background: #f5f5f5;
        `;
        
        // Create input container
        const inputContainer = document.createElement('div');
        inputContainer.className = 'ai-chatbot-input-container';
        inputContainer.style.cssText = `
            padding: 20px;
            background: white;
            border-top: 1px solid #e0e0e0;
            flex-shrink: 0;
        `;
        
        const inputWrapper = document.createElement('div');
        inputWrapper.style.cssText = `
            display: flex;
            gap: 10px;
        `;
        
        const input = document.createElement('input');
        input.className = 'ai-chatbot-input';
        input.type = 'text';
        input.placeholder = WIDGET_CONFIG.inputPlaceholder || 'Type your message...';
        input.style.cssText = `
            flex: 1;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
            font-family: inherit;
        `;
        
        const sendButton = document.createElement('button');
        sendButton.className = 'ai-chatbot-send';
        sendButton.innerHTML = '➤';
        sendButton.style.cssText = `
            padding: 10px 15px;
            background: ${WIDGET_CONFIG.primaryColor || '#007bff'};
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
            transition: background 0.2s;
        `;
        
        sendButton.addEventListener('mouseenter', function() {
            this.style.opacity = '0.9';
        });
        
        sendButton.addEventListener('mouseleave', function() {
            this.style.opacity = '1';
        });
        
        inputWrapper.appendChild(input);
        inputWrapper.appendChild(sendButton);
        inputContainer.appendChild(inputWrapper);
        
        // Assemble chat window
        chatWindow.appendChild(header);
        chatWindow.appendChild(messagesContainer);
        chatWindow.appendChild(inputContainer);
        container.appendChild(button);
        container.appendChild(chatWindow);
        
        // Helper functions
        function addMessage(role, content) {
            const msgDiv = document.createElement('div');
            msgDiv.className = `ai-chatbot-message ai-chatbot-message-${role}`;
            
            const baseStyle = `
                margin-bottom: 10px;
                padding: 10px 12px;
                border-radius: 8px;
                word-wrap: break-word;
                max-width: 80%;
                animation: fadeIn 0.3s ease-in;
            `;
            
            let specificStyle = '';
            switch(role) {
                case 'user':
                    specificStyle = `
                        background: ${WIDGET_CONFIG.primaryColor || '#007bff'};
                        color: white;
                        align-self: flex-end;
                        margin-left: auto;
                    `;
                    break;
                case 'assistant':
                    specificStyle = `
                        background: white;
                        color: #333;
                        align-self: flex-start;
                    `;
                    break;
                case 'system':
                    specificStyle = `
                        background: #ffc107;
                        color: #333;
                        align-self: center;
                        font-style: italic;
                        text-align: center;
                    `;
                    break;
            }
            
            msgDiv.style.cssText = baseStyle + specificStyle;
            msgDiv.textContent = content;
            
            messagesContainer.appendChild(msgDiv);
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }
        
        function sendMessage(message) {
            if (!message || !message.trim()) return;
            
            // Add user message to UI
            addMessage('user', message);
            
            // Clear input
            input.value = '';
            
            // Send via WebSocket if connected, otherwise use HTTP
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    message: message,
                    session_id: sessionId
                }));
            } else {
                // Fallback to HTTP
                fetch(API_URL + '/api/v1/chat/' + CHATBOT_ID, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        message: message,
                        session_id: sessionId
                    })
                })
                .then(function(response) {
                    if (!response.ok) {
                        throw new Error('API request failed: ' + response.status);
                    }
                    return response.json();
                })
                .then(function(data) {
                    if (data.response) {
                        addMessage('assistant', data.response);
                    }
                })
                .catch(function(error) {
                    console.error('Chat error:', error);
                    addMessage('system', 'Sorry, I am unable to respond right now. Please try again later.');
                });
            }
        }
        
        function connectWebSocket() {
            try {
                ws = new WebSocket(WS_URL);
                
                ws.onopen = function() {
                    console.log('WebSocket connected to chatbot:', CHATBOT_ID);
                    reconnectAttempts = 0;
                };
                
                ws.onmessage = function(event) {
                    try {
                        const data = JSON.parse(event.data);
                        if (data.type === 'message' && data.payload) {
                            addMessage('assistant', data.payload.response);
                        } else if (data.type === 'error') {
                            console.error('WebSocket error:', data.payload);
                            addMessage('system', 'An error occurred. Please try again.');
                        }
                    } catch (e) {
                        console.error('Failed to parse WebSocket message:', e);
                    }
                };
                
                ws.onerror = function(error) {
                    console.error('WebSocket error:', error);
                };
                
                ws.onclose = function() {
                    console.log('WebSocket disconnected');
                    ws = null;
                    
                    // Attempt to reconnect if chat window is still open
                    if (chatWindow.style.display === 'flex' && reconnectAttempts < maxReconnectAttempts) {
                        reconnectAttempts++;
                        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
                        console.log('Reconnecting in ' + delay + 'ms... (attempt ' + reconnectAttempts + ')');
                        setTimeout(connectWebSocket, delay);
                    }
                };
            } catch (err) {
                console.error('WebSocket connection failed:', err);
                ws = null;
            }
        }
        
        // Event handlers
        button.addEventListener('click', function() {
            const isVisible = chatWindow.style.display === 'flex';
            chatWindow.style.display = isVisible ? 'none' : 'flex';
            
            if (!isVisible) {
                // Focus input when opening
                setTimeout(() => input.focus(), 100);
                
                // Connect WebSocket if not connected
                if (!ws) {
                    connectWebSocket();
                }
            }
        });
        
        closeButton.addEventListener('click', function() {
            chatWindow.style.display = 'none';
        });
        
        input.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage(this.value);
            }
        });
        
        sendButton.addEventListener('click', function() {
            sendMessage(input.value);
        });
        
        // Add widget to page
        function init() {
            // Add CSS animation
            if (!document.getElementById('ai-chatbot-widget-styles')) {
                const style = document.createElement('style');
                style.id = 'ai-chatbot-widget-styles';
                style.textContent = `
                    @keyframes fadeIn {
                        from { opacity: 0; transform: translateY(10px); }
                        to { opacity: 1; transform: translateY(0); }
                    }
                    
                    .ai-chatbot-messages {
                        display: flex;
                        flex-direction: column;
                    }
                    
                    .ai-chatbot-messages::-webkit-scrollbar {
                        width: 6px;
                    }
                    
                    .ai-chatbot-messages::-webkit-scrollbar-track {
                        background: #f1f1f1;
                    }
                    
                    .ai-chatbot-messages::-webkit-scrollbar-thumb {
                        background: #888;
                        border-radius: 3px;
                    }
                    
                    .ai-chatbot-messages::-webkit-scrollbar-thumb:hover {
                        background: #555;
                    }
                `;
                document.head.appendChild(style);
            }
            
            document.body.appendChild(container);
            
            // Send welcome message
            setTimeout(function() {
                addMessage('assistant', WIDGET_CONFIG.welcomeMessage || 'Hello! How can I help you today?');
            }, 1000);
        }
        
        // Initialize when DOM is ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', init);
        } else {
            init();
        }
        
        // Public API
        return {
            open: function() {
                chatWindow.style.display = 'flex';
                if (!ws) connectWebSocket();
            },
            close: function() {
                chatWindow.style.display = 'none';
            },
            sendMessage: sendMessage,
            destroy: function() {
                if (ws) ws.close();
                container.remove();
            }
        };
    };
    
})(window, document);