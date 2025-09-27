import React, { useState } from "react";
import useExperienceStore from "../state/store.js";
import PhotoMode from "./PhotoMode.jsx";

const SceneControls = ({ experience }) => {
  const [audioEnabled, setAudioEnabled] = useState(true);
  const [volume, setVolume] = useState(70);
  const [debugMode, setDebugMode] = useState(false);
  const [showPhotoMode, setShowPhotoMode] = useState(false);
  const { timeOfDay, setTimeOfDay } = useExperienceStore();

  const handleCameraPreset = (preset) => {
    if (!experience?.cameraRailSystem) return;
    
    const railMap = {
      intro: 'intro',
      bookshelf: 'bookshelfFocus',
      window: 'windowPan',
      story: 'storyOrbit',
      lamp: 'lampZoom',
      toyChest: 'toyChestReveal'
    };
    
    const railName = railMap[preset];
    if (railName === 'storyOrbit') {
      experience.cameraRailSystem.playRail(railName, { loop: true });
    } else {
      experience.cameraRailSystem.playRail(railName);
    }
  };

  const handleTimeChange = (newTime) => {
    setTimeOfDay(newTime);
    if (experience?.audioAmbience) {
      experience.audioAmbience.setTimeOfDay(newTime);
    }
  };

  const toggleAudio = () => {
    if (!experience?.audioAmbience) return;
    const newState = experience.audioAmbience.toggleEnabled();
    setAudioEnabled(newState);
  };

  const handleVolumeChange = (e) => {
    const newVolume = parseInt(e.target.value);
    setVolume(newVolume);
    if (experience?.audioAmbience) {
      experience.audioAmbience.setMasterVolume(newVolume / 100);
    }
  };

  const toggleDebug = () => {
    if (!experience?.cameraRailSystem) return;
    const newState = !debugMode;
    setDebugMode(newState);
    
    if (newState) {
      experience.cameraRailSystem.enableDebug();
    } else {
      experience.cameraRailSystem.disableDebug();
    }
  };

  const stopCamera = () => {
    if (experience?.cameraRailSystem) {
      experience.cameraRailSystem.stopRail();
    }
  };

  return (
    <div className="scene-controls">
      <div className="scene-controls__section">
        <h4>Camera Views</h4>
        <div className="scene-controls__buttons">
          <button onClick={() => handleCameraPreset('intro')} className="scene-btn">
            Intro
          </button>
          <button onClick={() => handleCameraPreset('bookshelf')} className="scene-btn">
            Bookshelf
          </button>
          <button onClick={() => handleCameraPreset('window')} className="scene-btn">
            Window
          </button>
          <button onClick={() => handleCameraPreset('story')} className="scene-btn">
            Story Orbit
          </button>
          <button onClick={() => handleCameraPreset('lamp')} className="scene-btn">
            Reading Lamp
          </button>
          <button onClick={() => handleCameraPreset('toyChest')} className="scene-btn">
            Toy Chest
          </button>
          <button onClick={stopCamera} className="scene-btn scene-btn--stop">
            Stop
          </button>
        </div>
      </div>

      <div className="scene-controls__section">
        <h4>Time of Day</h4>
        <div className="scene-controls__buttons">
          <button 
            onClick={() => handleTimeChange('day')} 
            className={`scene-btn ${timeOfDay === 'day' ? 'scene-btn--active' : ''}`}
          >
            Day
          </button>
          <button 
            onClick={() => handleTimeChange('evening')} 
            className={`scene-btn ${timeOfDay === 'evening' ? 'scene-btn--active' : ''}`}
          >
            Evening
          </button>
          <button 
            onClick={() => handleTimeChange('night')} 
            className={`scene-btn ${timeOfDay === 'night' ? 'scene-btn--active' : ''}`}
          >
            Night
          </button>
        </div>
      </div>

      <div className="scene-controls__section">
        <h4>Audio Ambience</h4>
        <div className="scene-controls__audio">
          <button onClick={toggleAudio} className="scene-btn">
            {audioEnabled ? '🔊 On' : '🔇 Off'}
          </button>
          <input
            type="range"
            min="0"
            max="100"
            value={volume}
            onChange={handleVolumeChange}
            disabled={!audioEnabled}
            className="scene-controls__volume"
          />
          <span>{volume}%</span>
        </div>
      </div>

      <div className="scene-controls__section">
        <h4>Developer</h4>
        <div className="scene-controls__buttons">
          <button onClick={toggleDebug} className="scene-btn">
            {debugMode ? '✅ Debug On' : '⬜ Debug Off'}
          </button>
          <button onClick={() => setShowPhotoMode(!showPhotoMode)} className="scene-btn">
            📸 Photo Mode
          </button>
        </div>
      </div>

      {showPhotoMode && (
        <PhotoMode 
          experience={experience}
          onClose={() => setShowPhotoMode(false)}
        />
      )}

      <style jsx>{`
        .scene-controls {
          position: absolute;
          bottom: 20px;
          left: 20px;
          background: rgba(20, 20, 40, 0.9);
          border-radius: 12px;
          padding: 20px;
          color: white;
          font-family: monospace;
          max-width: 300px;
          backdrop-filter: blur(10px);
          z-index: 1000;
        }

        .scene-controls__section {
          margin-bottom: 20px;
        }

        .scene-controls__section:last-child {
          margin-bottom: 0;
        }

        .scene-controls h4 {
          margin: 0 0 10px 0;
          font-size: 14px;
          color: #b8b8ff;
          text-transform: uppercase;
          letter-spacing: 1px;
        }

        .scene-controls__buttons {
          display: flex;
          flex-wrap: wrap;
          gap: 8px;
        }

        .scene-btn {
          background: rgba(88, 88, 255, 0.2);
          color: white;
          border: 1px solid rgba(88, 88, 255, 0.4);
          border-radius: 6px;
          padding: 6px 12px;
          font-size: 12px;
          cursor: pointer;
          transition: all 0.3s ease;
          font-family: monospace;
        }

        .scene-btn:hover {
          background: rgba(88, 88, 255, 0.4);
          border-color: rgba(88, 88, 255, 0.6);
        }

        .scene-btn--active {
          background: rgba(88, 88, 255, 0.6);
          border-color: rgba(88, 88, 255, 1);
        }

        .scene-btn--stop {
          background: rgba(255, 88, 88, 0.2);
          border-color: rgba(255, 88, 88, 0.4);
        }

        .scene-btn--stop:hover {
          background: rgba(255, 88, 88, 0.4);
          border-color: rgba(255, 88, 88, 0.6);
        }

        .scene-controls__audio {
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .scene-controls__volume {
          flex: 1;
          height: 4px;
          -webkit-appearance: none;
          appearance: none;
          background: rgba(88, 88, 255, 0.2);
          border-radius: 2px;
          outline: none;
        }

        .scene-controls__volume::-webkit-slider-thumb {
          -webkit-appearance: none;
          appearance: none;
          width: 12px;
          height: 12px;
          background: #8888ff;
          border-radius: 50%;
          cursor: pointer;
        }

        .scene-controls__volume::-moz-range-thumb {
          width: 12px;
          height: 12px;
          background: #8888ff;
          border-radius: 50%;
          cursor: pointer;
          border: none;
        }

        .scene-controls__volume:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      `}</style>
    </div>
  );
};

export default SceneControls;