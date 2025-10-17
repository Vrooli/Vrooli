import React from "react";
import { motion } from "framer-motion";

export const floorOptions = [
  {
    value: "smooth",
    label: "Smooth Glow",
    description: "Soft gradient floor with no pattern.",
  },
  {
    value: "soft-stripes",
    label: "Lullaby Stripes",
    description: "Wide pastel stripes for a sleepy rhythm.",
  },
  {
    value: "storybook-checker",
    label: "Storybook Checker",
    description: "Playful checkerboard inspired by classic picture books.",
  },
  {
    value: "speckled-sparkle",
    label: "Starry Speckle",
    description: "Tiny sparkles scattered across the floor.",
  },
];

export const wallOptions = [
  {
    value: "solid",
    label: "Soft Wash",
    description: "Gentle wash of colour with no embellishment.",
  },
  {
    value: "storybook-clouds",
    label: "Floating Clouds",
    description: "Hand-painted clouds drifting across the wall.",
  },
  {
    value: "starlight",
    label: "Night Sky",
    description: "Tiny glowing stars across the wallpaper.",
  },
  {
    value: "sunset-gradient",
    label: "Sunset Fade",
    description: "Warm gradient that shifts from sky to dusk.",
  },
];

const overlayVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
  exit: { opacity: 0 },
};

const panelVariants = {
  hidden: { opacity: 0, y: 30, scale: 0.96 },
  visible: { opacity: 1, y: 0, scale: 1 },
  exit: { opacity: 0, y: 20, scale: 0.96 },
};

const SettingsPanel = ({
  floorTexture,
  wallTexture,
  onChangeFloor,
  onChangeWall,
  onClose,
}) => (
  <motion.div
    className="modal-overlay settings"
    initial="hidden"
    animate="visible"
    exit="exit"
    variants={overlayVariants}
  >
    <motion.div
      className="settings-panel"
      variants={panelVariants}
      role="dialog"
      aria-modal="true"
      onClick={(event) => event.stopPropagation()}
    >
      <button
        className="icon-button close"
        aria-label="Close settings"
        onClick={onClose}
      >
        ✕
      </button>
      <header className="panel-header">
        <div>
          <h2>Room Appearance</h2>
          <p>Fine tune textures to match tonight’s mood.</p>
        </div>
      </header>

      <section className="settings-section">
        <h3>Floor Finish</h3>
        <div className="settings-options">
          {floorOptions.map((option) => (
            <button
              key={option.value}
              type="button"
              className={`settings-option ${floorTexture === option.value ? "active" : ""}`}
              onClick={() => onChangeFloor(option.value)}
            >
              <div className="option-chip">{option.label}</div>
              <p>{option.description}</p>
            </button>
          ))}
        </div>
      </section>

      <section className="settings-section">
        <h3>Wall Finish</h3>
        <div className="settings-options">
          {wallOptions.map((option) => (
            <button
              key={option.value}
              type="button"
              className={`settings-option ${wallTexture === option.value ? "active" : ""}`}
              onClick={() => onChangeWall(option.value)}
            >
              <div className="option-chip">{option.label}</div>
              <p>{option.description}</p>
            </button>
          ))}
        </div>
      </section>

      <div className="panel-actions">
        <button className="primary-action" onClick={onClose}>
          Done
        </button>
      </div>
    </motion.div>
  </motion.div>
);

export default SettingsPanel;
