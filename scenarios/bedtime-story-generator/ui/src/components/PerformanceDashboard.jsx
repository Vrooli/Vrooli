import React, { useEffect, useState, useRef } from "react";
import { motion } from "framer-motion";
import useExperienceStore from "../state/store.js";
import { resetLongTaskHistory } from "../performance/longTaskCache.js";
import FeatureToggleDialog from "./FeatureToggleDialog.jsx";

const PANEL_VARIANTS = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: 10 },
};

const DEFAULT_PROFILE_INTERVAL = 15;

const PERFORMANCE_TARGETS = {
  fps: { min: 30, target: 60, label: "FPS" },
  drawCalls: { max: 150, warning: 100, label: "Draw Calls" },
  triangles: { max: 500000, warning: 300000, label: "Triangles" },
  textures: { max: 50, warning: 30, label: "Textures" },
  geometries: { max: 100, warning: 70, label: "Geometries" },
  memory: { max: 500, warning: 300, label: "Memory (MB)" },
  renderTime: { max: 16.67, warning: 12, label: "Render Time (ms)" },
};

const PerformanceDashboard = ({ experience, isOpen, onToggle }) => {
  const frameProfile = useExperienceStore((state) => state.frameProfile);
  const longTasks = useExperienceStore((state) => state.longTasks);
  const gcEvents = useExperienceStore((state) => state.gcEvents);
  const clearLongTasks = useExperienceStore((state) => state.clearLongTasks);
  const clearGCEvents = useExperienceStore((state) => state.clearGCEvents);
  const frameProfileTotal = frameProfile?.total ?? 0;
  const [togglesOpen, setTogglesOpen] = useState(false);
  const [metrics, setMetrics] = useState({
    fps: 0,
    frameTime: 0,
    drawCalls: 0,
    triangles: 0,
    textures: 0,
    geometries: 0,
    memory: 0,
    renderTime: 0,
  });

  const [history, setHistory] = useState({
    fps: [],
    renderTime: [],
  });

  const frameCountRef = useRef(0);
  const lastTimeRef = useRef(performance.now());
  const animationFrameRef = useRef(null);

  useEffect(() => {
    if (!experience) {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      return undefined;
    }

    let mounted = true;

    const updateMetrics = () => {
      if (!mounted) {
        return;
      }

      const now = performance.now();
      const deltaTime = now - lastTimeRef.current;
      frameCountRef.current += 1;

      if (deltaTime >= 500) {
        const framesCaptured = frameCountRef.current;
        const fps = framesCaptured > 0 ? Math.round((framesCaptured * 1000) / deltaTime) : 0;

        frameCountRef.current = 0;
        lastTimeRef.current = now;

        const renderer = experience.renderer?.instance;
        const scene = experience.scene;

        if (renderer && scene) {
          const info = renderer.info;
          const memoryInfo = info.memory;
          const renderInfo = info.render;

          const memoryUsage = performance.memory
            ? Math.round(performance.memory.usedJSHeapSize / 1048576)
            : 0;

          const safeFps = Number.isFinite(fps) && fps > 0 ? fps : 0;
          const frameTime = safeFps > 0 ? Math.round((1000 / safeFps) * 100) / 100 : 0;
          const renderTime = framesCaptured > 0
            ? Math.round((deltaTime / framesCaptured) * 100) / 100
            : 0;

          const nextMetrics = {
            fps: safeFps,
            frameTime,
            drawCalls: renderInfo.calls || 0,
            triangles: renderInfo.triangles || 0,
            textures: memoryInfo.textures || 0,
            geometries: memoryInfo.geometries || 0,
            memory: memoryUsage,
            renderTime,
          };

          setMetrics(nextMetrics);
          setHistory((prev) => ({
            fps: [...prev.fps.slice(-59), nextMetrics.fps],
            renderTime: [...prev.renderTime.slice(-59), nextMetrics.renderTime],
          }));
        }
      }

      animationFrameRef.current = requestAnimationFrame(updateMetrics);
    };

    animationFrameRef.current = requestAnimationFrame(updateMetrics);

    return () => {
      mounted = false;
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, [experience]);

  const getStatusColor = (value, target) => {
    if (target.min) {
      // For FPS (higher is better)
      if (value >= target.target) return "text-green-500";
      if (value >= target.min) return "text-yellow-500";
      return "text-red-500";
    } else {
      // For other metrics (lower is better)
      if (value <= target.warning) return "text-green-500";
      if (value <= target.max) return "text-yellow-500";
      return "text-red-500";
    }
  };

  const getStatusIcon = (value, target) => {
    if (target.min) {
      if (value >= target.target) return "✅";
      if (value >= target.min) return "⚠️";
      return "❌";
    } else {
      if (value <= target.warning) return "✅";
      if (value <= target.max) return "⚠️";
      return "❌";
    }
  };

  if (!isOpen) {
    return (
      <button
        className="performance-toggle"
        onClick={onToggle}
        aria-label="Show performance dashboard"
      >
        📊 {metrics.fps} FPS
      </button>
    );
  }

  return (
    <motion.div
      className="performance-dashboard"
      initial="hidden"
      animate="visible"
      exit="exit"
      variants={PANEL_VARIANTS}
    >
      <div className="performance-header">
        <h3>Performance Monitor</h3>
        <div className="performance-actions">
          <button
            type="button"
            className="ghost-action"
            onClick={() => setTogglesOpen(true)}
          >
            Manage Toggles
          </button>
          <button
            className="close-button"
            onClick={onToggle}
            aria-label="Close performance dashboard"
          >
            ×
          </button>
        </div>
      </div>

      <div className="performance-metrics">
        {/* FPS Section */}
        <div className="metric-section">
          <div className="metric-row">
            <span className="metric-label">FPS</span>
            <span className={`metric-value ${getStatusColor(metrics.fps, PERFORMANCE_TARGETS.fps)}`}>
              {metrics.fps} {getStatusIcon(metrics.fps, PERFORMANCE_TARGETS.fps)}
            </span>
          </div>
          <div className="fps-graph">
            <svg width="100%" height="40" viewBox="0 0 60 40">
              <polyline
                fill="none"
                stroke="currentColor"
                strokeWidth="1"
                points={history.fps
                  .map((fps, i) => `${i},${40 - (fps / 60) * 40}`)
                  .join(" ")}
              />
            </svg>
          </div>
        </div>

        {/* Draw Calls */}
        <div className="metric-row">
          <span className="metric-label">Draw Calls</span>
          <span className={`metric-value ${getStatusColor(metrics.drawCalls, PERFORMANCE_TARGETS.drawCalls)}`}>
            {metrics.drawCalls} {getStatusIcon(metrics.drawCalls, PERFORMANCE_TARGETS.drawCalls)}
          </span>
        </div>

        {/* Triangles */}
        <div className="metric-row">
          <span className="metric-label">Triangles</span>
          <span className={`metric-value ${getStatusColor(metrics.triangles, PERFORMANCE_TARGETS.triangles)}`}>
            {metrics.triangles.toLocaleString()} {getStatusIcon(metrics.triangles, PERFORMANCE_TARGETS.triangles)}
          </span>
        </div>

        {/* Textures */}
        <div className="metric-row">
          <span className="metric-label">Textures</span>
          <span className={`metric-value ${getStatusColor(metrics.textures, PERFORMANCE_TARGETS.textures)}`}>
            {metrics.textures} {getStatusIcon(metrics.textures, PERFORMANCE_TARGETS.textures)}
          </span>
        </div>

        {/* Geometries */}
        <div className="metric-row">
          <span className="metric-label">Geometries</span>
          <span className={`metric-value ${getStatusColor(metrics.geometries, PERFORMANCE_TARGETS.geometries)}`}>
            {metrics.geometries} {getStatusIcon(metrics.geometries, PERFORMANCE_TARGETS.geometries)}
          </span>
        </div>

        {/* Memory */}
        {metrics.memory > 0 && (
          <div className="metric-row">
            <span className="metric-label">Memory</span>
            <span className={`metric-value ${getStatusColor(metrics.memory, PERFORMANCE_TARGETS.memory)}`}>
              {metrics.memory} MB {getStatusIcon(metrics.memory, PERFORMANCE_TARGETS.memory)}
            </span>
          </div>
        )}

        {/* Render Time */}
        <div className="metric-row">
          <span className="metric-label">Frame Time</span>
          <span className={`metric-value ${getStatusColor(metrics.renderTime, PERFORMANCE_TARGETS.renderTime)}`}>
            {metrics.renderTime} ms {getStatusIcon(metrics.renderTime, PERFORMANCE_TARGETS.renderTime)}
          </span>
        </div>
      </div>

      {frameProfile && frameProfile.sections?.length > 0 && (
        <div className="frame-profile">
          <div className="frame-profile__header">
            <h4>Frame Breakdown</h4>
            <span>
              Sampled every {frameProfile.interval || DEFAULT_PROFILE_INTERVAL} frames • Total {frameProfileTotal.toFixed(2)} ms
              {frameProfile.rawDelta ? ` • rAF interval ${frameProfile.rawDelta.toFixed(1)} ms` : ""}
            </span>
          </div>
          <ul className="frame-profile__list">
            {[...frameProfile.sections]
              .sort((a, b) => b.duration - a.duration)
              .slice(0, 6)
              .map((section) => (
                <li key={section.label}>
                  <span className="frame-profile__label">{section.label}</span>
                  <span className="frame-profile__value">
                    {section.duration.toFixed(2)} ms
                    <span className="frame-profile__percent">{section.pct.toFixed(0)}%</span>
                  </span>
                </li>
              ))}
          </ul>
          <div className="frame-profile__footnote">
            <span>
              Draw Calls: {frameProfile.renderer?.drawCalls ?? 0} • Triangles: {frameProfile.renderer?.triangles?.toLocaleString?.() ?? frameProfile.renderer?.triangles ?? 0} • Geometries: {frameProfile.renderer?.geometries ?? 0}
            </span>
          </div>
        </div>
      )}

      {longTasks?.length > 0 && (
        <div className="longtask-panel">
          <div className="longtask-header">
            <h4>Recent Long Tasks</h4>
            <button
              type="button"
              onClick={() => {
                clearLongTasks();
                resetLongTaskHistory();
              }}
              className="ghost-action small"
            >
              Clear
            </button>
          </div>
          <ul>
            {[...longTasks]
              .slice(-6)
              .reverse()
              .map((task, index) => (
                <li key={`${task.timestamp}-${index}`}>
                  <div className="longtask-row">
                    <span className="longtask-duration">{task.duration.toFixed(1)} ms</span>
                    <span className="longtask-label">{task.name || "anonymous"}</span>
                  </div>
                  {task.attribution?.length > 0 && (
                    <ul className="longtask-attribution">
                      {task.attribution.map((item, idx) => (
                        <li key={idx}>
                          <span>{item.name || item.entryType}</span>
                          <span>{item.duration?.toFixed?.(1) ?? "—"} ms</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
          </ul>
        </div>
      )}

      {gcEvents?.length > 0 && (
        <div className="gc-panel">
          <div className="longtask-header">
            <h4>GC Events</h4>
            <button type="button" onClick={clearGCEvents} className="ghost-action small">
              Clear
            </button>
          </div>
          <ul>
            {[...gcEvents]
              .slice(-6)
              .reverse()
              .map((event, index) => (
                <li key={`${event.timestamp}-${index}`}>
                  <div className="longtask-row">
                    <span className="longtask-duration">{event.duration.toFixed(1)} ms</span>
                    <span className="longtask-label">{event.type || "gc"}</span>
                  </div>
                </li>
              ))}
          </ul>
        </div>
      )}

      <div className="performance-footer">
        <div className="performance-legend">
          <span>✅ Good</span>
          <span>⚠️ Warning</span>
          <span>❌ Critical</span>
        </div>
        <button
          className="reset-button"
          onClick={() => {
            if (experience?.renderer?.instance?.info) {
              experience.renderer.instance.info.reset();
            }
            setHistory({ fps: [], renderTime: [] });
          }}
        >
          Reset Stats
        </button>
      </div>

      <style>{`
        .performance-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          gap: 12px;
        }
        .performance-actions {
          display: flex;
          align-items: center;
          gap: 8px;
        }
        .frame-profile {
          margin-top: 16px;
          padding-top: 16px;
          border-top: 1px solid rgba(255, 255, 255, 0.1);
          display: flex;
          flex-direction: column;
          gap: 12px;
        }
        .frame-profile__header {
          display: flex;
          justify-content: space-between;
          font-size: 12px;
          color: rgba(255, 255, 255, 0.7);
        }
        .frame-profile__header h4 {
          margin: 0;
          font-size: 14px;
          color: #b8b8ff;
        }
        .frame-profile__list {
          list-style: none;
          margin: 0;
          padding: 0;
          display: flex;
          flex-direction: column;
          gap: 6px;
        }
        .frame-profile__list li {
          display: flex;
          justify-content: space-between;
          font-size: 13px;
          font-family: monospace;
        }
        .frame-profile__value {
          display: flex;
          align-items: center;
          gap: 6px;
        }
        .frame-profile__percent {
          font-size: 11px;
          color: rgba(255, 255, 255, 0.5);
        }
        .frame-profile__footnote {
          font-size: 11px;
          color: rgba(255, 255, 255, 0.5);
        }
        .longtask-panel {
          margin-top: 16px;
          padding-top: 16px;
          border-top: 1px solid rgba(255, 255, 255, 0.1);
          display: flex;
          flex-direction: column;
          gap: 8px;
        }
        .gc-panel {
          margin-top: 12px;
          padding-top: 12px;
          border-top: 1px solid rgba(255, 255, 255, 0.08);
          display: flex;
          flex-direction: column;
          gap: 6px;
          font-family: monospace;
          font-size: 12px;
        }
        .gc-panel ul {
          list-style: none;
          margin: 0;
          padding: 0;
          display: flex;
          flex-direction: column;
          gap: 4px;
        }
        .longtask-panel ul {
          list-style: none;
          margin: 0;
          padding: 0;
          display: flex;
          flex-direction: column;
          gap: 6px;
          font-family: monospace;
          font-size: 12px;
        }
        .longtask-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          font-size: 12px;
          color: rgba(255, 255, 255, 0.7);
        }
        .longtask-header h4 {
          margin: 0;
          font-size: 14px;
          color: #b8b8ff;
        }
        .longtask-row {
          display: flex;
          justify-content: space-between;
          align-items: center;
          gap: 8px;
        }
        .longtask-duration {
          color: #f87171;
        }
        .longtask-attribution {
          list-style: none;
          margin: 4px 0 0 0;
          padding: 0 0 0 8px;
          border-left: 1px solid rgba(255, 255, 255, 0.2);
          display: flex;
          flex-direction: column;
          gap: 2px;
          font-size: 11px;
          color: rgba(255, 255, 255, 0.6);
        }
        .ghost-action.small {
          font-size: 11px;
          padding: 2px 6px;
        }
      `}</style>

      <FeatureToggleDialog open={togglesOpen} onClose={() => setTogglesOpen(false)} />
    </motion.div>
  );
};

export default PerformanceDashboard;
