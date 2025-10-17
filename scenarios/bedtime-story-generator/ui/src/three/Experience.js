import * as THREE from "three";
import Time from "./core/Time.js";
import Sizes from "./core/Sizes.js";
import Camera from "./Camera.js";
import Renderer from "./Renderer.js";
import World from "./World.js";
import CameraRig from "./CameraRig.js";
import CameraRailSystem from "./CameraRailSystem.js";
import AudioAmbience from "./AudioAmbience.js";
import HotspotRegistry from "./HotspotRegistry.js";
import ResourceLoader from "./ResourceLoader.js";
import Navigation from "./Navigation.js";
import PostProcessing from "./PostProcessing.js";
import useExperienceStore from "../state/store.js";
import { pushLongTasks } from "../performance/longTaskCache.js";

const SCENE_COLORS = {
  day: "#bce3ff",
  evening: "#ffbb9a",
  night: "#0f112a",
};

const PROFILE_SAMPLE_INTERVAL = 15;

class FrameProfiler {
  constructor() {
    this.enabled = typeof performance !== "undefined";
    this.sections = [];
    this.frameCounter = 0;
    this.sampleInterval = PROFILE_SAMPLE_INTERVAL;
  }

  begin() {
    if (!this.enabled) return;
    this.startTime = performance.now();
    this.lastMark = this.startTime;
    this.sections = [];
  }

  mark(label) {
    if (!this.enabled || !this.startTime) return;
    const now = performance.now();
    this.sections.push({ label, duration: now - this.lastMark });
    this.lastMark = now;
  }

  end({ renderer, frame }) {
    if (!this.enabled || !this.startTime) return;
    const endTime = performance.now();
    const total = endTime - this.startTime;

    this.frameCounter += 1;
    if (this.frameCounter % this.sampleInterval !== 0) {
      return;
    }

    const info = renderer?.info;
    const renderInfo = info?.render || {};
    const memoryInfo = info?.memory || {};

    const snapshot = {
      total,
      sections: this.sections
        .map((section) => ({
          label: section.label,
          duration: section.duration,
          pct: total > 0 ? (section.duration / total) * 100 : 0,
        }))
        .filter((section) => section.duration >= 0.01),
      renderer: {
        drawCalls: renderInfo.calls || 0,
        triangles: renderInfo.triangles || 0,
        points: renderInfo.points || 0,
        lines: renderInfo.lines || 0,
        geometries: memoryInfo.geometries || 0,
        textures: memoryInfo.textures || 0,
      },
      timestamp: endTime,
      interval: this.sampleInterval,
      rawDelta: frame?.rawDelta,
    };

    const store = useExperienceStore.getState();
    store.recordFrameProfile?.(snapshot);
  }
}

class PerformanceMonitor {
  constructor() {
    const Observer = window?.PerformanceObserver;
    if (!Observer) {
      this.enabled = false;
      return;
    }

    this.enabled = true;
    this.longTaskObserver = null;
    this.gcObserver = null;

    const supported = Observer.supportedEntryTypes || [];
    if (supported.includes("longtask")) {
      this.longTaskObserver = new Observer((list) => {
        const entries = list.getEntries();
        const store = useExperienceStore.getState();
        pushLongTasks(entries);
        entries.forEach((entry) => {
          const payload = {
            name: entry.name,
            duration: entry.duration,
            startTime: entry.startTime,
            attribution: (entry.attribution || []).map((item) => ({
              name: item.name,
              entryType: item.entryType,
              startTime: item.startTime,
              duration: item.duration,
              containerType: item.containerType,
            })),
            timestamp: performance.now(),
          };
          store.recordLongTask?.(payload);
        });
      });
      this.longTaskObserver.observe({ entryTypes: ["longtask"] });
    }

    if (supported.includes("gc")) {
      this.gcObserver = new Observer((list) => {
        const entries = list.getEntries();
        const store = useExperienceStore.getState();
        entries.forEach((entry) => {
          const payload = {
            duration: entry.duration,
            startTime: entry.startTime,
            type: entry.detail?.type || entry.name,
            timestamp: performance.now(),
          };
          store.recordGCEvent?.(payload);
        });
      });
      this.gcObserver.observe({ entryTypes: ["gc"] });
    }
  }

  destroy() {
    this.longTaskObserver?.disconnect?.();
    this.gcObserver?.disconnect?.();
  }
}

export default class Experience {
  constructor({ target }) {
    if (!target) {
      throw new Error("Experience requires a target element.");
    }

    this.target = target;
    this.scene = new THREE.Scene();

    const initialState = useExperienceStore.getState();
    this.currentRoom = initialState.activeRoom || "studio";
    const initialSceneColor = new THREE.Color(
      SCENE_COLORS[initialState.timeOfDay] || "#0b1026",
    );
    this.scene.background = initialSceneColor;
    this.sceneBackgroundTarget = initialSceneColor.clone();

    this.sizes = new Sizes({ target: this.target });
    this.time = new Time();

    this.camera = new Camera({ experience: this });
    this.renderer = new Renderer({ experience: this });
    this.world = new World({ experience: this, initialState });
    this.cameraRig = new CameraRig({ experience: this, initialRoom: this.currentRoom });
    this.cameraRailSystem = new CameraRailSystem({
      camera: this.camera.instance,
      scene: this.scene,
    });
    this.audioAmbience = new AudioAmbience({
      scene: this.scene,
      camera: this.camera.instance,
    });
    this.hotspotRegistry = new HotspotRegistry();
    this.navigation = new Navigation({ experience: this, initialRoom: this.currentRoom });
    this.resourceLoader = new ResourceLoader({ initialRoom: this.currentRoom });
    this._ensureRoomAssets(this.currentRoom);

    ["studio", "bedroom"].forEach((roomId) => {
      if (roomId === this.currentRoom) {
        return;
      }
      this.resourceLoader.loadRoom(roomId).catch((error) => {
        console.warn(`[Experience] Failed to prefetch room ${roomId}`, error);
      });
    });

    // Initialize post-processing (off by default for performance)
    this.postProcessing = new PostProcessing(this);
    this.postProcessing.enabled = false;

    this.frameProfiler = new FrameProfiler();
    this.performanceMonitor = new PerformanceMonitor();
    this.profiling = initialState.profiling || {};
    this.prevHeapUsage = performance.memory?.usedJSHeapSize ?? null;

    // Subscribe to asset reload events for key assets
    this._setupHotReloadSubscriptions();

    this._handleResize = this._handleResize.bind(this);
    this._handleTick = this._handleTick.bind(this);
    this._handleStoreChange = this._handleStoreChange.bind(this);

    this.sizes.on("resize", this._handleResize);
    this.time.on("tick", this._handleTick);

    this.unsubscribeStore = useExperienceStore.subscribe(
      (state) => ({
        timeOfDay: state.timeOfDay,
        selectedStory: state.selectedStory,
        cameraFocus: state.cameraFocus,
        activeRoom: state.activeRoom,
        cameraAutopilot: state.cameraAutopilot,
        developerMode: state.developerMode,
        profiling: state.profiling,
      }),
      this._handleStoreChange,
    );

    this._applyProfilingSettings();
  }

  get config() {
    return {
      width: this.sizes.width,
      height: this.sizes.height,
      pixelRatio: this.sizes.pixelRatio,
    };
  }

  _handleResize() {
    this.camera.resize();
    this.renderer.resize();
    this.postProcessing?.resize?.();
  }

  _handleTick(frame) {
    this.frameProfiler.begin();

    if (this.scene?.background && this.sceneBackgroundTarget) {
      this.scene.background.lerp(this.sceneBackgroundTarget, 0.03);
    }
    this.frameProfiler.mark("background");
    this.camera.update(frame);
    this.frameProfiler.mark("camera");
    this.navigation.update(frame);
    this.frameProfiler.mark("navigation");
    this.cameraRig.update(frame);
    this.frameProfiler.mark("cameraRig");
    this.cameraRailSystem.update(frame.delta);
    this.frameProfiler.mark("cameraRail");
    if (!this.profiling?.disableAudio) {
      this.audioAmbience.update(frame.delta);
      this.frameProfiler.mark("audio");
    }
    this.world.update(frame);
    this.frameProfiler.mark("world");
    
    // Render with post-processing if available
    if (this.postProcessing && this.postProcessing.enabled && !this.profiling?.disablePostProcessing) {
      this.postProcessing.render();
      this.frameProfiler.mark("postProcessing");
    } else {
      this.renderer.update(frame);
      this.frameProfiler.mark("render");
    }

    this.frameProfiler.end({ renderer: this.renderer.instance, frame });

    if (performance.memory) {
      const currentHeap = performance.memory.usedJSHeapSize;
      if (this.prevHeapUsage != null) {
        const delta = currentHeap - this.prevHeapUsage;
        if (delta < -2_000_000) {
          const store = useExperienceStore.getState();
          store.recordGCEvent?.({
            duration: Math.abs(delta) / 1_000,
            startTime: frame?.now ?? performance.now(),
            type: "heap-drop",
            timestamp: performance.now(),
            delta,
            current: currentHeap,
          });
        }
      }
      this.prevHeapUsage = currentHeap;
    }
  }

  _handleStoreChange(snapshot) {
    this.storeSnapshot = snapshot;
    const targetColor = SCENE_COLORS[snapshot.timeOfDay] || SCENE_COLORS.day;
    this.sceneBackgroundTarget.set(targetColor);
    this.profiling = snapshot.profiling || this.profiling;
    
    // Update post-processing for time of day
    if (this.postProcessing) {
      this.postProcessing.setTimeOfDay(snapshot.timeOfDay);
    }
    this.world.setStoreSnapshot(snapshot);
    this.world.setProfilingSettings(this.profiling);
    if (snapshot.activeRoom && snapshot.activeRoom !== this.currentRoom) {
      this.currentRoom = snapshot.activeRoom;
      this.cameraRig.updateRoom(this.currentRoom);
      this.navigation.updateRoom(this.currentRoom);
      this._ensureRoomAssets(this.currentRoom);
    }

    this.cameraRig.updateFocus(snapshot.cameraFocus, this.currentRoom);
    this.navigation.setAutopilot(snapshot.cameraAutopilot);

    // Update audio ambience based on time of day
    if (this.audioAmbience && snapshot.timeOfDay) {
      this.audioAmbience.setTimeOfDay(snapshot.timeOfDay);
    }
    
    // Update story mood for audio if a story is selected
    if (this.audioAmbience && snapshot.selectedStory && snapshot.selectedStory.theme) {
      this.audioAmbience.setStoryMood(snapshot.selectedStory.theme.toLowerCase());
    }

    if (typeof snapshot.developerMode === "boolean") {
      this.postProcessing.enabled = snapshot.developerMode && !this.profiling?.disablePostProcessing;
    }

    this._applyProfilingSettings();

    // Play camera intro rail on first load
    if (this.cameraRailSystem && !this.hasPlayedIntro) {
      this.cameraRailSystem.playRail("intro");
      this.hasPlayedIntro = true;
    }
  }

  _setupHotReloadSubscriptions() {
    this.assetUnsubscribers = [];
    
    // Subscribe to room model updates
    this.assetUnsubscribers.push(
      this.resourceLoader.onAssetReload("model_room", (newModel) => {
        console.log("[Experience] Room model reloaded");
        this.world.applyRoomAssets(this.currentRoom, { model_room: newModel });
      })
    );
    
    // Subscribe to texture updates
    this.assetUnsubscribers.push(
      this.resourceLoader.onAssetReload("baked_day", (newTexture) => {
        console.log("[Experience] Day texture reloaded");
        this.world.applyRoomAssets(this.currentRoom, { baked_day: newTexture });
      })
    );
    
    // Subscribe to HDR updates
    this.assetUnsubscribers.push(
      this.resourceLoader.onAssetReload("hdr_day", (newHDR) => {
        console.log("[Experience] HDR environment reloaded");
        this.world.applyRoomAssets(this.currentRoom, { hdr_day: newHDR });
      })
    );
  }

  destroy() {
    this.time.stop();
    
    // Clean up hot reload subscriptions
    if (this.assetUnsubscribers) {
      this.assetUnsubscribers.forEach((unsubscribe) => unsubscribe());
    }
    
    this.sizes.destroy();
    this.renderer.destroy();
    this.camera.destroy();
    this.world.destroy();
    this.cameraRig.destroy();
    this.cameraRailSystem?.destroy();
    this.audioAmbience?.destroy();
    this.hotspotRegistry.destroy();
    this.resourceLoader.destroy();
    this.navigation.destroy?.();
    this.unsubscribeStore?.();
    this.performanceMonitor?.destroy?.();

    this.scene.traverse((child) => {
      if (child.geometry) child.geometry.dispose?.();
      if (child.material) {
        if (Array.isArray(child.material)) {
          child.material.forEach((mat) => mat.dispose?.());
        } else {
          child.material.dispose?.();
        }
      }
    });
  }

  _ensureRoomAssets(roomId) {
    const token = Symbol(roomId);
    this.pendingRoomToken = token;

    this.resourceLoader
      .loadRoom(roomId)
      .then((assets) => {
        if (this.pendingRoomToken !== token) return;
        this.world.applyRoomAssets(roomId, assets);
        this.world.setActiveRoom(roomId);
      })
      .catch((error) => {
        console.error("Failed to load room assets", roomId, error);
      });
  }

  _applyProfilingSettings() {
    if (this.audioAmbience) {
      this.audioAmbience.setEnabled(!this.profiling?.disableAudio);
    }
    if (this.postProcessing) {
      this.postProcessing.enabled = !!this.storeSnapshot?.developerMode && !this.profiling?.disablePostProcessing;
    }
    if (this.world) {
      this.world.setProfilingSettings(this.profiling);
    }
  }
}
