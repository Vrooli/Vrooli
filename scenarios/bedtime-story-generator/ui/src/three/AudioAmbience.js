import * as THREE from "three";

export default class AudioAmbience {
  constructor({ scene, camera }) {
    this.scene = scene;
    this.camera = camera;
    this.listener = null;
    this.sounds = {};
    this.enabled = true;
    this.masterVolume = 0.7;
    
    // Initialize audio context and listener
    this._initializeAudio();
    
    // Time of day ambience configurations
    this.ambienceConfig = {
      day: {
        birds: { volume: 0.3, fadeIn: 2 },
        wind: { volume: 0.1, fadeIn: 3 },
        children: { volume: 0.15, fadeIn: 4 }
      },
      evening: {
        crickets: { volume: 0.25, fadeIn: 2 },
        wind: { volume: 0.15, fadeIn: 3 },
        birds: { volume: 0.1, fadeIn: 3 }
      },
      night: {
        crickets: { volume: 0.4, fadeIn: 2 },
        owls: { volume: 0.2, fadeIn: 5 },
        wind: { volume: 0.05, fadeIn: 3 }
      }
    };
    
    // Story mood configurations
    this.storyMoodConfig = {
      adventure: {
        drums: { volume: 0.2, fadeIn: 1 },
        heroic: { volume: 0.3, fadeIn: 2 }
      },
      magic: {
        chimes: { volume: 0.3, fadeIn: 1.5 },
        mystical: { volume: 0.25, fadeIn: 2 }
      },
      ocean: {
        waves: { volume: 0.4, fadeIn: 2 },
        seagulls: { volume: 0.2, fadeIn: 3 }
      },
      space: {
        synth: { volume: 0.3, fadeIn: 2 },
        cosmic: { volume: 0.25, fadeIn: 3 }
      },
      forest: {
        leaves: { volume: 0.3, fadeIn: 2 },
        birds: { volume: 0.25, fadeIn: 2 }
      }
    };
    
    // Current states
    this.currentTimeOfDay = "day";
    this.currentStoryMood = null;
    this.activeAmbience = new Set();
    
    // Create mock sound sources (in production, these would be actual audio files)
    this._createSoundSources();
  }
  
  _initializeAudio() {
    // Create audio listener and attach to camera
    this.listener = new THREE.AudioListener();
    this.camera.add(this.listener);
    
    // Check if Web Audio API is available
    const AudioContext = window.AudioContext || window.webkitAudioContext;
    if (!AudioContext) {
      console.warn("Web Audio API not supported");
      this.enabled = false;
      return;
    }
  }
  
  _createSoundSources() {
    // Create oscillator-based ambient sounds as placeholders
    // In production, these would load actual audio files
    
    const soundTypes = [
      "birds", "wind", "children", "crickets", "owls",
      "drums", "heroic", "chimes", "mystical", "waves",
      "seagulls", "synth", "cosmic", "leaves"
    ];
    
    soundTypes.forEach(type => {
      this.sounds[type] = this._createMockSound(type);
    });
  }
  
  _createMockSound(type) {
    // Create a positional audio source
    const sound = new THREE.PositionalAudio(this.listener);
    
    // Create oscillator for mock ambient sound
    const oscillator = this.listener.context.createOscillator();
    const gainNode = this.listener.context.createGain();
    
    // Configure based on sound type
    const frequencies = {
      birds: 800,
      wind: 100,
      children: 400,
      crickets: 4000,
      owls: 250,
      drums: 60,
      heroic: 300,
      chimes: 2000,
      mystical: 600,
      waves: 80,
      seagulls: 1200,
      synth: 440,
      cosmic: 220,
      leaves: 150
    };
    
    oscillator.frequency.value = frequencies[type] || 440;
    oscillator.type = type.includes("wind") || type.includes("waves") ? "triangle" : "sine";
    
    // Add some randomness for natural sound
    if (type === "crickets" || type === "birds") {
      const lfo = this.listener.context.createOscillator();
      const lfoGain = this.listener.context.createGain();
      lfo.frequency.value = Math.random() * 2 + 0.5;
      lfoGain.gain.value = 50;
      lfo.connect(lfoGain);
      lfoGain.connect(oscillator.frequency);
      lfo.start();
    }
    
    // Initial volume is 0
    gainNode.gain.value = 0;
    
    // Connect nodes
    oscillator.connect(gainNode);
    
    // Store references
    sound.userData = {
      oscillator,
      gainNode,
      type,
      targetVolume: 0,
      currentVolume: 0,
      fadeSpeed: 0.01
    };
    
    // Position sound in 3D space
    this._positionSound(sound, type);
    
    return sound;
  }
  
  _positionSound(sound, type) {
    // Position sounds around the scene for spatial audio
    const positions = {
      birds: new THREE.Vector3(5, 5, -5),
      wind: new THREE.Vector3(0, 8, 0),
      children: new THREE.Vector3(-5, 2, 5),
      crickets: new THREE.Vector3(-3, 0, -3),
      owls: new THREE.Vector3(4, 6, -4),
      drums: new THREE.Vector3(0, 1, 0),
      heroic: new THREE.Vector3(0, 3, 0),
      chimes: new THREE.Vector3(2, 4, 2),
      mystical: new THREE.Vector3(-2, 3, -2),
      waves: new THREE.Vector3(0, 1, -8),
      seagulls: new THREE.Vector3(3, 6, -6),
      synth: new THREE.Vector3(0, 2, 0),
      cosmic: new THREE.Vector3(0, 5, 0),
      leaves: new THREE.Vector3(-4, 2, 4)
    };
    
    const position = positions[type] || new THREE.Vector3(0, 2, 0);
    sound.position.copy(position);
    
    // Set distance model
    sound.setRefDistance(5);
    sound.setMaxDistance(20);
    sound.setRolloffFactor(1);
    
    // Add to scene
    this.scene.add(sound);
  }
  
  setTimeOfDay(timeOfDay) {
    if (this.currentTimeOfDay === timeOfDay || !this.enabled) return;
    
    this.currentTimeOfDay = timeOfDay;
    const config = this.ambienceConfig[timeOfDay];
    
    if (!config) return;
    
    // Fade out sounds not in new config
    this.activeAmbience.forEach(soundType => {
      if (!config[soundType]) {
        this._fadeOutSound(soundType);
        this.activeAmbience.delete(soundType);
      }
    });
    
    // Fade in new sounds
    Object.entries(config).forEach(([soundType, settings]) => {
      this._fadeInSound(soundType, settings.volume, settings.fadeIn);
      this.activeAmbience.add(soundType);
    });
  }
  
  setStoryMood(mood) {
    if (this.currentStoryMood === mood || !this.enabled) return;
    
    // Fade out previous story mood
    if (this.currentStoryMood) {
      const prevConfig = this.storyMoodConfig[this.currentStoryMood];
      if (prevConfig) {
        Object.keys(prevConfig).forEach(soundType => {
          this._fadeOutSound(soundType);
        });
      }
    }
    
    this.currentStoryMood = mood;
    const config = this.storyMoodConfig[mood];
    
    if (!config) return;
    
    // Fade in new story mood
    Object.entries(config).forEach(([soundType, settings]) => {
      this._fadeInSound(soundType, settings.volume * 0.5, settings.fadeIn);
    });
  }
  
  _fadeInSound(soundType, targetVolume, duration = 2) {
    const sound = this.sounds[soundType];
    if (!sound || !sound.userData) return;
    
    const userData = sound.userData;
    userData.targetVolume = targetVolume * this.masterVolume;
    userData.fadeSpeed = 1 / (duration * 60); // Assuming 60 FPS
    
    // Start oscillator if not started
    if (!userData.isPlaying) {
      userData.oscillator.start();
      userData.isPlaying = true;
    }
  }
  
  _fadeOutSound(soundType, duration = 1) {
    const sound = this.sounds[soundType];
    if (!sound || !sound.userData) return;
    
    const userData = sound.userData;
    userData.targetVolume = 0;
    userData.fadeSpeed = 1 / (duration * 60);
  }
  
  playOneShot(soundType, volume = 0.5) {
    if (!this.enabled) return;
    
    // Create a one-shot sound effect
    const buffer = this.listener.context.createBuffer(1, 44100, 44100);
    const data = buffer.getChannelData(0);
    
    // Generate simple sound effect
    for (let i = 0; i < 44100; i++) {
      data[i] = Math.sin(i * 0.1) * Math.exp(-i * 0.0001) * volume * this.masterVolume;
    }
    
    const sound = new THREE.Audio(this.listener);
    sound.setBuffer(buffer);
    sound.setVolume(volume * this.masterVolume);
    sound.play();
  }
  
  setMasterVolume(volume) {
    this.masterVolume = Math.max(0, Math.min(1, volume));
    
    // Update all active sounds
    Object.values(this.sounds).forEach(sound => {
      if (sound.userData) {
        const currentRatio = sound.userData.targetVolume / (this.masterVolume || 1);
        sound.userData.targetVolume = currentRatio * this.masterVolume;
      }
    });
  }
  
  toggleEnabled() {
    this.enabled = !this.enabled;
    
    if (!this.enabled) {
      // Fade out all sounds
      Object.keys(this.sounds).forEach(soundType => {
        this._fadeOutSound(soundType, 0.5);
      });
    } else {
      // Restore time of day ambience
      this.setTimeOfDay(this.currentTimeOfDay);
      if (this.currentStoryMood) {
        this.setStoryMood(this.currentStoryMood);
      }
    }
    
    return this.enabled;
  }
  
  update(delta) {
    if (!this.enabled) return;
    
    // Update volume fades
    Object.values(this.sounds).forEach(sound => {
      if (!sound.userData) return;
      
      const userData = sound.userData;
      const volumeDiff = userData.targetVolume - userData.currentVolume;
      
      if (Math.abs(volumeDiff) > 0.001) {
        const fadeAmount = userData.fadeSpeed * delta * 60;
        if (volumeDiff > 0) {
          userData.currentVolume = Math.min(
            userData.currentVolume + fadeAmount,
            userData.targetVolume
          );
        } else {
          userData.currentVolume = Math.max(
            userData.currentVolume - fadeAmount,
            userData.targetVolume
          );
        }
        
        userData.gainNode.gain.value = userData.currentVolume;
      }
      
      // Stop oscillator if volume is 0
      if (userData.currentVolume === 0 && userData.isPlaying) {
        // In production, we would actually stop the sound
        // For mock sounds, we just set gain to 0
        userData.gainNode.gain.value = 0;
      }
    });
  }
  
  destroy() {
    // Clean up all sounds
    Object.values(this.sounds).forEach(sound => {
      if (sound.userData && sound.userData.isPlaying) {
        sound.userData.oscillator.stop();
      }
      if (sound.parent) {
        sound.parent.remove(sound);
      }
    });
    
    // Remove listener from camera
    if (this.camera && this.listener) {
      this.camera.remove(this.listener);
    }
    
    this.sounds = {};
    this.activeAmbience.clear();
  }

  setEnabled(enabled) {
    if (this.enabled === enabled) {
      return;
    }

    this.enabled = enabled;

    if (!enabled) {
      Object.keys(this.sounds).forEach((soundType) => {
        this._fadeOutSound(soundType, 0.5);
      });
    } else {
      this.setTimeOfDay(this.currentTimeOfDay || "day");
      if (this.currentStoryMood) {
        this.setStoryMood(this.currentStoryMood);
      }
    }
  }
}
