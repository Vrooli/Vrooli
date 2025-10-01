import React from "react";
import useExperienceStore from "../state/store.js";

const TOGGLES = [
  {
    flag: "disableAudio",
    label: "Disable Ambient Audio",
    description: "Turns off oscillator-based ambience and fade processing.",
  },
  {
    flag: "disableWorldAnimations",
    label: "Freeze Scene Animations",
    description: "Skips particle, lighting, and prop animations inside the world update loop.",
  },
  {
    flag: "disableProjectorCanvas",
    label: "Stop Projector Canvas Updates",
    description: "Halts story text rendering on the projector screen.",
  },
  {
    flag: "disablePostProcessing",
    label: "Disable Post Processing",
    description: "Bypasses bloom and color grading passes (if developer mode enabled).",
  },
];

const FeatureToggleDialog = ({ open, onClose }) => {
  const profiling = useExperienceStore((state) => state.profiling);
  const setProfilingFlag = useExperienceStore((state) => state.setProfilingFlag);

  if (!open) {
    return null;
  }

  return (
    <div className="feature-toggle-overlay" role="dialog" aria-modal="true">
      <div className="feature-toggle-panel">
        <header className="feature-toggle-header">
          <div>
            <h3>Performance Toggles</h3>
            <p>Disable subsystems to understand their impact. Changes apply instantly.</p>
          </div>
          <button type="button" className="ghost-action" onClick={onClose}>
            Close
          </button>
        </header>

        <ul className="feature-toggle-list">
          {TOGGLES.map((item) => (
            <li key={item.flag}>
              <label className="feature-toggle-item">
                <input
                  type="checkbox"
                  checked={!!profiling?.[item.flag]}
                  onChange={(event) => setProfilingFlag(item.flag, event.target.checked)}
                />
                <div>
                  <strong>{item.label}</strong>
                  <p>{item.description}</p>
                </div>
              </label>
            </li>
          ))}
        </ul>
      </div>

      <style>{`
        .feature-toggle-overlay {
          position: fixed;
          inset: 0;
          background: rgba(10, 12, 25, 0.65);
          backdrop-filter: blur(6px);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 1200;
          padding: 24px;
        }
        .feature-toggle-panel {
          width: min(420px, 100%);
          background: rgba(20, 24, 44, 0.95);
          border-radius: 14px;
          padding: 20px;
          box-shadow: 0 18px 40px rgba(0, 0, 0, 0.35);
          color: #f8faff;
          display: flex;
          flex-direction: column;
          gap: 16px;
        }
        .feature-toggle-header {
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          gap: 12px;
        }
        .feature-toggle-header h3 {
          margin: 0 0 4px;
          font-size: 18px;
        }
        .feature-toggle-header p {
          margin: 0;
          font-size: 13px;
          color: rgba(255, 255, 255, 0.7);
        }
        .feature-toggle-list {
          list-style: none;
          margin: 0;
          padding: 0;
          display: flex;
          flex-direction: column;
          gap: 12px;
        }
        .feature-toggle-item {
          display: flex;
          gap: 12px;
          align-items: flex-start;
          background: rgba(255, 255, 255, 0.05);
          border-radius: 10px;
          padding: 12px;
        }
        .feature-toggle-item input {
          margin-top: 4px;
        }
        .feature-toggle-item strong {
          display: block;
          font-size: 14px;
          margin-bottom: 4px;
        }
        .feature-toggle-item p {
          margin: 0;
          font-size: 12px;
          color: rgba(255, 255, 255, 0.65);
        }
      `}</style>
    </div>
  );
};

export default FeatureToggleDialog;
