import React from 'react';
import { motion } from 'framer-motion';

const OVERLAY_ITEMS = [
  { id: 'bed', label: 'Bed & Bedding', description: 'Focus on the bed frame, duvet, and pillows.' },
  { id: 'bookshelf', label: 'Bookshelf', description: 'Inspect spine colors, shelves, and halo.' },
  { id: 'nightstand', label: 'Nightstand & Lamp', description: 'Lamp glow, nightstand body, and controls.' },
  { id: 'window', label: 'Window & Wall', description: 'Frame, wallpaper, and window view.' },
  { id: 'toys', label: 'Soft Toys', description: 'Floating teddy and play objects.' },
  { id: 'mobile', label: 'Hanging Mobile', description: 'Orb cluster above the bed.' },
  { id: 'arch', label: 'Decorative Arch', description: 'Curved play arch near the front.' }
];

const overlayVariants = {
  hidden: { opacity: 0, x: 40 },
  visible: { opacity: 1, x: 0 },
  exit: { opacity: 0, x: 20 }
};

const SceneDebugPanel = ({ selectedId, onSelect, onClose }) => (
  <motion.div
    className="debug-panel"
    initial="hidden"
    animate="visible"
    exit="exit"
    variants={overlayVariants}
  >
    <header className="debug-header">
      <div>
        <h3>Scene Debugger</h3>
        <p>Locate items, toggle outlines, and copy screenshot cues.</p>
      </div>
      <button className="icon-button" onClick={onClose} aria-label="Close scene debugger">
        ✕
      </button>
    </header>

    <div className="debug-list">
      {OVERLAY_ITEMS.map((item) => (
        <button
          key={item.id}
          className={`debug-item ${selectedId === item.id ? 'active' : ''}`}
          onClick={() => onSelect(selectedId === item.id ? null : item.id)}
        >
          <span className="debug-label">{item.label}</span>
          <p>{item.description}</p>
        </button>
      ))}
    </div>

    <div className="debug-actions">
      <button
        className="ghost-action"
        onClick={async () => {
          try {
            await navigator.clipboard?.writeText('Screenshot tip: focus centered on the selected item with UI hidden.');
          } catch (error) {
            console.warn('Clipboard unsupported', error);
          }
        }}
      >
        Copy Screenshot Tip
      </button>
    </div>
  </motion.div>
);

export default SceneDebugPanel;
