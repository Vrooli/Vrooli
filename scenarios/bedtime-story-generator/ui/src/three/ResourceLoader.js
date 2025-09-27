import * as THREE from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { RGBELoader } from "three/examples/jsm/loaders/RGBELoader.js";
import { DRACOLoader } from "three/examples/jsm/loaders/DRACOLoader.js";
import assetManifest, {
  ROOM_ASSET_MANIFEST,
  SHARED_ASSETS,
} from "../assets/manifest.js";
import useExperienceStore from "../state/store.js";

const loaderStore = useExperienceStore;

const ROOM_LABELS = {
  studio: "Loading studio assets",
  bedroom: "Loading bedroom suite",
};

const DEFAULT_ROOM = "studio";

export default class ResourceLoader {
  constructor({ initialRoom = DEFAULT_ROOM } = {}) {
    this.manager = new THREE.LoadingManager();
    this.textureLoader = new THREE.TextureLoader(this.manager);
    this.hdrLoader = new RGBELoader(this.manager);
    this.dracoLoader = new DRACOLoader(this.manager);
    this.dracoLoader.setDecoderPath(SHARED_ASSETS.dracoDecoder || "");
    this.dracoLoader.setDecoderConfig({ type: "js" });

    this.gltfLoader = new GLTFLoader(this.manager);
    this.gltfLoader.setDRACOLoader(this.dracoLoader);

    this.roomAssets = {};
    this.loadingPromises = {};
    this.expectedTotal = 0;
    this.currentRoom = initialRoom;
    this.hotReloadCallbacks = new Map();
    this.fileWatcher = null;

    this._setupManager();
    this._setupHotReload();
  }

  _setupManager() {
    this.manager.onProgress = (_, loaded, total) => {
      loaderStore.getState().updateLoader({ loaded, total });
    };

    this.manager.onLoad = () => {
      const state = loaderStore.getState();
      if (state.loader.status === "loading") {
        state.completeLoader();
      }
    };
  }

  async loadRoom(roomId = DEFAULT_ROOM) {
    if (this.roomAssets[roomId]) {
      return this.roomAssets[roomId];
    }

    if (this.loadingPromises[roomId]) {
      return this.loadingPromises[roomId];
    }

    const manifest = ROOM_ASSET_MANIFEST[roomId];
    if (!manifest) {
      console.warn(`[ResourceLoader] Unknown room manifest: ${roomId}`);
      return {};
    }

    const bucket = {};
    const tasks = [];

    const enqueueTexture = (key, path, encoder = (texture) => texture) => {
      tasks.push(
        new Promise((resolve) => {
          this.textureLoader.load(
            path,
            (texture) => {
              const processed = encoder(texture);
              bucket[`baked_${key}`] = processed;
              resolve({ key, success: true });
            },
            undefined,
            (error) => {
              console.error(`Failed to load texture ${path}`, error);
              resolve({ key, success: false });
            },
          );
        }),
      );
    };

    const enqueueGLTF = (key, path) => {
      tasks.push(
        new Promise((resolve) => {
          this.gltfLoader.load(
            path,
            (gltf) => {
              bucket[`model_${key}`] = gltf;
              resolve({ key, success: true });
            },
            undefined,
            (error) => {
              console.error(`Failed to load model ${path}`, error);
              resolve({ key, success: false });
            },
          );
        }),
      );
    };

    const enqueueHDR = (key, path) => {
      tasks.push(
        new Promise((resolve) => {
          this.hdrLoader.setDataType(THREE.FloatType).load(
            path,
            (texture) => {
              texture.mapping = THREE.EquirectangularReflectionMapping;
              bucket[`hdr_${key}`] = texture;
              resolve({ key, success: true });
            },
            undefined,
            (error) => {
              console.error(`Failed to load HDR ${path}`, error);
              resolve({ key, success: false });
            },
          );
        }),
      );
    };

    if (manifest.baked) {
      Object.entries(manifest.baked).forEach(([key, path]) => {
        enqueueTexture(key, path, (texture) => {
          texture.encoding = THREE.sRGBEncoding;
          texture.flipY = false;
          return texture;
        });
      });
    }

    if (manifest.hdr) {
      Object.entries(manifest.hdr).forEach(([key, path]) => {
        enqueueHDR(key, path);
      });
    }

    if (manifest.textures) {
      Object.entries(manifest.textures).forEach(([key, path]) => {
        enqueueTexture(key, path, (texture) => {
          texture.encoding = THREE.sRGBEncoding;
          texture.flipY = false;
          bucket[`texture_${key}`] = texture;
          return texture;
        });
      });
    }

    if (manifest.models) {
      Object.entries(manifest.models).forEach(([key, path]) => {
        enqueueGLTF(key, path);
      });
    }

    this.expectedTotal = tasks.length;

    if (this.expectedTotal > 0) {
      loaderStore.getState().startLoader({
        group: `experience-assets-${roomId}`,
        total: this.expectedTotal,
        label: ROOM_LABELS[roomId] || "Loading immersive assets",
      });
    }

    const roomPromise = Promise.allSettled(tasks).then(() => {
      if (roomId === "studio") {
        this._applyStudioFallbacks(bucket);
      }

      this.roomAssets[roomId] = bucket;
      delete this.loadingPromises[roomId];
      return bucket;
    });

    this.loadingPromises[roomId] = roomPromise;
    return roomPromise;
  }

  _applyStudioFallbacks(bucket) {
    if (!bucket.model_room) {
      bucket.model_room = { scene: this._createFallbackRoom() };
    }

    if (!bucket.model_accessories) {
      bucket.model_accessories = {
        scene: this._createFallbackAccessories(),
      };
    }

    if (!bucket.baked_day) {
      const texture = new THREE.Texture();
      texture.needsUpdate = true;
      bucket.baked_day = texture;
    }
  }

  _setupHotReload() {
    if (import.meta.hot) {
      // HMR support for Vite
      import.meta.hot.accept("../assets/manifest.js", async () => {
        console.log("[ResourceLoader] Asset manifest updated, reloading...");
        await this.reloadCurrentRoom();
      });
    }

    // Keyboard shortcut for manual reload
    if (typeof window !== "undefined") {
      window.addEventListener("keydown", this._handleHotkey.bind(this));
    }
  }

  _handleHotkey(event) {
    // Ctrl/Cmd + Shift + R to reload assets
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === "R") {
      event.preventDefault();
      console.log("[ResourceLoader] Manual asset reload triggered");
      this.reloadCurrentRoom();
    }
  }

  async reloadAsset(assetKey, assetPath) {
    console.log(`[ResourceLoader] Reloading asset: ${assetKey}`);
    
    // Clear cache for this specific asset
    if (assetPath) {
      // Add timestamp to bypass browser cache
      const cacheBuster = `${assetPath}?t=${Date.now()}`;
      
      if (assetKey.startsWith("model_")) {
        return new Promise((resolve) => {
          this.gltfLoader.load(
            cacheBuster,
            (gltf) => {
              const roomId = this.currentRoom;
              if (this.roomAssets[roomId]) {
                // Dispose old asset
                this._disposeAsset(this.roomAssets[roomId][assetKey]);
                // Replace with new
                this.roomAssets[roomId][assetKey] = gltf;
              }
              
              // Trigger callbacks
              const callbacks = this.hotReloadCallbacks.get(assetKey);
              if (callbacks) {
                callbacks.forEach((cb) => cb(gltf));
              }
              
              resolve(gltf);
            },
            undefined,
            (error) => {
              console.error(`Failed to reload asset ${assetKey}`, error);
              resolve(null);
            },
          );
        });
      } else if (assetKey.startsWith("texture_") || assetKey.startsWith("baked_")) {
        return new Promise((resolve) => {
          this.textureLoader.load(
            cacheBuster,
            (texture) => {
              texture.encoding = THREE.sRGBEncoding;
              texture.flipY = false;
              
              const roomId = this.currentRoom;
              if (this.roomAssets[roomId]) {
                // Dispose old texture
                this._disposeAsset(this.roomAssets[roomId][assetKey]);
                // Replace with new
                this.roomAssets[roomId][assetKey] = texture;
              }
              
              // Trigger callbacks
              const callbacks = this.hotReloadCallbacks.get(assetKey);
              if (callbacks) {
                callbacks.forEach((cb) => cb(texture));
              }
              
              resolve(texture);
            },
            undefined,
            (error) => {
              console.error(`Failed to reload texture ${assetKey}`, error);
              resolve(null);
            },
          );
        });
      }
    }
    
    return null;
  }

  async reloadCurrentRoom() {
    const roomId = this.currentRoom;
    console.log(`[ResourceLoader] Reloading room: ${roomId}`);
    
    // Dispose current assets
    if (this.roomAssets[roomId]) {
      Object.values(this.roomAssets[roomId]).forEach((asset) => {
        this._disposeAsset(asset);
      });
    }
    
    // Clear cache
    delete this.roomAssets[roomId];
    delete this.loadingPromises[roomId];
    
    // Reload
    const assets = await this.loadRoom(roomId);
    
    // Notify all callbacks
    this.hotReloadCallbacks.forEach((callbacks, key) => {
      if (assets[key]) {
        callbacks.forEach((cb) => cb(assets[key]));
      }
    });
    
    console.log(`[ResourceLoader] Room ${roomId} reloaded successfully`);
    return assets;
  }

  onAssetReload(assetKey, callback) {
    if (!this.hotReloadCallbacks.has(assetKey)) {
      this.hotReloadCallbacks.set(assetKey, new Set());
    }
    this.hotReloadCallbacks.get(assetKey).add(callback);
    
    // Return unsubscribe function
    return () => {
      const callbacks = this.hotReloadCallbacks.get(assetKey);
      if (callbacks) {
        callbacks.delete(callback);
        if (callbacks.size === 0) {
          this.hotReloadCallbacks.delete(assetKey);
        }
      }
    };
  }

  _disposeAsset(asset) {
    if (!asset) return;
    
    if (asset instanceof THREE.Texture) {
      asset.dispose?.();
    } else if (asset.scene) {
      // GLTF asset
      asset.scene.traverse((child) => {
        if (child.geometry) child.geometry.dispose();
        if (child.material) {
          if (Array.isArray(child.material)) {
            child.material.forEach((mat) => mat.dispose());
          } else {
            child.material.dispose();
          }
        }
      });
    }
  }

  _createFallbackRoom() {
    const group = new THREE.Group();

    const floor = new THREE.Mesh(
      new THREE.BoxGeometry(6.5, 0.2, 6.5),
      new THREE.MeshStandardMaterial({
        color: "#7d8bff",
        roughness: 0.8,
      }),
    );
    floor.position.y = -1.1;
    group.add(floor);

    const wallMaterial = new THREE.MeshStandardMaterial({
      color: "#cdd7ff",
      roughness: 0.6,
    });
    const backWall = new THREE.Mesh(
      new THREE.PlaneGeometry(6.5, 3.5),
      wallMaterial.clone(),
    );
    backWall.position.set(0, 0.65, -3.2);
    group.add(backWall);

    const sideWall = new THREE.Mesh(
      new THREE.PlaneGeometry(6.5, 3.5),
      wallMaterial.clone(),
    );
    sideWall.rotation.y = Math.PI / 2;
    sideWall.position.set(-3.2, 0.65, 0);
    group.add(sideWall);

    return group;
  }

  _createFallbackAccessories() {
    const group = new THREE.Group();

    const bed = new THREE.Mesh(
      new THREE.BoxGeometry(2.4, 0.4, 1.6),
      new THREE.MeshStandardMaterial({ color: "#fef3c7", roughness: 0.4 }),
    );
    bed.position.set(-1.6, -0.8, -1.2);
    group.add(bed);

    const bookshelf = new THREE.Mesh(
      new THREE.BoxGeometry(0.6, 2.4, 1.2),
      new THREE.MeshStandardMaterial({ color: "#facc15", roughness: 0.5 }),
    );
    bookshelf.position.set(2.4, -0.3, -1.6);
    group.add(bookshelf);

    return group;
  }

  destroy() {
    // Clean up hot reload listeners
    if (typeof window !== "undefined") {
      window.removeEventListener("keydown", this._handleHotkey.bind(this));
    }
    
    // Dispose all assets
    Object.values(this.roomAssets).forEach((roomBucket) => {
      Object.values(roomBucket).forEach((asset) => {
        this._disposeAsset(asset);
      });
    });

    this.roomAssets = {};
    this.loadingPromises = {};
    this.hotReloadCallbacks.clear();
    this.dracoLoader.dispose?.();
  }
}
