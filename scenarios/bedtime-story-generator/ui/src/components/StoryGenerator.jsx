import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import axios from 'axios';

const API_URL = process.env.NODE_ENV === 'production' 
  ? '/api' 
  : `http://localhost:${import.meta.env.VITE_API_PORT || '20000'}/api`;

const StoryGenerator = ({ onGenerate, onCancel, theme }) => {
  const [formData, setFormData] = useState({
    age_group: '6-8',
    theme: 'Adventure',
    length: 'medium',
    character_names: []
  });
  const [themes, setThemes] = useState([]);
  const [characterInput, setCharacterInput] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchThemes();
  }, []);

  const fetchThemes = async () => {
    try {
      const response = await axios.get(`${API_URL}/v1/themes`);
      setThemes(response.data || []);
    } catch (error) {
      // Use default themes if fetch fails
      setThemes([
        { name: 'Adventure', emoji: '🗺️' },
        { name: 'Animals', emoji: '🦁' },
        { name: 'Fantasy', emoji: '🦄' },
        { name: 'Space', emoji: '🚀' },
        { name: 'Ocean', emoji: '🐠' },
        { name: 'Forest', emoji: '🌲' },
        { name: 'Friendship', emoji: '🤝' },
        { name: 'Bedtime', emoji: '🌙' },
        { name: 'Dinosaurs', emoji: '🦕' },
        { name: 'Fairy Tales', emoji: '👑' }
      ]);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsGenerating(true);
    setError(null);

    try {
      const characters = characterInput
        .split(',')
        .map(name => name.trim())
        .filter(name => name.length > 0);

      await onGenerate({
        ...formData,
        character_names: characters
      });
    } catch (error) {
      setError('Failed to generate story. Please try again.');
      setIsGenerating(false);
    }
  };

  if (isGenerating) {
    return (
      <motion.div
        className="story-generator"
        initial={{ scale: 0, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ duration: 0.3 }}
      >
        <div className="loading-container">
          <div className="spinner" />
          <div className="loading-text">
            Creating your magical story...
          </div>
          <p style={{ 
            marginTop: '20px', 
            color: '#8b6f47',
            fontSize: '0.9rem'
          }}>
            This may take a few moments while our storyteller 
            crafts something special just for you!
          </p>
        </div>
      </motion.div>
    );
  }

  return (
    <motion.div
      className="story-generator"
      initial={{ scale: 0, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <h2 className="generator-title">✨ Create Your Story</h2>

      {error && (
        <div style={{
          background: '#ffebee',
          color: '#c62828',
          padding: '10px',
          borderRadius: '10px',
          marginBottom: '20px',
          textAlign: 'center'
        }}>
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="age_group">
            👶 Age Group
          </label>
          <select
            id="age_group"
            value={formData.age_group}
            onChange={(e) => setFormData({...formData, age_group: e.target.value})}
          >
            <option value="3-5">3-5 years (Simple & Sweet)</option>
            <option value="6-8">6-8 years (Fun Adventures)</option>
            <option value="9-12">9-12 years (Epic Tales)</option>
          </select>
        </div>

        <div className="form-group">
          <label htmlFor="theme">
            🎨 Story Theme
          </label>
          <select
            id="theme"
            value={formData.theme}
            onChange={(e) => setFormData({...formData, theme: e.target.value})}
          >
            {themes.map(theme => (
              <option key={theme.name} value={theme.name}>
                {theme.emoji} {theme.name}
              </option>
            ))}
          </select>
        </div>

        <div className="form-group">
          <label htmlFor="length">
            ⏱️ Story Length
          </label>
          <select
            id="length"
            value={formData.length}
            onChange={(e) => setFormData({...formData, length: e.target.value})}
          >
            <option value="short">Short (3-5 minutes)</option>
            <option value="medium">Medium (8-10 minutes)</option>
            <option value="long">Long (12-15 minutes)</option>
          </select>
        </div>

        <div className="form-group">
          <label htmlFor="characters">
            🦸 Character Names (optional)
          </label>
          <input
            id="characters"
            type="text"
            placeholder="Emma, Max, Luna (comma-separated)"
            value={characterInput}
            onChange={(e) => setCharacterInput(e.target.value)}
          />
          <small style={{ 
            display: 'block', 
            marginTop: '5px',
            color: '#8b6f47',
            fontSize: '0.85rem'
          }}>
            Add names of children, pets, or favorite characters!
          </small>
        </div>

        <div className="form-actions">
          <button 
            type="button" 
            className="btn btn-secondary"
            onClick={onCancel}
          >
            Cancel
          </button>
          <button 
            type="submit" 
            className="btn btn-primary"
          >
            🪄 Generate Story
          </button>
        </div>
      </form>

      <div style={{
        marginTop: '30px',
        padding: '15px',
        background: 'rgba(102, 126, 234, 0.1)',
        borderRadius: '10px',
        textAlign: 'center'
      }}>
        <p style={{ 
          fontSize: '0.9rem',
          color: '#667eea',
          fontStyle: 'italic'
        }}>
          💡 Tip: Stories are crafted to be calming and perfect for bedtime, 
          with gentle life lessons and peaceful endings.
        </p>
      </div>
    </motion.div>
  );
};

export default StoryGenerator;