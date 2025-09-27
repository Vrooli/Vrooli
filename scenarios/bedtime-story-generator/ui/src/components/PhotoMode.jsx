import React, { useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";

const PANEL_VARIANTS = {
  hidden: { opacity: 0, scale: 0.9, y: 20 },
  visible: { opacity: 1, scale: 1, y: 0 },
  exit: { opacity: 0, scale: 0.9, y: 20 },
};

const RESOLUTIONS = [
  { label: "HD (1920x1080)", width: 1920, height: 1080 },
  { label: "2K (2560x1440)", width: 2560, height: 1440 },
  { label: "4K (3840x2160)", width: 3840, height: 2160 },
  { label: "6K (6144x3456)", width: 6144, height: 3456 },
];

const PRESETS = [
  { 
    id: "hero",
    label: "Hero Shot", 
    description: "Wide angle view of the entire room",
    camera: { position: [3, 2, 5], target: [0, 0, 0] }
  },
  { 
    id: "bookshelf",
    label: "Bookshelf Focus", 
    description: "Close-up of the magical bookshelf",
    camera: { position: [2.5, 0.5, 1], target: [2, 0.5, -1] }
  },
  { 
    id: "window",
    label: "Window View", 
    description: "Atmospheric shot with window lighting",
    camera: { position: [-1, 1.2, 2], target: [-2, 1, -2] }
  },
  { 
    id: "lamp",
    label: "Reading Lamp", 
    description: "Cozy lamp with warm glow",
    camera: { position: [1.5, 1, 0.5], target: [0.5, 0.5, -0.5] }
  },
  { 
    id: "toys",
    label: "Toy Chest", 
    description: "Playful view of animated toys",
    camera: { position: [-0.5, 0.8, 2], target: [-1.5, 0.3, 0] }
  },
];

const PhotoMode = ({ experience, onClose }) => {
  const [selectedResolution, setSelectedResolution] = useState(RESOLUTIONS[2]); // Default 4K
  const [selectedPreset, setSelectedPreset] = useState(PRESETS[0]);
  const [isCapturing, setIsCapturing] = useState(false);
  const [captureProgress, setCaptureProgress] = useState(0);
  const [includeUI, setIncludeUI] = useState(false);
  const [transparentBg, setTransparentBg] = useState(false);

  const captureScreenshot = useCallback(async () => {
    if (!experience?.renderer?.instance || isCapturing) return;

    setIsCapturing(true);
    setCaptureProgress(0);

    try {
      const renderer = experience.renderer.instance;
      const scene = experience.scene;
      const camera = experience.camera.instance;
      
      // Store original settings
      const originalWidth = renderer.domElement.width;
      const originalHeight = renderer.domElement.height;
      const originalPixelRatio = renderer.getPixelRatio();
      const originalBackground = scene.background;
      const originalCameraPosition = camera.position.clone();
      const originalCameraQuaternion = camera.quaternion.clone();

      // Apply preset camera position if selected
      if (selectedPreset) {
        camera.position.set(...selectedPreset.camera.position);
        camera.lookAt(...selectedPreset.camera.target);
      }

      setCaptureProgress(25);

      // Set up high-resolution render
      const renderWidth = selectedResolution.width;
      const renderHeight = selectedResolution.height;
      
      // Create offscreen canvas for high-res capture
      const offscreenCanvas = document.createElement("canvas");
      offscreenCanvas.width = renderWidth;
      offscreenCanvas.height = renderHeight;
      
      // Update renderer size
      renderer.setSize(renderWidth, renderHeight);
      renderer.setPixelRatio(1); // Use 1 for exact resolution
      
      // Apply transparent background if requested
      if (transparentBg) {
        scene.background = null;
        renderer.setClearAlpha(0);
      }

      setCaptureProgress(50);

      // Update camera aspect ratio
      camera.aspect = renderWidth / renderHeight;
      camera.updateProjectionMatrix();

      // Render the scene
      renderer.render(scene, camera);

      setCaptureProgress(75);

      // Get the image data
      const dataURL = renderer.domElement.toDataURL("image/png");
      
      // Create download link
      const link = document.createElement("a");
      const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
      const presetName = selectedPreset ? `-${selectedPreset.id}` : "";
      link.download = `bedtime-story-${timestamp}${presetName}-${renderWidth}x${renderHeight}.png`;
      link.href = dataURL;
      link.click();

      setCaptureProgress(100);

      // Restore original settings
      renderer.setSize(originalWidth, originalHeight);
      renderer.setPixelRatio(originalPixelRatio);
      scene.background = originalBackground;
      if (transparentBg) {
        renderer.setClearAlpha(1);
      }
      camera.position.copy(originalCameraPosition);
      camera.quaternion.copy(originalCameraQuaternion);
      camera.aspect = originalWidth / originalHeight;
      camera.updateProjectionMatrix();
      
      // Re-render with original settings
      renderer.render(scene, camera);

      // Show success message
      setTimeout(() => {
        setCaptureProgress(0);
        setIsCapturing(false);
      }, 1000);
    } catch (error) {
      console.error("Failed to capture screenshot:", error);
      setIsCapturing(false);
      setCaptureProgress(0);
    }
  }, [experience, selectedResolution, selectedPreset, transparentBg]);

  const copyMarketingPrompt = useCallback(() => {
    const prompt = `Create marketing materials for a children's bedtime story app featuring:
- ${selectedPreset.description}
- Resolution: ${selectedResolution.label}
- Scene: Immersive 3D bedroom with interactive elements
- Mood: Cozy, magical, child-friendly
- Key features: AI-generated stories, dynamic time-of-day lighting, interactive bookshelf`;
    
    navigator.clipboard.writeText(prompt);
  }, [selectedPreset, selectedResolution]);

  return (
    <AnimatePresence>
      <motion.div
        className="photo-mode-panel"
        initial="hidden"
        animate="visible"
        exit="exit"
        variants={PANEL_VARIANTS}
      >
        <div className="photo-mode-header">
          <h3>📸 Photo Mode</h3>
          <button
            className="close-button"
            onClick={onClose}
            aria-label="Close photo mode"
          >
            ×
          </button>
        </div>

        <div className="photo-mode-content">
          {/* Resolution Selection */}
          <div className="photo-mode-section">
            <h4>Resolution</h4>
            <div className="resolution-grid">
              {RESOLUTIONS.map((res) => (
                <button
                  key={res.label}
                  className={`resolution-option ${selectedResolution === res ? "active" : ""}`}
                  onClick={() => setSelectedResolution(res)}
                >
                  {res.label}
                </button>
              ))}
            </div>
          </div>

          {/* Preset Selection */}
          <div className="photo-mode-section">
            <h4>Camera Presets</h4>
            <div className="preset-grid">
              {PRESETS.map((preset) => (
                <button
                  key={preset.id}
                  className={`preset-option ${selectedPreset === preset ? "active" : ""}`}
                  onClick={() => setSelectedPreset(preset)}
                  title={preset.description}
                >
                  {preset.label}
                </button>
              ))}
            </div>
            {selectedPreset && (
              <p className="preset-description">{selectedPreset.description}</p>
            )}
          </div>

          {/* Options */}
          <div className="photo-mode-section">
            <h4>Options</h4>
            <label className="photo-mode-checkbox">
              <input
                type="checkbox"
                checked={transparentBg}
                onChange={(e) => setTransparentBg(e.target.checked)}
              />
              <span>Transparent Background</span>
            </label>
            <label className="photo-mode-checkbox">
              <input
                type="checkbox"
                checked={includeUI}
                onChange={(e) => setIncludeUI(e.target.checked)}
                disabled
              />
              <span>Include UI Elements (Coming Soon)</span>
            </label>
          </div>

          {/* Capture Progress */}
          {isCapturing && (
            <div className="capture-progress">
              <div className="progress-bar">
                <div 
                  className="progress-fill" 
                  style={{ width: `${captureProgress}%` }}
                />
              </div>
              <p>Capturing {selectedResolution.label} image...</p>
            </div>
          )}

          {/* Actions */}
          <div className="photo-mode-actions">
            <button
              className="capture-button"
              onClick={captureScreenshot}
              disabled={isCapturing}
            >
              {isCapturing ? "Capturing..." : "📸 Capture Screenshot"}
            </button>
            <button
              className="secondary-button"
              onClick={copyMarketingPrompt}
            >
              📋 Copy Marketing Prompt
            </button>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
};

export default PhotoMode;