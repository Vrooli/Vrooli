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
import PhysicsSimulation from "./PhysicsSimulation.js";
import useExperienceStore from "../state/store.js";

const SCENE_COLORS = {
  day: "#bce3ff",
  evening: "#ffbb9a",
  night: "#0f112a",
};

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

    // Initialize post-processing and physics
    this.postProcessing = new PostProcessing(this);
    this.postProcessing.enabled = false;
    this.physicsSimulation = new PhysicsSimulation(this);
    this.physicsSimulation.disable();
    
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
      }),
      this._handleStoreChange,
    );
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
    if (this.scene?.background && this.sceneBackgroundTarget) {
      this.scene.background.lerp(this.sceneBackgroundTarget, 0.03);
    }
    this.camera.update(frame);
    this.navigation.update(frame);
    this.cameraRig.update(frame);
    this.cameraRailSystem.update(frame.delta);
    this.audioAmbience.update(frame.delta);
    this.world.update(frame);
    
    // Update physics simulation
    if (this.physicsSimulation) {
      this.physicsSimulation.update(frame.delta);
    }
    
    // Render with post-processing if available
    if (this.postProcessing && this.postProcessing.enabled) {
      this.postProcessing.render();
    } else {
      this.renderer.update(frame);
    }
  }

  _handleStoreChange(snapshot) {
    const targetColor = SCENE_COLORS[snapshot.timeOfDay] || SCENE_COLORS.day;
    this.sceneBackgroundTarget.set(targetColor);
    
    // Update post-processing for time of day
    if (this.postProcessing) {
      this.postProcessing.setTimeOfDay(snapshot.timeOfDay);
    }
    this.world.setStoreSnapshot(snapshot);
    if (this.physicsSimulation) {
      if (snapshot.developerMode) {
        this.physicsSimulation.enable();
      } else {
        this.physicsSimulation.disable();
      }
    }
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
      if (snapshot.developerMode) {
        this.postProcessing.enabled = true;
        this.physicsSimulation?.enable();
      } else {
        this.postProcessing.enabled = false;
        this.physicsSimulation?.disable();
      }
    }

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
}
