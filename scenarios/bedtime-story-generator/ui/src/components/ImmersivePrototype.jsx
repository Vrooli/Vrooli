import React, { useEffect, useRef, useState } from "react";
import { Experience } from "../three/index.js";
import useExperienceStore from "../state/store.js";
import SceneControls from "./SceneControls.jsx";
import PerformanceDashboard from "./PerformanceDashboard.jsx";

const ImmersivePrototype = ({ onClose }) => {
  const canvasRef = useRef(null);
  const [experience, setExperience] = useState(null);
  const [showPerformance, setShowPerformance] = useState(false);
  const timeOfDay = useExperienceStore((state) => state.timeOfDay);
  const selectedStory = useExperienceStore((state) => state.selectedStory);
  const activeRoom = useExperienceStore((state) => state.activeRoom);
  const availableRooms = useExperienceStore((state) => state.availableRooms);

  const activeRoomLabel =
    availableRooms.find((room) => room.id === activeRoom)?.label || activeRoom;

  useEffect(() => {
    const host = canvasRef.current;
    if (!host) {
      return undefined;
    }

    let experienceInstance;
    try {
      experienceInstance = new Experience({ target: host });
      setExperience(experienceInstance);
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Failed to start immersive prototype", error);
    }

    return () => {
      experienceInstance?.destroy?.();
      setExperience(null);
    };
  }, []);

  return (
    <div className="prototype-overlay" role="dialog" aria-modal="true">
      <div className="prototype-toolbar">
        <div className="prototype-toolbar__info">
          <strong>Immersive Prototype</strong>
          <span>
            {`Scene mood: ${timeOfDay} • Room: ${activeRoomLabel}`}
            {selectedStory
              ? ` • Currently focusing on “${selectedStory.title}”`
              : " • Select a story to see live responses"}
          </span>
        </div>
        <button type="button" className="ghost-action" onClick={onClose}>
          Close
        </button>
      </div>
      <div
        ref={canvasRef}
        className="prototype-canvas"
        aria-label="Immersive prototype canvas"
      />
      {experience && <SceneControls experience={experience} />}
      {experience && (
        <PerformanceDashboard 
          experience={experience} 
          isOpen={showPerformance}
          onToggle={() => setShowPerformance(!showPerformance)}
        />
      )}
    </div>
  );
};

export default ImmersivePrototype;
