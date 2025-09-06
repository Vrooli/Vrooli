import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import axios from 'axios';
import ChildrensRoom from './components/ChildrensRoom';
import Bookshelf from './components/Bookshelf';
import BookReader from './components/BookReader';
import StoryGenerator from './components/StoryGenerator';
import { getTimeOfDay, getThemeForTime } from './utils/timeUtils';

const API_URL = process.env.NODE_ENV === 'production' 
  ? '/api' 
  : `http://localhost:${import.meta.env.VITE_API_PORT || '20000'}/api`;

function App() {
  const [stories, setStories] = useState([]);
  const [selectedStory, setSelectedStory] = useState(null);
  const [isReading, setIsReading] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [timeOfDay, setTimeOfDay] = useState(getTimeOfDay());
  const [theme, setTheme] = useState(getThemeForTime());
  const [loading, setLoading] = useState(true);

  // Update time of day every minute
  useEffect(() => {
    const interval = setInterval(() => {
      setTimeOfDay(getTimeOfDay());
      setTheme(getThemeForTime());
    }, 60000);
    return () => clearInterval(interval);
  }, []);

  // Fetch stories on mount
  useEffect(() => {
    fetchStories();
  }, []);

  const fetchStories = async () => {
    try {
      const response = await axios.get(`${API_URL}/v1/stories`);
      setStories(response.data || []);
    } catch (error) {
      console.error('Failed to fetch stories:', error);
      setStories([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectStory = async (story) => {
    setSelectedStory(story);
    setIsReading(true);
    
    // Track reading session
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
    setIsGenerating(true);
    try {
      const response = await axios.post(`${API_URL}/v1/stories/generate`, params);
      const newStory = response.data;
      setStories([newStory, ...stories]);
      setSelectedStory(newStory);
      setIsReading(true);
      setIsGenerating(false);
      return newStory;
    } catch (error) {
      console.error('Failed to generate story:', error);
      setIsGenerating(false);
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

  const handleDeleteStory = async (storyId) => {
    try {
      await axios.delete(`${API_URL}/v1/stories/${storyId}`);
      setStories(stories.filter(s => s.id !== storyId));
      if (selectedStory?.id === storyId) {
        handleCloseBook();
      }
    } catch (error) {
      console.error('Failed to delete story:', error);
    }
  };

  return (
    <div className={`app ${theme.class}`} style={{ 
      background: theme.background,
      minHeight: '100vh',
      transition: 'all 1s ease'
    }}>
      <ChildrensRoom timeOfDay={timeOfDay} theme={theme}>
        <AnimatePresence mode="wait">
          {isReading && selectedStory ? (
            <BookReader
              key="reader"
              story={selectedStory}
              onClose={handleCloseBook}
              theme={theme}
            />
          ) : isGenerating ? (
            <StoryGenerator
              key="generator"
              onGenerate={handleGenerateStory}
              onCancel={() => setIsGenerating(false)}
              theme={theme}
            />
          ) : (
            <Bookshelf
              key="bookshelf"
              stories={stories}
              onSelectStory={handleSelectStory}
              onGenerateNew={() => setIsGenerating(true)}
              onToggleFavorite={handleToggleFavorite}
              onDeleteStory={handleDeleteStory}
              loading={loading}
              theme={theme}
            />
          )}
        </AnimatePresence>
      </ChildrensRoom>
    </div>
  );
}

export default App;