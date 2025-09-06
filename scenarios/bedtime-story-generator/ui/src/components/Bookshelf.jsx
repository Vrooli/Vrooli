import React from 'react';
import { motion } from 'framer-motion';

const Bookshelf = ({ 
  stories, 
  onSelectStory, 
  onGenerateNew, 
  onToggleFavorite,
  onDeleteStory,
  loading,
  theme 
}) => {
  const bookColors = [
    ['#ff6b6b', '#ff5252'],
    ['#4ecdc4', '#3db5ac'],
    ['#45b7d1', '#2196f3'],
    ['#96d03d', '#7cb342'],
    ['#ffd93d', '#ffc107'],
    ['#a86cc1', '#9c27b0'],
    ['#ff9a56', '#ff7043'],
    ['#6c5ce7', '#5f4bdb'],
    ['#fd79a8', '#e84393'],
    ['#00b894', '#00a381']
  ];

  const getBookColor = (index) => {
    const colorPair = bookColors[index % bookColors.length];
    return {
      '--book-color-1': colorPair[0],
      '--book-color-2': colorPair[1]
    };
  };

  if (loading) {
    return (
      <motion.div
        className="bookshelf"
        initial={{ opacity: 0, y: 50 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
      >
        <h2 className="bookshelf-title">📚 Your Magical Bookshelf</h2>
        <div className="loading-container">
          <div className="spinner" />
          <div className="loading-text">Loading your stories...</div>
        </div>
      </motion.div>
    );
  }

  return (
    <motion.div
      className="bookshelf"
      initial={{ opacity: 0, y: 50 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <h2 className="bookshelf-title">📚 Your Magical Bookshelf</h2>
      
      <div className="books-container">
        {stories.length === 0 ? (
          <div style={{
            gridColumn: '1 / -1',
            textAlign: 'center',
            padding: '40px',
            color: theme.textColor
          }}>
            <p style={{ fontSize: '1.2rem', marginBottom: '20px' }}>
              Your bookshelf is empty! 
            </p>
            <p>Click the magic button below to create your first story!</p>
          </div>
        ) : (
          stories.map((story, index) => (
            <motion.div
              key={story.id}
              className={`book-spine ${story.is_favorite ? 'favorite' : ''}`}
              style={getBookColor(index)}
              initial={{ opacity: 0, rotateY: -90 }}
              animate={{ opacity: 1, rotateY: 0 }}
              transition={{ delay: index * 0.1 }}
              onClick={() => onSelectStory(story)}
              whileHover={{ 
                y: -10,
                transition: { duration: 0.2 }
              }}
              title={story.title}
            >
              <div className="book-title">
                {story.title.length > 20 
                  ? story.title.substring(0, 20) + '...' 
                  : story.title}
              </div>
              
              {/* Book actions */}
              <div className="book-actions" onClick={(e) => e.stopPropagation()}>
                <button
                  className="book-action-btn favorite-btn"
                  onClick={(e) => {
                    e.stopPropagation();
                    onToggleFavorite(story.id);
                  }}
                  style={{
                    position: 'absolute',
                    bottom: '10px',
                    left: '50%',
                    transform: 'translateX(-50%)',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    fontSize: '1rem',
                    opacity: 0.8
                  }}
                >
                  {story.is_favorite ? '💖' : '🤍'}
                </button>
              </div>

              {/* Age badge */}
              <div style={{
                position: 'absolute',
                top: '30px',
                left: '50%',
                transform: 'translateX(-50%)',
                background: 'rgba(255, 255, 255, 0.9)',
                color: '#333',
                padding: '2px 6px',
                borderRadius: '10px',
                fontSize: '0.6rem',
                fontWeight: 'bold'
              }}>
                {story.age_group}
              </div>
            </motion.div>
          ))
        )}
      </div>

      <motion.button
        className="generate-story-btn"
        onClick={onGenerateNew}
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
      >
        ✨ Create New Story
      </motion.button>
    </motion.div>
  );
};

export default Bookshelf;