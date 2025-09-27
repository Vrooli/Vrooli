import React, { useEffect, useState, useRef } from "react";
import { motion } from "framer-motion";
import * as THREE from "three";

const PANEL_VARIANTS = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: 10 },
};

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
    if (!experience || !isOpen) {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      return;
    }

    const updateMetrics = () => {
      const now = performance.now();
      const deltaTime = now - lastTimeRef.current;
      frameCountRef.current++;

      // Update FPS every 500ms
      if (deltaTime >= 500) {
        const fps = Math.round((frameCountRef.current * 1000) / deltaTime);
        frameCountRef.current = 0;
        lastTimeRef.current = now;

        const renderer = experience.renderer?.instance;
        const scene = experience.scene;

        if (renderer && scene) {
          const info = renderer.info;
          const memory = info.memory;
          const render = info.render;

          // Calculate memory usage
          const memoryUsage = performance.memory
            ? Math.round(performance.memory.usedJSHeapSize / 1048576)
            : 0;

          const newMetrics = {
            fps,
            frameTime: Math.round(1000 / fps * 100) / 100,
            drawCalls: render.calls || 0,
            triangles: render.triangles || 0,
            textures: memory.textures || 0,
            geometries: memory.geometries || 0,
            memory: memoryUsage,
            renderTime: Math.round(deltaTime / frameCountRef.current * 100) / 100,
          };

          setMetrics(newMetrics);

          // Update history (keep last 60 samples)
          setHistory((prev) => ({
            fps: [...prev.fps.slice(-59), fps],
            renderTime: [...prev.renderTime.slice(-59), newMetrics.renderTime],
          }));
        }
      }

      animationFrameRef.current = requestAnimationFrame(updateMetrics);
    };

    updateMetrics();

    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, [experience, isOpen]);

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
        <button
          className="close-button"
          onClick={onToggle}
          aria-label="Close performance dashboard"
        >
          ×
        </button>
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
    </motion.div>
  );
};

export default PerformanceDashboard;