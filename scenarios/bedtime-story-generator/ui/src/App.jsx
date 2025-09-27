import React, { useState, useEffect } from 'react';
import axios from 'axios';
import Bookshelf from './components/Bookshelf';
import BookReader from './components/BookReader';
import StoryGenerator from './components/StoryGenerator';
import ParentDashboard from './ParentDashboard';
import { getTimeOfDay, getThemeForTime, getGreeting } from './utils/timeUtils';

const API_URL = process.env.NODE_ENV === 'production' 
  ? '/api' 
  : `http://localhost:16902/api`;

function App() {
  const [stories, setStories] = useState([]);
  const [selectedStory, setSelectedStory] = useState(null);
  const [isReading, setIsReading] = useState(false);
  const [generatorOpen, setGeneratorOpen] = useState(false);
  const [parentDashboardOpen, setParentDashboardOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [theme] = useState(getThemeForTime());

  useEffect(() => {
    fetchStories();
  }, []);

  const fetchStories = async () => {
    try {
      console.log('Fetching stories from:', `${API_URL}/v1/stories`);
      const response = await axios.get(`${API_URL}/v1/stories`);
      console.log('Stories response:', response.data);
      setStories(response.data || []);
    } catch (error) {
      console.error('Failed to fetch stories:', error);
      // Try direct API URL as fallback
      try {
        const directResponse = await axios.get('http://localhost:16902/api/v1/stories');
        console.log('Direct API response:', directResponse.data);
        setStories(directResponse.data || []);
      } catch (directError) {
        console.error('Direct API also failed:', directError);
        setStories([]);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSelectStory = async (story) => {
    setSelectedStory(story);
    setIsReading(true);
    try {
      await axios.post(`${API_URL}/v1/stories/${story.id}/read`);
    } catch (error) {
      console.error('Failed to track reading:', error);
    }
  };

  const handleCloseBook = () => {
    setIsReading(false);
    setTimeout(() => setSelectedStory(null), 500);
  };

  const handleGenerateStory = async (params) => {
    try {
      const response = await axios.post(`${API_URL}/v1/stories/generate`, params);
      const newStory = response.data;
      setStories([newStory, ...stories]);
      setSelectedStory(newStory);
      setIsReading(true);
      setGeneratorOpen(false);
      return newStory;
    } catch (error) {
      console.error('Failed to generate story:', error);
      throw error;
    }
  };

  const handleToggleFavorite = async (storyId) => {
    try {
      await axios.post(`${API_URL}/v1/stories/${storyId}/favorite`);
      setStories(stories.map(s => 
        s.id === storyId ? { ...s, is_favorite: !s.is_favorite } : s
      ));
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
    }
  };

  return (
    <div className={`app ${theme}`} style={{ 
      minHeight: '100vh', 
      background: 'linear-gradient(180deg, #bce3ff 0%, #f5fbff 40%, #fef9f2 100%)',
      padding: '2rem'
    }}>
      <div style={{ maxWidth: '1200px', margin: '0 auto' }}>
        <h1 style={{ fontSize: '2.5rem', marginBottom: '1rem', color: '#1c3550' }}>
          {getGreeting()} 📚
        </h1>
        <p style={{ fontSize: '1.2rem', marginBottom: '2rem', color: '#4a5568' }}>
          Welcome to the Bedtime Story Generator
        </p>

        <div style={{ marginBottom: '2rem' }}>
          <button
            onClick={() => setGeneratorOpen(true)}
            style={{
              background: 'linear-gradient(135deg, #4f6dff, #a36bff)',
              color: 'white',
              border: 'none',
              padding: '12px 24px',
              borderRadius: '999px',
              fontSize: '1.1rem',
              fontWeight: '600',
              cursor: 'pointer',
              marginRight: '1rem'
            }}
          >
            ✨ Generate New Story
          </button>
          <button
            onClick={() => setParentDashboardOpen(true)}
            style={{
              background: 'linear-gradient(135deg, #667eea, #764ba2)',
              color: 'white',
              border: 'none',
              padding: '12px 24px',
              borderRadius: '999px',
              fontSize: '1.1rem',
              fontWeight: '600',
              cursor: 'pointer'
            }}
          >
            👨‍👩‍👧‍👦 Parent Dashboard
          </button>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '4rem' }}>
            <div className="spinner large"></div>
            <p style={{ marginTop: '1rem' }}>Loading stories...</p>
          </div>
        ) : (
          <Bookshelf
            stories={stories}
            onSelectStory={handleSelectStory}
            onToggleFavorite={handleToggleFavorite}
            loading={loading}
          />
        )}

        {generatorOpen && (
          <StoryGenerator
            onGenerate={handleGenerateStory}
            onClose={() => setGeneratorOpen(false)}
          />
        )}

        {isReading && selectedStory && (
          <BookReader
            story={selectedStory}
            onClose={handleCloseBook}
          />
        )}
        
        {parentDashboardOpen && (
          <ParentDashboard
            onClose={() => setParentDashboardOpen(false)}
          />
        )}
      </div>
    </div>
  );
}

export default App;