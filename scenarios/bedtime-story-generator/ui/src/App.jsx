import React, { useState, useEffect, useMemo } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import axios from 'axios';
import ChildrensRoom from './components/ChildrensRoom';
import Bookshelf from './components/Bookshelf';
import BookReader from './components/BookReader';
import StoryGenerator from './components/StoryGenerator';
import SettingsPanel from './components/SettingsPanel';
import SceneDebugPanel from './components/SceneDebugPanel';
import { getTimeOfDay, getThemeForTime, getGreeting } from './utils/timeUtils';

const API_URL = process.env.NODE_ENV === 'production' 
  ? '/api' 
  : `http://localhost:${import.meta.env.VITE_API_PORT || '20000'}/api`;

function App() {
  const [stories, setStories] = useState([]);
  const [selectedStory, setSelectedStory] = useState(null);
  const [isReading, setIsReading] = useState(false);
  const [generatorOpen, setGeneratorOpen] = useState(false);
  const [libraryOpen, setLibraryOpen] = useState(true);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [floorTexture, setFloorTexture] = useState('smooth');
  const [wallTexture, setWallTexture] = useState('solid');
  const [debugOpen, setDebugOpen] = useState(false);
  const [debugSelection, setDebugSelection] = useState(null);
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
    try {
      const response = await axios.post(`${API_URL}/v1/stories/generate`, params);
      const newStory = response.data;
      setStories([newStory, ...stories]);
      setSelectedStory(newStory);
      setIsReading(true);
      setGeneratorOpen(false);
      setLibraryOpen(true);
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

  const handleCloseLibrary = () => {
    setGeneratorOpen(false);
    setLibraryOpen(false);
  };

  const handleActivateLibrary = () => {
    setLibraryOpen(true);
  };

  const favoriteCount = useMemo(
    () => stories.filter((story) => story.is_favorite).length,
    [stories]
  );

  const lastGenerated = useMemo(() => {
    if (!stories.length) return null;
    return stories.reduce((latest, story) => {
      if (!story.created_at) return latest;
      const created = new Date(story.created_at);
      if (!latest) return created;
      return created > latest ? created : latest;
    }, null);
  }, [stories]);

  return (
    <div className={`app ${theme.class}`}>
      <button
        className="settings-toggle"
        onClick={() => setSettingsOpen(true)}
        aria-label="Open room settings"
      >
        ⚙️
      </button>
      <button
        className="debug-toggle"
        onClick={() => setDebugOpen((open) => !open)}
        aria-label="Toggle scene debugger"
      >
        🧭
      </button>
      <ChildrensRoom
        timeOfDay={timeOfDay}
        theme={theme}
        libraryOpen={libraryOpen}
        onLibraryActivate={handleActivateLibrary}
        floorTextureType={floorTexture}
        wallTextureType={wallTexture}
        debugSelection={debugSelection}
      >
        <AnimatePresence>
          {libraryOpen && (
            <motion.div
              className="hud"
              key="hud"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 20 }}
              transition={{ duration: 0.35, ease: 'easeOut' }}
            >
              <div className="hud-top">
                <div className="hud-card">
                  <div className="hud-card-heading">
                    <span className="hud-greeting">{getGreeting()}</span>
                    <button
                      className="icon-button close-hud"
                      onClick={handleCloseLibrary}
                      aria-label="Hide library"
                    >
                      ✕
                    </button>
                  </div>
                  <p className="hud-subtext">
                    {timeOfDay === 'night'
                      ? 'Lights are low, stories are cozy. Perfect for winding down.'
                      : 'Settle in for a calm reading adventure tailored to bedtime.'}
                  </p>
                  <div className="hud-stats">
                    <div className="hud-stat">
                      <span className="stat-label">Favorites</span>
                      <span className="stat-value">{favoriteCount}</span>
                    </div>
                    <div className="hud-stat">
                      <span className="stat-label">Stories Saved</span>
                      <span className="stat-value">{stories.length}</span>
                    </div>
                    <div className="hud-stat">
                      <span className="stat-label">Last Story</span>
                      <span className="stat-value">
                        {lastGenerated
                          ? lastGenerated.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
                          : '—'}
                      </span>
                    </div>
                  </div>
                </div>
                <button
                  className="primary-action"
                  onClick={() => setGeneratorOpen(true)}
                >
                  ✨ Craft A New Story
                </button>
              </div>

              <Bookshelf
                stories={stories}
                onSelectStory={handleSelectStory}
                onGenerateNew={() => setGeneratorOpen(true)}
                onToggleFavorite={handleToggleFavorite}
                onDeleteStory={handleDeleteStory}
                loading={loading}
                onCloseLibrary={handleCloseLibrary}
              />
            </motion.div>
          )}
        </AnimatePresence>
      </ChildrensRoom>

      <AnimatePresence>
        {generatorOpen && (
          <StoryGenerator
            key="generator"
            onGenerate={handleGenerateStory}
            onCancel={() => setGeneratorOpen(false)}
          />
        )}
        {isReading && selectedStory && (
          <BookReader
            key="reader"
            story={selectedStory}
            onClose={handleCloseBook}
          />
        )}
        {settingsOpen && (
          <SettingsPanel
            key="settings"
            floorTexture={floorTexture}
            wallTexture={wallTexture}
            onChangeFloor={setFloorTexture}
            onChangeWall={setWallTexture}
            onClose={() => setSettingsOpen(false)}
          />
        )}
        {debugOpen && (
          <SceneDebugPanel
            key="debug"
            selectedId={debugSelection}
            onSelect={(id) => setDebugSelection(id)}
            onClose={() => {
              setDebugOpen(false);
              setDebugSelection(null);
            }}
          />
        )}
      </AnimatePresence>
    </div>
  );
}

export default App;
  useEffect(() => {
    if (!debugOpen) {
      setDebugSelection(null);
    }
  }, [debugOpen]);
