import React, { useEffect, useMemo, useState } from "react";
import axios from "axios";
import { AnimatePresence, motion } from "framer-motion";

import ChildrensRoom from "./components/ChildrensRoom";
import Bookshelf from "./components/Bookshelf";
import BookReader from "./components/BookReader";
import StoryGenerator from "./components/StoryGenerator";
import ParentDashboard from "./ParentDashboard";
import ImmersivePrototype from "./components/ImmersivePrototype";
import SceneDebugPanel from "./components/SceneDebugPanel";
import SettingsPanel, {
  floorOptions,
  wallOptions,
} from "./components/SettingsPanel";
import { DEBUG_OVERLAY_ITEMS } from "./components/SceneDebugPanel";
import useExperienceStore from "./state/store";
import { getGreeting } from "./utils/timeUtils";

const DEFAULT_API_PORT = import.meta.env.VITE_API_PORT || "16902";
const API_URL =
  process.env.NODE_ENV === "production"
    ? "/api"
    : `http://localhost:${DEFAULT_API_PORT}/api`;

const HUD_VARIANTS = {
  hidden: { opacity: 0, y: 30 },
  visible: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: 18 },
};

const PANEL_VARIANTS = {
  hidden: { opacity: 0, y: 24 },
  visible: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: 12 },
};

const MOOD_COPY = {
  day: "Sunlit energy for adventurous tales.",
  evening: "Warm twilight to wind down gently.",
  night: "Starlit calm for whisper-soft endings.",
};

const CAMERA_DEBUG_FOCUSES = new Set(["bed", "bookshelf", "window", "mobile"]);

function App() {
  const timeOfDay = useExperienceStore((state) => state.timeOfDay);
  const theme = useExperienceStore((state) => state.theme);
  const stories = useExperienceStore((state) => state.stories);
  const selectedStory = useExperienceStore((state) => state.selectedStory);
  const isReading = useExperienceStore((state) => state.isReading);
  const generatorOpen = useExperienceStore((state) => state.generatorOpen);
  const parentDashboardOpen = useExperienceStore(
    (state) => state.parentDashboardOpen,
  );
  const prototypeOpen = useExperienceStore((state) => state.prototypeOpen);
  const developerMode = useExperienceStore((state) => state.developerMode);
  const developerConsoleOpen = useExperienceStore(
    (state) => state.developerConsoleOpen,
  );
  const loader = useExperienceStore((state) => state.loader);
  const hotspots = useExperienceStore((state) => state.hotspots);
  const cameraFocus = useExperienceStore((state) => state.cameraFocus);
  const cameraAutopilot = useExperienceStore((state) => state.cameraAutopilot);
  const activeRoom = useExperienceStore((state) => state.activeRoom);
  const availableRooms = useExperienceStore((state) => state.availableRooms);

  const setStories = useExperienceStore((state) => state.setStories);
  const prependStory = useExperienceStore((state) => state.prependStory);
  const openStory = useExperienceStore((state) => state.openStory);
  const closeStory = useExperienceStore((state) => state.closeStory);
  const clearSelectedStory = useExperienceStore(
    (state) => state.clearSelectedStory,
  );
  const setGeneratorOpen = useExperienceStore(
    (state) => state.setGeneratorOpen,
  );
  const setParentDashboardOpen = useExperienceStore(
    (state) => state.setParentDashboardOpen,
  );
  const setPrototypeOpen = useExperienceStore(
    (state) => state.setPrototypeOpen,
  );
  const setDeveloperMode = useExperienceStore(
    (state) => state.setDeveloperMode,
  );
  const setDeveloperConsoleOpen = useExperienceStore(
    (state) => state.setDeveloperConsoleOpen,
  );
  const setCameraFocus = useExperienceStore((state) => state.setCameraFocus);
  const setCameraAutopilot = useExperienceStore(
    (state) => state.setCameraAutopilot,
  );
  const setActiveRoom = useExperienceStore((state) => state.setActiveRoom);
  const registerHotspots = useExperienceStore(
    (state) => state.registerHotspots,
  );

  const [loading, setLoading] = useState(true);
  const [libraryOpen, setLibraryOpen] = useState(true);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [debugOpen, setDebugOpen] = useState(false);
  const [debugSelection, setDebugSelection] = useState(null);
  const [floorTexture, setFloorTexture] = useState("smooth");
  const [wallTexture, setWallTexture] = useState("solid");
  const [prefersDomPanels, setPrefersDomPanels] = useState(() => {
    if (typeof window === "undefined") return false;
    const coarse = window.matchMedia?.("(pointer: coarse)").matches;
    const reducedMotion = window.matchMedia?.(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    return coarse || reducedMotion;
  });

  useEffect(() => {
    let isMounted = true;

    const fetchStories = async () => {
      try {
        const response = await axios.get(`${API_URL}/v1/stories`);
        if (isMounted) {
          setStories(response.data || []);
        }
      } catch (error) {
        console.error("Failed to fetch stories:", error);
        if (isMounted) {
          setStories([]);
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    fetchStories();

    return () => {
      isMounted = false;
    };
  }, [setStories]);

  useEffect(() => {
    if (!developerMode) {
      setDeveloperConsoleOpen(false);
      setDebugOpen(false);
      setDebugSelection(null);
    }
  }, [developerMode, setDeveloperConsoleOpen]);

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) {
      return undefined;
    }

    const coarseQuery = window.matchMedia("(pointer: coarse)");
    const motionQuery = window.matchMedia("(prefers-reduced-motion: reduce)");

    const updatePanelPreference = () => {
      setPrefersDomPanels(coarseQuery.matches || motionQuery.matches);
    };

    updatePanelPreference();

    coarseQuery.addEventListener("change", updatePanelPreference);
    motionQuery.addEventListener("change", updatePanelPreference);

    return () => {
      coarseQuery.removeEventListener("change", updatePanelPreference);
      motionQuery.removeEventListener("change", updatePanelPreference);
    };
  }, []);

  useEffect(() => {
    if (!cameraAutopilot) {
      return;
    }

    if (debugSelection && CAMERA_DEBUG_FOCUSES.has(debugSelection)) {
      setCameraFocus(debugSelection);
      return;
    }

    if (isReading) {
      setCameraFocus("bed");
    } else if (libraryOpen) {
      setCameraFocus("bookshelf");
    } else {
      setCameraFocus("wide");
    }
  }, [
    cameraAutopilot,
    debugSelection,
    developerMode,
    developerConsoleOpen,
    isReading,
    libraryOpen,
    activeRoom,
    setCameraFocus,
  ]);

  useEffect(() => {
    const roomEntries = {
      studio: [
        { id: "wide", label: "Cinematic Wide" },
        { id: "bookshelf", label: "Bookshelf" },
        { id: "bed", label: "Bed & Reader" },
        { id: "window", label: "Window" },
        { id: "mobile", label: "Ceiling Mobile" },
      ],
      bedroom: [
        { id: "bed", label: "Bedtime Nook" },
        { id: "wide", label: "Dreamy Wide" },
        { id: "bookshelf", label: "Story Shelves" },
        { id: "window", label: "Stargazing Window" },
        { id: "mobile", label: "Ceiling Mobile" },
      ],
    };

    Object.entries(roomEntries).forEach(([roomId, entries]) => {
      registerHotspots(entries, { roomId });
    });
  }, [registerHotspots]);

  const loaderDescription = useMemo(() => {
    if (loader.status === "idle") return "Immersive scene idle";
    if (loader.status === "complete") return "Immersive scene ready";
    if (loader.total && loader.total > 0) {
      const percent = Math.round((loader.progress || 0) * 100);
      return `${percent}% • ${loader.label || "Loading assets"}`;
    }
    return loader.label || "Loading assets";
  }, [loader]);

  const storiesCount = stories?.length || 0;
  const favoritesCount = useMemo(
    () => stories?.filter((story) => story.is_favorite)?.length || 0,
    [stories],
  );
  const totalReadingMinutes = useMemo(
    () =>
      stories?.reduce(
        (minutes, story) => minutes + (story.reading_time_minutes || 0),
        0,
      ) || 0,
    [stories],
  );
  const activeStoryTitle =
    selectedStory?.title || stories?.[0]?.title || "Select a story to begin";
  const currentFocusLabel =
    hotspots?.[cameraFocus]?.label || "Cinematic Wide Orbit";
  const loaderPercent =
    loader.total && loader.total > 0
      ? Math.round((loader.progress || 0) * 100)
      : null;
  const activeRoomLabel =
    availableRooms.find((room) => room.id === activeRoom)?.label || activeRoom;

  const nextRoomOption = availableRooms.find((room) => room.id !== activeRoom);

  const toggleDeveloperMode = () => {
    if (developerMode) {
      setDeveloperConsoleOpen(false);
      setDebugOpen(false);
      setDebugSelection(null);
      setDeveloperMode(false);
    } else {
      setDeveloperMode(true);
    }
  };

  const handleSelectStory = async (story) => {
    setLibraryOpen(false);
    setDebugSelection(null);
    setCameraAutopilot(true);
    openStory(story);
    try {
      await axios.post(`${API_URL}/v1/stories/${story.id}/read`);
    } catch (error) {
      console.error("Failed to track reading:", error);
    }
  };

  const handleCloseBook = () => {
    closeStory();
    window.setTimeout(() => {
      clearSelectedStory();
    }, 400);
  };

  const handleGenerateStory = async (params) => {
    try {
      const response = await axios.post(
        `${API_URL}/v1/stories/generate`,
        params,
      );
      const newStory = response.data;
      prependStory(newStory);
      setLibraryOpen(false);
      setDebugSelection(null);
      setCameraAutopilot(true);
      openStory(newStory);
      setGeneratorOpen(false);
      return newStory;
    } catch (error) {
      console.error("Failed to generate story:", error);
      throw error;
    }
  };

  const handleToggleFavorite = async (storyId) => {
    try {
      await axios.post(`${API_URL}/v1/stories/${storyId}/favorite`);
      useExperienceStore.setState((state) => ({
        stories: state.stories.map((story) =>
          story.id === storyId
            ? { ...story, is_favorite: !story.is_favorite }
            : story,
        ),
      }));
    } catch (error) {
      console.error("Failed to toggle favorite:", error);
    }
  };

  const handleDeleteStory = async (storyId) => {
    try {
      await axios.delete(`${API_URL}/v1/stories/${storyId}`);
      useExperienceStore.setState((state) => ({
        stories: state.stories.filter((story) => story.id !== storyId),
      }));
      if (selectedStory?.id === storyId) {
        closeStory();
        clearSelectedStory();
      }
    } catch (error) {
      console.error("Failed to remove story:", error);
    }
  };

  const handleDebugSelect = (id) => {
    setDebugSelection((previous) => (previous === id ? null : id));
    if (id) {
      setCameraAutopilot(true);
    }
  };

  const openLibraryPanel = () => {
    setLibraryOpen(true);
    setDebugSelection(null);
    setCameraAutopilot(true);
  };

  const hideLibraryPanel = () => {
    setLibraryOpen(false);
  };

  return (
    <div className={`app ${theme?.class ?? ""}`}>
      <ChildrensRoom
        timeOfDay={timeOfDay}
        theme={theme}
        libraryOpen={libraryOpen}
        onLibraryActivate={openLibraryPanel}
        floorTextureType={floorTexture}
        wallTextureType={wallTexture}
        debugSelection={debugSelection}
        settingsOpen={!prefersDomPanels && settingsOpen}
        onCloseSettings={() => setSettingsOpen(false)}
        onChangeFloor={setFloorTexture}
        onChangeWall={setWallTexture}
        developerMode={developerMode}
        developerConsoleOpen={
          !prefersDomPanels && developerMode && developerConsoleOpen
        }
        hotspots={hotspots}
        cameraFocus={cameraFocus}
        onSelectHotspot={setCameraFocus}
        loaderDescription={loaderDescription}
        selectedStory={selectedStory}
        debugOpen={!prefersDomPanels && debugOpen}
        onSelectDebug={handleDebugSelect}
        onCloseDebug={() => {
          setDebugOpen(false);
          setDebugSelection(null);
        }}
        debugItems={DEBUG_OVERLAY_ITEMS}
        floorOptions={floorOptions}
        wallOptions={wallOptions}
        loaderPercent={loaderPercent}
        cameraAutopilot={cameraAutopilot}
        setCameraAutopilot={setCameraAutopilot}
      >
        <>
          <button
            type="button"
            aria-label="Room appearance settings"
            className="settings-toggle"
            onClick={() => setSettingsOpen(true)}
          >
            🎨
          </button>

          {developerMode && (
            <button
              type="button"
              aria-label="Open scene debugger"
              className="debug-toggle"
              onClick={() => setDebugOpen((open) => !open)}
            >
              🛰️
            </button>
          )}

          <motion.div
            className="hud"
            variants={HUD_VARIANTS}
            initial="hidden"
            animate="visible"
          >
            <div className="hud-top">
              <article className="hud-card">
                <div className="hud-card-heading">
                  <div>
                    <h1 className="hud-greeting">{getGreeting()}</h1>
                    <p className="hud-subtext">
                      {MOOD_COPY[timeOfDay] ||
                        "Settle in and let the storyteller take over."}
                    </p>
                  </div>
                </div>

                <div className="hud-stats">
                  <div className="hud-stat">
                    <span className="stat-label">Stories Ready</span>
                    <span className="stat-value">{storiesCount}</span>
                  </div>
                  <div className="hud-stat">
                    <span className="stat-label">Favourites</span>
                    <span className="stat-value">{favoritesCount}</span>
                  </div>
                  <div className="hud-stat">
                    <span className="stat-label">Bedtime Minutes</span>
                    <span className="stat-value">{totalReadingMinutes}</span>
                  </div>
                  <div className="hud-stat">
                    <span className="stat-label">Camera Focus</span>
                    <span className="stat-value">{currentFocusLabel}</span>
                  </div>
                </div>

                <div style={{ display: "grid", gap: "8px" }}>
                  <span className="stat-label">Immersive Scene</span>
                  {loaderPercent !== null ? (
                    <div className="progress-track">
                      <div
                        className="progress-thumb"
                        style={{ width: `${loaderPercent}%` }}
                      />
                    </div>
                  ) : (
                    loader.status === "loading" && (
                      <div
                        style={{
                          display: "flex",
                          alignItems: "center",
                          gap: 12,
                        }}
                      >
                        <div
                          className="spinner"
                          style={{ width: 24, height: 24 }}
                        />
                        <span className="hud-subtext">Loading...</span>
                      </div>
                    )
                  )}
                  <span className="hud-subtext">{loaderDescription}</span>
                </div>
              </article>

              <article className="hud-card">
                <div className="hud-card-heading">
                  <div>
                    <h2 className="hud-greeting">Tonight&apos;s Toolkit</h2>
                    <p className="hud-subtext">
                      We&apos;re queued on “{activeStoryTitle}”. Spin something
                      new or revisit a favourite.
                    </p>
                  </div>
                </div>

                <div
                  style={{
                    display: "flex",
                    flexWrap: "wrap",
                    gap: "12px",
                  }}
                >
                  <button
                    type="button"
                    className="primary-action"
                    onClick={() => setGeneratorOpen(true)}
                  >
                    ✨ Generate Story
                  </button>
                  <button
                    type="button"
                    className="secondary-action"
                    onClick={() => setParentDashboardOpen(true)}
                  >
                    👨‍👩‍👧 Parent Dashboard
                  </button>
                  <button
                    type="button"
                    className="ghost-action"
                    onClick={() => setPrototypeOpen(true)}
                  >
                    🌌 Immersive Prototype
                  </button>
                </div>

                <div
                  style={{
                    display: "flex",
                    flexWrap: "wrap",
                    gap: "12px",
                    marginTop: "12px",
                  }}
                >
                  <button
                    type="button"
                    className="ghost-action"
                    onClick={libraryOpen ? hideLibraryPanel : openLibraryPanel}
                  >
                    {libraryOpen ? "Hide Library" : "Open Library"}
                  </button>
                  <button
                    type="button"
                    className="ghost-action"
                    onClick={toggleDeveloperMode}
                  >
                    {developerMode
                      ? "🛠️ Developer Mode: On"
                      : "🛠️ Developer Mode"}
                  </button>
                  {developerMode && (
                    <button
                      type="button"
                      className="ghost-action"
                      onClick={() =>
                        setDeveloperConsoleOpen(!developerConsoleOpen)
                      }
                    >
                      {developerConsoleOpen ? "Close Console" : "Open Console"}
                    </button>
                  )}
                  <button
                    type="button"
                    className="ghost-action"
                    onClick={() => setCameraAutopilot(!cameraAutopilot)}
                  >
                    {cameraAutopilot ? "Unlock Camera" : "Lock Camera"}
                  </button>
                  {nextRoomOption && (
                    <button
                      type="button"
                      className="ghost-action"
                      onClick={() => setActiveRoom(nextRoomOption.id)}
                    >
                      {`Switch to ${nextRoomOption.label}`}
                    </button>
                  )}
                </div>
              </article>
            </div>

            <AnimatePresence>
              {libraryOpen && !loading && (
                <motion.div
                  key="library"
                  variants={PANEL_VARIANTS}
                  initial="hidden"
                  animate="visible"
                  exit="exit"
                >
                  <Bookshelf
                    stories={stories}
                    onSelectStory={handleSelectStory}
                    onGenerateNew={() => setGeneratorOpen(true)}
                    onToggleFavorite={handleToggleFavorite}
                    onDeleteStory={handleDeleteStory}
                    onCloseLibrary={hideLibraryPanel}
                    loading={loading}
                  />
                </motion.div>
              )}
            </AnimatePresence>

            {loading && (
              <div className="library-panel">
                <div className="panel-header">
                  <div>
                    <h2>Bedtime Library</h2>
                    <p>Loading bedtime stories…</p>
                  </div>
                </div>
                <div className="panel-loading">
                  <div className="spinner large" />
                </div>
              </div>
            )}
          </motion.div>
        </>
      </ChildrensRoom>

      {prefersDomPanels && (
        <>
          <AnimatePresence>
            {settingsOpen && (
              <SettingsPanel
                floorTexture={floorTexture}
                wallTexture={wallTexture}
                onChangeFloor={setFloorTexture}
                onChangeWall={setWallTexture}
                onClose={() => setSettingsOpen(false)}
              />
            )}
          </AnimatePresence>

          <AnimatePresence>
            {debugOpen && developerMode && (
              <SceneDebugPanel
                selectedId={debugSelection}
                onSelect={handleDebugSelect}
                onClose={() => {
                  setDebugOpen(false);
                  setDebugSelection(null);
                }}
              />
            )}
          </AnimatePresence>
        </>
      )}

      <AnimatePresence>
        {generatorOpen && (
          <StoryGenerator
            onGenerate={handleGenerateStory}
            onCancel={() => setGeneratorOpen(false)}
          />
        )}
      </AnimatePresence>

      <AnimatePresence>
        {isReading && selectedStory && (
          <BookReader story={selectedStory} onClose={handleCloseBook} />
        )}
      </AnimatePresence>

      <AnimatePresence>
        {parentDashboardOpen && (
          <ParentDashboard onClose={() => setParentDashboardOpen(false)} />
        )}
      </AnimatePresence>

      {prototypeOpen && (
        <ImmersivePrototype onClose={() => setPrototypeOpen(false)} />
      )}

      {developerMode && developerConsoleOpen && prefersDomPanels && (
        <aside className="developer-overlay" role="status" aria-live="polite">
          <header className="developer-overlay__header">
            <strong>Developer Console</strong>
            <span>{loaderDescription}</span>
          </header>
          <section className="developer-overlay__section">
            <h4>Camera Presets</h4>
            <div className="developer-overlay__hotspots">
              {Object.values(hotspots).map((hotspot) => (
                <button
                  key={hotspot.id}
                  type="button"
                  className={`developer-hotspot ${
                    hotspot.active ? "active" : ""
                  }`}
                  onClick={() => setCameraFocus(hotspot.id)}
                >
                  {hotspot.label}
                </button>
              ))}
            </div>
            <p className="developer-overlay__meta">
              Current focus: {cameraFocus} • Active room: {activeRoomLabel}
            </p>
            <button
              type="button"
              className="ghost-action"
              onClick={() => setCameraAutopilot(!cameraAutopilot)}
            >
              {cameraAutopilot ? "Unlock Camera" : "Lock Camera"}
            </button>
            {nextRoomOption && (
              <button
                type="button"
                className="ghost-action"
                onClick={() => setActiveRoom(nextRoomOption.id)}
              >
                {`Switch to ${nextRoomOption.label}`}
              </button>
            )}
          </section>
          <section className="developer-overlay__section">
            <h4>Story Context</h4>
            {selectedStory ? (
              <ul>
                <li>
                  <strong>Title:</strong> {selectedStory.title}
                </li>
                <li>
                  <strong>Theme:</strong> {selectedStory.theme || "—"}
                </li>
                <li>
                  <strong>Age:</strong> {selectedStory.age_group || "—"}
                </li>
              </ul>
            ) : (
              <p>No story selected</p>
            )}
          </section>
        </aside>
      )}
    </div>
  );
}

export default App;
