import React, { useState, useEffect, useRef } from "react";

/**
 * Parent Narration Mode Component
 * Allows parents to narrate stories with text-to-speech
 * Audio plays through the 3D environment's radio/lamp
 */
const NarrationMode = ({ story, isActive, onToggle, onNarrationUpdate }) => {
  const [isNarrating, setIsNarrating] = useState(false);
  const [currentWord, setCurrentWord] = useState(0);
  const [speed, setSpeed] = useState(1.0);
  const [voice, setVoice] = useState(null);
  const [voices, setVoices] = useState([]);
  const [isPaused, setIsPaused] = useState(false);
  const speechRef = useRef(null);
  const wordTimerRef = useRef(null);
  
  useEffect(() => {
    // Load available voices
    const loadVoices = () => {
      const availableVoices = window.speechSynthesis.getVoices();
      setVoices(availableVoices.filter(v => v.lang.startsWith("en")));
      
      // Set default voice (prefer female voice for bedtime stories)
      const preferredVoice = availableVoices.find(v => 
        v.name.includes("Female") || v.name.includes("Samantha") || 
        v.name.includes("Victoria") || v.name.includes("Karen")
      ) || availableVoices[0];
      
      setVoice(preferredVoice);
    };
    
    loadVoices();
    window.speechSynthesis.onvoiceschanged = loadVoices;
    
    return () => {
      stopNarration();
    };
  }, []);
  
  const startNarration = () => {
    if (!story || !story.content) return;
    
    // Cancel any ongoing speech
    window.speechSynthesis.cancel();
    
    // Create new utterance
    const utterance = new SpeechSynthesisUtterance(story.content);
    utterance.voice = voice;
    utterance.rate = speed;
    utterance.pitch = 1.0;
    utterance.volume = 1.0;
    
    // Track progress
    const words = story.content.split(" ");
    let wordIndex = 0;
    
    utterance.onboundary = (event) => {
      if (event.name === "word") {
        setCurrentWord(wordIndex++);
        
        // Notify parent component for 3D environment updates
        if (onNarrationUpdate) {
          onNarrationUpdate({
            progress: wordIndex / words.length,
            currentWord: wordIndex,
            totalWords: words.length
          });
        }
      }
    };
    
    utterance.onstart = () => {
      setIsNarrating(true);
      setIsPaused(false);
      
      // Send event to 3D environment
      if (onNarrationUpdate) {
        onNarrationUpdate({ event: "start", source: "lamp" });
      }
    };
    
    utterance.onend = () => {
      setIsNarrating(false);
      setCurrentWord(0);
      
      // Send event to 3D environment
      if (onNarrationUpdate) {
        onNarrationUpdate({ event: "end" });
      }
    };
    
    utterance.onerror = (error) => {
      console.error("Speech synthesis error:", error);
      setIsNarrating(false);
    };
    
    speechRef.current = utterance;
    window.speechSynthesis.speak(utterance);
  };
  
  const pauseNarration = () => {
    if (window.speechSynthesis.speaking && !window.speechSynthesis.paused) {
      window.speechSynthesis.pause();
      setIsPaused(true);
      
      if (onNarrationUpdate) {
        onNarrationUpdate({ event: "pause" });
      }
    }
  };
  
  const resumeNarration = () => {
    if (window.speechSynthesis.paused) {
      window.speechSynthesis.resume();
      setIsPaused(false);
      
      if (onNarrationUpdate) {
        onNarrationUpdate({ event: "resume" });
      }
    }
  };
  
  const stopNarration = () => {
    window.speechSynthesis.cancel();
    setIsNarrating(false);
    setIsPaused(false);
    setCurrentWord(0);
    
    if (onNarrationUpdate) {
      onNarrationUpdate({ event: "stop" });
    }
  };
  
  const handleSpeedChange = (newSpeed) => {
    setSpeed(newSpeed);
    
    // If currently narrating, restart with new speed
    if (isNarrating) {
      stopNarration();
      setTimeout(() => startNarration(), 100);
    }
  };
  
  const handleVoiceChange = (voiceName) => {
    const newVoice = voices.find(v => v.name === voiceName);
    if (newVoice) {
      setVoice(newVoice);
      
      // If currently narrating, restart with new voice
      if (isNarrating) {
        stopNarration();
        setTimeout(() => startNarration(), 100);
      }
    }
  };
  
  if (!isActive) return null;
  
  return (
    <div className="narration-mode">
      <div className="narration-header">
        <h3>🎙️ Parent Narration Mode</h3>
        <button 
          onClick={onToggle} 
          className="close-btn"
          aria-label="Close narration mode"
        >
          ✕
        </button>
      </div>
      
      <div className="narration-controls">
        <div className="playback-controls">
          {!isNarrating ? (
            <button 
              onClick={startNarration} 
              className="play-btn"
              disabled={!story}
            >
              ▶️ Start Narration
            </button>
          ) : (
            <>
              {isPaused ? (
                <button onClick={resumeNarration} className="resume-btn">
                  ▶️ Resume
                </button>
              ) : (
                <button onClick={pauseNarration} className="pause-btn">
                  ⏸️ Pause
                </button>
              )}
              <button onClick={stopNarration} className="stop-btn">
                ⏹️ Stop
              </button>
            </>
          )}
        </div>
        
        <div className="voice-controls">
          <label htmlFor="voice-select">Voice:</label>
          <select 
            id="voice-select"
            value={voice?.name || ""} 
            onChange={(e) => handleVoiceChange(e.target.value)}
            disabled={isNarrating}
          >
            {voices.map((v) => (
              <option key={v.name} value={v.name}>
                {v.name} ({v.lang})
              </option>
            ))}
          </select>
        </div>
        
        <div className="speed-controls">
          <label htmlFor="speed-slider">Speed: {speed.toFixed(1)}x</label>
          <input 
            id="speed-slider"
            type="range" 
            min="0.5" 
            max="2.0" 
            step="0.1" 
            value={speed}
            onChange={(e) => handleSpeedChange(parseFloat(e.target.value))}
            disabled={isNarrating}
          />
        </div>
      </div>
      
      {isNarrating && (
        <div className="narration-progress">
          <div className="progress-bar">
            <div 
              className="progress-fill" 
              style={{ 
                width: `${(currentWord / (story?.content?.split(" ").length || 1)) * 100}%` 
              }}
            />
          </div>
          <p className="progress-text">
            Word {currentWord} of {story?.content?.split(" ").length || 0}
          </p>
        </div>
      )}
      
      <div className="narration-tips">
        <h4>💡 Tips for Great Narration:</h4>
        <ul>
          <li>Choose a calm, soothing voice for bedtime</li>
          <li>Adjust speed to match your child's comprehension</li>
          <li>The lamp in the 3D scene will glow while narrating</li>
          <li>Pause to discuss the story with your child</li>
        </ul>
      </div>
      
      <style jsx>{`
        .narration-mode {
          position: fixed;
          bottom: 20px;
          right: 20px;
          width: 350px;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          border-radius: 12px;
          padding: 20px;
          box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
          z-index: 1000;
          color: white;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        
        .narration-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 20px;
        }
        
        .narration-header h3 {
          margin: 0;
          font-size: 18px;
          font-weight: 600;
        }
        
        .close-btn {
          background: rgba(255, 255, 255, 0.2);
          border: none;
          color: white;
          width: 30px;
          height: 30px;
          border-radius: 50%;
          cursor: pointer;
          font-size: 16px;
          display: flex;
          align-items: center;
          justify-content: center;
          transition: all 0.3s;
        }
        
        .close-btn:hover {
          background: rgba(255, 255, 255, 0.3);
          transform: scale(1.1);
        }
        
        .narration-controls {
          display: flex;
          flex-direction: column;
          gap: 15px;
        }
        
        .playback-controls {
          display: flex;
          gap: 10px;
        }
        
        .play-btn, .pause-btn, .resume-btn, .stop-btn {
          flex: 1;
          padding: 10px;
          border: none;
          border-radius: 8px;
          background: rgba(255, 255, 255, 0.9);
          color: #667eea;
          font-weight: 600;
          cursor: pointer;
          transition: all 0.3s;
        }
        
        .play-btn:hover:not(:disabled), 
        .pause-btn:hover, 
        .resume-btn:hover, 
        .stop-btn:hover {
          background: white;
          transform: translateY(-2px);
          box-shadow: 0 5px 15px rgba(0, 0, 0, 0.2);
        }
        
        .play-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
        
        .voice-controls, .speed-controls {
          display: flex;
          align-items: center;
          gap: 10px;
        }
        
        .voice-controls label, .speed-controls label {
          font-size: 14px;
          font-weight: 500;
          min-width: 80px;
        }
        
        .voice-controls select {
          flex: 1;
          padding: 8px;
          border: none;
          border-radius: 6px;
          background: rgba(255, 255, 255, 0.9);
          color: #333;
          font-size: 14px;
        }
        
        .speed-controls input[type="range"] {
          flex: 1;
          height: 6px;
          border-radius: 3px;
          background: rgba(255, 255, 255, 0.3);
          outline: none;
          -webkit-appearance: none;
        }
        
        .speed-controls input[type="range"]::-webkit-slider-thumb {
          -webkit-appearance: none;
          width: 20px;
          height: 20px;
          border-radius: 50%;
          background: white;
          cursor: pointer;
          box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
        }
        
        .narration-progress {
          margin-top: 15px;
        }
        
        .progress-bar {
          height: 8px;
          background: rgba(255, 255, 255, 0.2);
          border-radius: 4px;
          overflow: hidden;
        }
        
        .progress-fill {
          height: 100%;
          background: linear-gradient(90deg, #ffd89b 0%, #19547b 100%);
          transition: width 0.3s ease;
        }
        
        .progress-text {
          font-size: 12px;
          text-align: center;
          margin-top: 5px;
          opacity: 0.8;
        }
        
        .narration-tips {
          margin-top: 20px;
          padding-top: 15px;
          border-top: 1px solid rgba(255, 255, 255, 0.2);
        }
        
        .narration-tips h4 {
          font-size: 14px;
          margin-bottom: 10px;
        }
        
        .narration-tips ul {
          font-size: 12px;
          margin: 0;
          padding-left: 20px;
          opacity: 0.9;
        }
        
        .narration-tips li {
          margin-bottom: 5px;
        }
      `}</style>
    </div>
  );
};

export default NarrationMode;