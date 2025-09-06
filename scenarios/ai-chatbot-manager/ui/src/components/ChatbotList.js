import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';

function ChatbotList() {
  const [loading, setLoading] = useState(true);
  const [chatbots, setChatbots] = useState([]);
  const [filter, setFilter] = useState('all'); // all, active, inactive

  useEffect(() => {
    loadChatbots();
  }, []);

  const loadChatbots = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/v1/chatbots');
      const data = await response.json();
      setChatbots(data);
    } catch (error) {
      console.error('Failed to load chatbots:', error);
    } finally {
      setLoading(false);
    }
  };

  const filteredChatbots = chatbots.filter(chatbot => {
    if (filter === 'active') return chatbot.is_active;
    if (filter === 'inactive') return !chatbot.is_active;
    return true;
  });

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  };

  const copyEmbedCode = (chatbotId) => {
    const embedCode = generateEmbedCode(chatbotId);
    navigator.clipboard.writeText(embedCode).then(() => {
      alert('Embed code copied to clipboard!');
    });
  };

  const generateEmbedCode = (chatbotId) => {
    return `<script src="${window.location.origin}/widget.js" data-chatbot-id="${chatbotId}"></script>`;
  };

  if (loading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner"></div>
        <p>Loading chatbots...</p>
      </div>
    );
  }

  return (
    <div className="chatbot-list-page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Chatbots</h1>
          <p className="page-subtitle">
            Manage your AI-powered chatbots and monitor their performance
          </p>
        </div>
        <div>
          <Link to="/chatbots/new" className="btn btn-primary">
            <span>➕</span>
            Create New Chatbot
          </Link>
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="filter-tabs">
        <button
          className={`filter-tab ${filter === 'all' ? 'active' : ''}`}
          onClick={() => setFilter('all')}
        >
          All ({chatbots.length})
        </button>
        <button
          className={`filter-tab ${filter === 'active' ? 'active' : ''}`}
          onClick={() => setFilter('active')}
        >
          Active ({chatbots.filter(c => c.is_active).length})
        </button>
        <button
          className={`filter-tab ${filter === 'inactive' ? 'active' : ''}`}
          onClick={() => setFilter('inactive')}
        >
          Inactive ({chatbots.filter(c => !c.is_active).length})
        </button>
      </div>

      {filteredChatbots.length === 0 ? (
        <div className="card">
          <div className="empty-state">
            <div className="empty-icon">🤖</div>
            <h3>
              {filter === 'all' 
                ? 'No chatbots yet' 
                : `No ${filter} chatbots`}
            </h3>
            <p>
              {filter === 'all' 
                ? 'Create your first AI chatbot to get started' 
                : `You don't have any ${filter} chatbots at the moment`}
            </p>
            {filter === 'all' && (
              <Link to="/chatbots/new" className="btn btn-primary">
                Create Your First Chatbot
              </Link>
            )}
          </div>
        </div>
      ) : (
        <div className="chatbots-grid">
          {filteredChatbots.map(chatbot => (
            <ChatbotCard
              key={chatbot.id}
              chatbot={chatbot}
              onCopyEmbed={() => copyEmbedCode(chatbot.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ChatbotCard({ chatbot, onCopyEmbed }) {
  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  };

  return (
    <div className="card chatbot-card">
      <div className="chatbot-card-header">
        <div className="chatbot-info">
          <h3 className="chatbot-name">{chatbot.name}</h3>
          <p className="chatbot-description">
            {chatbot.description || 'No description provided'}
          </p>
        </div>
        <div className="chatbot-status-badge">
          <span className={`badge ${chatbot.is_active ? 'badge-success' : 'badge-gray'}`}>
            {chatbot.is_active ? 'Active' : 'Inactive'}
          </span>
        </div>
      </div>

      <div className="chatbot-details">
        <div className="detail-item">
          <span className="detail-label">Model:</span>
          <span className="detail-value">
            {chatbot.model_config?.model || 'llama3.2'}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label">Created:</span>
          <span className="detail-value">
            {formatDate(chatbot.created_at)}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label">Updated:</span>
          <span className="detail-value">
            {formatDate(chatbot.updated_at)}
          </span>
        </div>
      </div>

      <div className="chatbot-personality">
        <div className="detail-label">Personality:</div>
        <div className="personality-text">
          {chatbot.personality.length > 100 
            ? `${chatbot.personality.substring(0, 100)}...`
            : chatbot.personality
          }
        </div>
      </div>

      <div className="chatbot-actions">
        <div className="action-group">
          <Link
            to={`/chatbots/${chatbot.id}/test`}
            className="btn btn-outline btn-sm"
            title="Test this chatbot"
          >
            💬 Test
          </Link>
          <Link
            to={`/chatbots/${chatbot.id}/analytics`}
            className="btn btn-outline btn-sm"
            title="View analytics"
          >
            📊 Analytics
          </Link>
          <Link
            to={`/chatbots/${chatbot.id}/edit`}
            className="btn btn-outline btn-sm"
            title="Edit chatbot"
          >
            ✏️ Edit
          </Link>
        </div>
        <div className="action-group">
          <button
            onClick={onCopyEmbed}
            className="btn btn-primary btn-sm"
            title="Copy embed code"
          >
            📋 Embed Code
          </button>
        </div>
      </div>
    </div>
  );
}

export default ChatbotList;