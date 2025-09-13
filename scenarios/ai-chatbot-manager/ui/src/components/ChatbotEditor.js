import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import apiClient from '../utils/api';

function ChatbotEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEditing = !!id;

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    personality: 'You are a helpful assistant.',
    knowledge_base: '',
    model_config: {
      model: 'llama3.2',
      temperature: 0.7,
      max_tokens: 1000
    },
    widget_config: {
      theme: 'light',
      position: 'bottom-right',
      primaryColor: '#3b82f6',
      greeting: 'Hi! How can I help you today?'
    }
  });

  useEffect(() => {
    if (isEditing) {
      loadChatbot();
    }
  }, [id, isEditing]);

  const loadChatbot = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get(`/api/v1/chatbots/${id}`);
      if (response.ok) {
        const chatbot = await response.json();
        setFormData({
          name: chatbot.name || '',
          description: chatbot.description || '',
          personality: chatbot.personality || 'You are a helpful assistant.',
          knowledge_base: chatbot.knowledge_base || '',
          model_config: {
            model: chatbot.model_config?.model || 'llama3.2',
            temperature: chatbot.model_config?.temperature || 0.7,
            max_tokens: chatbot.model_config?.max_tokens || 1000
          },
          widget_config: {
            theme: chatbot.widget_config?.theme || 'light',
            position: chatbot.widget_config?.position || 'bottom-right',
            primaryColor: chatbot.widget_config?.primaryColor || '#3b82f6',
            greeting: chatbot.widget_config?.greeting || 'Hi! How can I help you today?'
          }
        });
      } else {
        alert('Failed to load chatbot');
        navigate('/chatbots');
      }
    } catch (error) {
      console.error('Failed to load chatbot:', error);
      alert('Failed to load chatbot');
      navigate('/chatbots');
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleModelConfigChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      model_config: {
        ...prev.model_config,
        [field]: value
      }
    }));
  };

  const handleWidgetConfigChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      widget_config: {
        ...prev.widget_config,
        [field]: value
      }
    }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    if (!formData.name.trim()) {
      alert('Please enter a chatbot name');
      return;
    }

    try {
      setSaving(true);
      
      const method = isEditing ? 'PUT' : 'POST';
      const url = isEditing ? `/api/v1/chatbots/${id}` : '/api/v1/chatbots';
      
      const response = isEditing 
        ? await apiClient.put(`/api/v1/chatbots/${id}`, formData)
        : await apiClient.post('/api/v1/chatbots', formData);

      if (response.ok) {
        const result = await response.json();
        alert(isEditing ? 'Chatbot updated successfully!' : 'Chatbot created successfully!');
        
        if (!isEditing && result.chatbot) {
          // Show embed code for new chatbots
          const embedCode = `<script src="${window.location.origin}/widget.js" data-chatbot-id="${result.chatbot.id}"></script>`;
          if (confirm('Chatbot created! Would you like to copy the embed code to clipboard?')) {
            navigator.clipboard.writeText(embedCode);
          }
        }
        
        navigate('/chatbots');
      } else {
        const error = await response.text();
        alert(`Failed to ${isEditing ? 'update' : 'create'} chatbot: ${error}`);
      }
    } catch (error) {
      console.error('Failed to save chatbot:', error);
      alert(`Failed to ${isEditing ? 'update' : 'create'} chatbot`);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner"></div>
        <p>Loading chatbot...</p>
      </div>
    );
  }

  return (
    <div className="chatbot-editor">
      <div className="page-header">
        <h1 className="page-title">
          {isEditing ? 'Edit Chatbot' : 'Create New Chatbot'}
        </h1>
        <p className="page-subtitle">
          {isEditing 
            ? 'Update your chatbot configuration and personality'
            : 'Configure your AI-powered chatbot for your website'
          }
        </p>
      </div>

      <form onSubmit={handleSubmit} className="chatbot-form">
        <div className="form-sections">
          {/* Basic Information */}
          <div className="card form-section">
            <div className="card-header">
              <h2 className="card-title">Basic Information</h2>
            </div>
            
            <div className="form-group">
              <label className="form-label">Chatbot Name *</label>
              <input
                type="text"
                className="form-input"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
                placeholder="e.g., Sales Assistant, Support Bot"
                required
              />
              <div className="form-help">
                A descriptive name for your chatbot
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Description</label>
              <textarea
                className="form-textarea"
                value={formData.description}
                onChange={(e) => handleInputChange('description', e.target.value)}
                placeholder="Brief description of what this chatbot does"
                rows={3}
              />
              <div className="form-help">
                Optional description to help you identify this chatbot
              </div>
            </div>
          </div>

          {/* AI Configuration */}
          <div className="card form-section">
            <div className="card-header">
              <h2 className="card-title">AI Configuration</h2>
            </div>

            <div className="form-group">
              <label className="form-label">Personality & Instructions *</label>
              <textarea
                className="form-textarea"
                value={formData.personality}
                onChange={(e) => handleInputChange('personality', e.target.value)}
                placeholder="You are a helpful sales assistant who..."
                rows={5}
                required
              />
              <div className="form-help">
                Define how your chatbot should behave and respond to users
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Knowledge Base</label>
              <textarea
                className="form-textarea"
                value={formData.knowledge_base}
                onChange={(e) => handleInputChange('knowledge_base', e.target.value)}
                placeholder="Add specific information about your products, services, or company..."
                rows={6}
              />
              <div className="form-help">
                Domain-specific information the chatbot can reference when answering questions
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">AI Model</label>
                <select
                  className="form-select"
                  value={formData.model_config.model}
                  onChange={(e) => handleModelConfigChange('model', e.target.value)}
                >
                  <option value="llama3.2">Llama 3.2 (Recommended)</option>
                  <option value="mistral">Mistral</option>
                  <option value="codellama">CodeLlama</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Temperature</label>
                <input
                  type="number"
                  className="form-input"
                  value={formData.model_config.temperature}
                  onChange={(e) => handleModelConfigChange('temperature', parseFloat(e.target.value))}
                  min="0"
                  max="1"
                  step="0.1"
                />
                <div className="form-help">
                  Creativity level (0 = focused, 1 = creative)
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Max Tokens</label>
                <input
                  type="number"
                  className="form-input"
                  value={formData.model_config.max_tokens}
                  onChange={(e) => handleModelConfigChange('max_tokens', parseInt(e.target.value))}
                  min="100"
                  max="4000"
                  step="100"
                />
                <div className="form-help">
                  Maximum response length
                </div>
              </div>
            </div>
          </div>

          {/* Widget Appearance */}
          <div className="card form-section">
            <div className="card-header">
              <h2 className="card-title">Widget Appearance</h2>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Theme</label>
                <select
                  className="form-select"
                  value={formData.widget_config.theme}
                  onChange={(e) => handleWidgetConfigChange('theme', e.target.value)}
                >
                  <option value="light">Light</option>
                  <option value="dark">Dark</option>
                  <option value="professional">Professional</option>
                  <option value="modern">Modern</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Position</label>
                <select
                  className="form-select"
                  value={formData.widget_config.position}
                  onChange={(e) => handleWidgetConfigChange('position', e.target.value)}
                >
                  <option value="bottom-right">Bottom Right</option>
                  <option value="bottom-left">Bottom Left</option>
                  <option value="center">Center</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">Primary Color</label>
                <input
                  type="color"
                  className="form-input"
                  value={formData.widget_config.primaryColor}
                  onChange={(e) => handleWidgetConfigChange('primaryColor', e.target.value)}
                />
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Greeting Message</label>
              <input
                type="text"
                className="form-input"
                value={formData.widget_config.greeting}
                onChange={(e) => handleWidgetConfigChange('greeting', e.target.value)}
                placeholder="Hi! How can I help you today?"
              />
              <div className="form-help">
                The first message users will see when opening the chat
              </div>
            </div>
          </div>
        </div>

        {/* Form Actions */}
        <div className="form-actions">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => navigate('/chatbots')}
            disabled={saving}
          >
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={saving}
          >
            {saving ? (
              <>
                <div className="loading-spinner small"></div>
                {isEditing ? 'Updating...' : 'Creating...'}
              </>
            ) : (
              isEditing ? 'Update Chatbot' : 'Create Chatbot'
            )}
          </button>
        </div>
      </form>
    </div>
  );
}

export default ChatbotEditor;