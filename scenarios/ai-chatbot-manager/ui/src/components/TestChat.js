import React, { useState, useEffect, useRef } from 'react';
import { useParams } from 'react-router-dom';
import apiClient from '../utils/api';

function TestChat() {
  const { id } = useParams();
  const [chatbot, setChatbot] = useState(null);
  const [messages, setMessages] = useState([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId] = useState(() => `test-session-${Date.now()}`);
  const messagesEndRef = useRef(null);

  useEffect(() => {
    loadChatbot();
    // Add initial greeting
    addMessage('assistant', 'Hi! I\'m ready to chat. How can I help you today?');
  }, [id]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const loadChatbot = async () => {
    try {
      const response = await apiClient.get(`/api/v1/chatbots/${id}`);
      if (response.ok) {
        const data = await response.json();
        setChatbot(data);
      } else {
        alert('Failed to load chatbot');
      }
    } catch (error) {
      console.error('Failed to load chatbot:', error);
      alert('Failed to load chatbot');
    }
  };

  const addMessage = (role, content, metadata = {}) => {
    const message = {
      id: Date.now() + Math.random(),
      role,
      content,
      metadata,
      timestamp: new Date()
    };
    setMessages(prev => [...prev, message]);
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const sendMessage = async () => {
    if (!inputValue.trim() || isLoading) return;

    const userMessage = inputValue.trim();
    setInputValue('');
    addMessage('user', userMessage);
    setIsLoading(true);

    try {
      const response = await apiClient.post(`/api/v1/chat/${id}`, {
          message: userMessage,
          session_id: sessionId,
          context: { source: 'test-interface' }
        })
      });

      if (response.ok) {
        const data = await response.json();
        addMessage('assistant', data.response, {
          confidence: data.confidence,
          should_escalate: data.should_escalate,
          conversation_id: data.conversation_id
        });
      } else {
        addMessage('assistant', 'Sorry, I encountered an error processing your message. Please try again.');
      }
    } catch (error) {
      console.error('Failed to send message:', error);
      addMessage('assistant', 'Sorry, I\'m having trouble connecting. Please check your connection and try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const clearChat = () => {
    if (confirm('Are you sure you want to clear the chat history?')) {
      setMessages([]);
      addMessage('assistant', 'Chat cleared. How can I help you?');
    }
  };

  const formatTime = (timestamp) => {
    return timestamp.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  if (!chatbot) {
    return (
      <div className="loading-container">
        <div className="loading-spinner"></div>
        <p>Loading chatbot...</p>
      </div>
    );
  }

  return (
    <div className="test-chat">
      <div className="page-header">
        <div>
          <h1 className="page-title">Test Chat: {chatbot.name}</h1>
          <p className="page-subtitle">
            Test your chatbot's responses in a live chat environment
          </p>
        </div>
        <div>
          <button onClick={clearChat} className="btn btn-outline">
            Clear Chat
          </button>
        </div>
      </div>

      <div className="chat-container">
        <div className="chat-info">
          <div className="card">
            <h3>Chatbot Information</h3>
            <div className="info-item">
              <strong>Name:</strong> {chatbot.name}
            </div>
            <div className="info-item">
              <strong>Model:</strong> {chatbot.model_config?.model || 'llama3.2'}
            </div>
            <div className="info-item">
              <strong>Status:</strong> 
              <span className={`badge ${chatbot.is_active ? 'badge-success' : 'badge-gray'}`}>
                {chatbot.is_active ? 'Active' : 'Inactive'}
              </span>
            </div>
            {chatbot.description && (
              <div className="info-item">
                <strong>Description:</strong> {chatbot.description}
              </div>
            )}
            
            <div className="personality-preview">
              <strong>Personality:</strong>
              <div className="personality-text">
                {chatbot.personality.length > 150 
                  ? `${chatbot.personality.substring(0, 150)}...`
                  : chatbot.personality
                }
              </div>
            </div>
          </div>
        </div>

        <div className="chat-interface">
          <div className="card chat-card">
            <div className="chat-messages">
              {messages.map(message => (
                <div key={message.id} className={`message message-${message.role}`}>
                  <div className="message-content">
                    <div className="message-text">{message.content}</div>
                    <div className="message-meta">
                      <span className="message-time">
                        {formatTime(message.timestamp)}
                      </span>
                      {message.metadata?.confidence && (
                        <span className="message-confidence">
                          Confidence: {(message.metadata.confidence * 100).toFixed(0)}%
                        </span>
                      )}
                      {message.metadata?.should_escalate && (
                        <span className="message-flag">
                          ⚠️ Low confidence
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="message-avatar">
                    {message.role === 'user' ? '👤' : '🤖'}
                  </div>
                </div>
              ))}
              
              {isLoading && (
                <div className="message message-assistant">
                  <div className="message-content">
                    <div className="typing-indicator">
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                  </div>
                  <div className="message-avatar">🤖</div>
                </div>
              )}
              
              <div ref={messagesEndRef} />
            </div>

            <div className="chat-input">
              <div className="input-group">
                <textarea
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder="Type your message..."
                  className="chat-textarea"
                  rows={1}
                  disabled={isLoading}
                />
                <button
                  onClick={sendMessage}
                  className="btn btn-primary send-button"
                  disabled={!inputValue.trim() || isLoading}
                >
                  {isLoading ? (
                    <div className="loading-spinner small"></div>
                  ) : (
                    '➤'
                  )}
                </button>
              </div>
              <div className="input-help">
                Press Enter to send, Shift+Enter for new line
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default TestChat;