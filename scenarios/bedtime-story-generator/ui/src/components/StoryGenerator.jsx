import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import axios from 'axios';

const API_URL = process.env.NODE_ENV === 'production'
  ? '/api'
  : `http://localhost:${import.meta.env.VITE_API_PORT || '20000'}/api`;

const overlayVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
  exit: { opacity: 0 }
};

const panelVariants = {
  hidden: { opacity: 0, y: 40, scale: 0.96 },
  visible: { opacity: 1, y: 0, scale: 1 },
  exit: { opacity: 0, y: 20, scale: 0.98 }
};

const AGE_OPTIONS = [
  { value: '3-5', label: '3-5 years', helper: 'Simple & sweet', icon: '🧸' },
  { value: '6-8', label: '6-8 years', helper: 'Fun adventures', icon: '🧭' },
  { value: '9-12', label: '9-12 years', helper: 'Epic tales', icon: '🐉' }
];

const LENGTH_OPTIONS = [
  { value: 'short', label: 'Short', helper: '3-5 minutes', icon: '⏱️' },
  { value: 'medium', label: 'Medium', helper: '8-10 minutes', icon: '🕯️' },
  { value: 'long', label: 'Long', helper: '12-15 minutes', icon: '🌌' }
];

const StoryGenerator = ({ onGenerate, onCancel }) => {
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
    } catch (fetchError) {
      setThemes([
        { name: 'Adventure', emoji: '🗺️' },
        { name: 'Animals', emoji: '🦁' },
        { name: 'Fantasy', emoji: '🦄' },
        { name: 'Space', emoji: '🚀' },
        { name: 'Friendship', emoji: '🤝' },
        { name: 'Ocean', emoji: '🐠' }
      ]);
    }
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    setIsGenerating(true);
    setError(null);

    try {
      const characters = characterInput
        .split(',')
        .map((name) => name.trim())
        .filter(Boolean);

      await onGenerate({
        ...formData,
        character_names: characters
      });
    } catch (submitError) {
      setError('The storyteller stumbled. Please try again.');
      setIsGenerating(false);
    }
  };

  const selectTheme = (name) => {
    setFormData((prev) => ({ ...prev, theme: name }));
  };

  const selectAge = (value) => {
    setFormData((prev) => ({ ...prev, age_group: value }));
  };

  const selectLength = (value) => {
    setFormData((prev) => ({ ...prev, length: value }));
  };

  return (
    <motion.div
      className="modal-overlay"
      initial="hidden"
      animate="visible"
      exit="exit"
      variants={overlayVariants}
    >
      <motion.div
        className="story-generator-panel"
        role="dialog"
        aria-modal="true"
        variants={panelVariants}
      >
        <button className="icon-button close" onClick={onCancel} aria-label="Close story generator">
          ✕
        </button>

        <header className="panel-heading">
          <h2>✨ Compose a bedtime tale</h2>
          <p>Weave a calming adventure tailored to tonight’s routine.</p>
        </header>

        {error && (
          <div className="form-alert" role="alert">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <fieldset disabled={isGenerating}>
            <legend className="visually-hidden">Story options</legend>

            <div className="control-group">
              <span className="control-label">Age focus</span>
              <div className="chip-grid">
                {AGE_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    type="button"
                    className={`chip ${formData.age_group === option.value ? 'active' : ''}`}
                    onClick={() => selectAge(option.value)}
                  >
                    <span className="chip-icon">{option.icon}</span>
                    <div>
                      <div>{option.label}</div>
                      <small>{option.helper}</small>
                    </div>
                  </button>
                ))}
              </div>
            </div>

            <div className="control-group">
              <span className="control-label">Story tone</span>
              <div className="chip-row">
                {themes.map((option) => (
                  <button
                    key={option.name}
                    type="button"
                    className={`pill ${formData.theme === option.name ? 'active' : ''}`}
                    onClick={() => selectTheme(option.name)}
                  >
                    {option.emoji} {option.name}
                  </button>
                ))}
              </div>
            </div>

            <div className="control-group">
              <span className="control-label">Length</span>
              <div className="chip-grid length">
                {LENGTH_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    type="button"
                    className={`chip ${formData.length === option.value ? 'active' : ''}`}
                    onClick={() => selectLength(option.value)}
                  >
                    <span className="chip-icon">{option.icon}</span>
                    <div>
                      <div>{option.label}</div>
                      <small>{option.helper}</small>
                    </div>
                  </button>
                ))}
              </div>
            </div>

            <div className="control-group">
              <label className="control-label" htmlFor="characters">Character names</label>
              <input
                id="characters"
                type="text"
                placeholder="Emma, Max, Luna (optional)"
                value={characterInput}
                onChange={(event) => setCharacterInput(event.target.value)}
              />
              <small>Add children, pets, or favourite characters to personalise the tale.</small>
            </div>
          </fieldset>

          <div className="panel-actions">
            <button type="button" className="ghost-action" onClick={onCancel}>
              Cancel
            </button>
            <button type="submit" className="primary-action" disabled={isGenerating}>
              {isGenerating ? 'Crafting…' : 'Generate story'}
            </button>
          </div>
        </form>

        {isGenerating && (
          <div className="panel-loading-overlay" aria-live="polite">
            <div className="spinner large" />
            <p>Our storyteller is weaving gentle dreams…</p>
          </div>
        )}
      </motion.div>
    </motion.div>
  );
};

export default StoryGenerator;
