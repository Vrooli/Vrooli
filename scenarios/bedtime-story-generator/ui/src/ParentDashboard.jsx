import React, { useState, useEffect } from 'react';
import './ParentDashboard.css';

const ParentDashboard = ({ onClose }) => {
  const [stories, setStories] = useState([]);
  const [selectedStory, setSelectedStory] = useState(null);
  const [filterAgeGroup, setFilterAgeGroup] = useState('all');
  const [sortBy, setSortBy] = useState('created_at');
  const [isLoading, setIsLoading] = useState(true);
  const [stats, setStats] = useState({
    totalStories: 0,
    totalReadingTime: 0,
    favoriteCount: 0,
    mostReadStory: null
  });

  useEffect(() => {
    fetchStories();
    calculateStats();
  }, [filterAgeGroup, sortBy]);

  const fetchStories = async () => {
    setIsLoading(true);
    try {
      const apiUrl = import.meta.env.VITE_API_URL || `http://localhost:${window.location.port.replace('300', '69')}`;
      const response = await fetch(`${apiUrl}/api/v1/stories`);
      if (response.ok) {
        let data = await response.json();
        
        // Apply filters
        if (filterAgeGroup !== 'all') {
          data = data.filter(story => story.age_group === filterAgeGroup);
        }
        
        // Apply sorting
        data.sort((a, b) => {
          switch (sortBy) {
            case 'times_read':
              return b.times_read - a.times_read;
            case 'title':
              return a.title.localeCompare(b.title);
            case 'created_at':
            default:
              return new Date(b.created_at) - new Date(a.created_at);
          }
        });
        
        setStories(data);
      }
    } catch (error) {
      console.error('Failed to fetch stories:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const calculateStats = () => {
    const totalStories = stories.length;
    const totalReadingTime = stories.reduce((acc, story) => 
      acc + (story.times_read * story.reading_time_minutes), 0);
    const favoriteCount = stories.filter(s => s.is_favorite).length;
    const mostReadStory = stories.reduce((max, story) => 
      !max || story.times_read > max.times_read ? story : max, null);
    
    setStats({
      totalStories,
      totalReadingTime,
      favoriteCount,
      mostReadStory
    });
  };

  const deleteStory = async (storyId) => {
    if (!window.confirm('Are you sure you want to delete this story?')) {
      return;
    }
    
    try {
      const apiUrl = import.meta.env.VITE_API_URL || `http://localhost:${window.location.port.replace('300', '69')}`;
      const response = await fetch(`${apiUrl}/api/v1/stories/${storyId}`, {
        method: 'DELETE'
      });
      
      if (response.ok) {
        setStories(stories.filter(s => s.id !== storyId));
        if (selectedStory?.id === storyId) {
          setSelectedStory(null);
        }
      }
    } catch (error) {
      console.error('Failed to delete story:', error);
    }
  };

  const exportStory = async (story) => {
    // Use the API export endpoint that generates HTML for PDF printing
    const apiUrl = import.meta.env.VITE_API_URL || `http://localhost:${window.location.port.replace('300', '69')}`;
    const exportUrl = `${apiUrl}/api/v1/stories/${story.id}/export`;
    
    // Open in new window for direct download
    window.open(exportUrl, '_blank');
  };

  return (
    <div className="parent-dashboard">
      <div className="dashboard-header">
        <h1>Parent Dashboard</h1>
        <button className="close-btn" onClick={onClose}>×</button>
      </div>
      
      <div className="dashboard-stats">
        <div className="stat-card">
          <div className="stat-value">{stats.totalStories}</div>
          <div className="stat-label">Total Stories</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.totalReadingTime}</div>
          <div className="stat-label">Minutes Read</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.favoriteCount}</div>
          <div className="stat-label">Favorites</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.mostReadStory?.title || 'None'}</div>
          <div className="stat-label">Most Read Story</div>
        </div>
      </div>
      
      <div className="dashboard-controls">
        <select 
          value={filterAgeGroup} 
          onChange={(e) => setFilterAgeGroup(e.target.value)}
          className="filter-select"
        >
          <option value="all">All Ages</option>
          <option value="3-5">Ages 3-5</option>
          <option value="6-8">Ages 6-8</option>
          <option value="9-12">Ages 9-12</option>
        </select>
        
        <select 
          value={sortBy} 
          onChange={(e) => setSortBy(e.target.value)}
          className="sort-select"
        >
          <option value="created_at">Newest First</option>
          <option value="times_read">Most Read</option>
          <option value="title">Alphabetical</option>
        </select>
      </div>
      
      <div className="dashboard-content">
        <div className="stories-list">
          <h2>Story Library</h2>
          {isLoading ? (
            <div className="loading">Loading stories...</div>
          ) : stories.length === 0 ? (
            <div className="no-stories">No stories found</div>
          ) : (
            <div className="story-cards">
              {stories.map(story => (
                <div 
                  key={story.id} 
                  className={`story-card ${selectedStory?.id === story.id ? 'selected' : ''}`}
                  onClick={() => setSelectedStory(story)}
                >
                  <div className="story-card-header">
                    <h3>{story.title}</h3>
                    {story.is_favorite && <span className="favorite-star">⭐</span>}
                  </div>
                  <div className="story-card-meta">
                    <span className="age-badge">{story.age_group}</span>
                    <span className="theme-badge">{story.theme}</span>
                    <span className="read-count">Read {story.times_read}x</span>
                  </div>
                  <div className="story-card-actions">
                    <button 
                      onClick={(e) => {
                        e.stopPropagation();
                        exportStory(story);
                      }}
                      className="export-btn"
                    >
                      Export
                    </button>
                    <button 
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteStory(story.id);
                      }}
                      className="delete-btn"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
        
        {selectedStory && (
          <div className="story-preview">
            <h2>{selectedStory.title}</h2>
            <div className="story-details">
              <p><strong>Age Group:</strong> {selectedStory.age_group}</p>
              <p><strong>Theme:</strong> {selectedStory.theme}</p>
              <p><strong>Reading Time:</strong> {selectedStory.reading_time_minutes} minutes</p>
              <p><strong>Created:</strong> {new Date(selectedStory.created_at).toLocaleDateString()}</p>
              <p><strong>Times Read:</strong> {selectedStory.times_read}</p>
            </div>
            <div className="story-content">
              <h3>Content Preview</h3>
              <pre>{selectedStory.content}</pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ParentDashboard;