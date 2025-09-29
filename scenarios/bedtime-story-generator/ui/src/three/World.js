import * as THREE from "three";

const TIME_OF_DAY_LOOKUPS = {
  day: {
    ground: "#fff6f0",
    glow: "#4f6dff",
    ambient: "#ffffff",
    directional: "#ffe6d5",
    fog: "#bce3ff",
    fogDensity: 0.01,
  },
  evening: {
    ground: "#fbe9dd",
    glow: "#ff9f85",
    ambient: "#ffa366",
    directional: "#ff8c42",
    fog: "#ffbb9a",
    fogDensity: 0.015,
  },
  night: {
    ground: "#1f2145",
    glow: "#8894ff",
    ambient: "#4a5099",
    directional: "#6677ff",
    fog: "#0f112a",
    fogDensity: 0.02,
  },
};

const DEFAULT_ROOM = "studio";

export default class World {
  constructor({ experience, initialState }) {
    this.experience = experience;
    this.scene = experience.scene;

    this.activeRoom = initialState?.activeRoom || DEFAULT_ROOM;

    this.currentGlow = new THREE.Color("#4f6dff");
    this.targetGlow = new THREE.Color("#4f6dff");
    this.currentGround = new THREE.Color("#fff6f0");
    this.targetGround = new THREE.Color("#fff6f0");
    this.currentAmbient = new THREE.Color("#ffffff");
    this.targetAmbient = new THREE.Color("#ffffff");
    this.currentDirectional = new THREE.Color("#ffe6d5");
    this.targetDirectional = new THREE.Color("#ffe6d5");
    this.rotationSpeed = 0.00025;

    this.roomGroups = {
      studio: new THREE.Group(),
      bedroom: new THREE.Group(),
    };

    this.environmentMaps = {};

    this.roomGroups.studio.visible = this.activeRoom === "studio";
    this.roomGroups.bedroom.visible = this.activeRoom === "bedroom";

    this.scene.add(this.roomGroups.studio);
    this.scene.add(this.roomGroups.bedroom);

    this._createLighting();
    this._setupFog();
    this._buildStudioFallbacks();

    if (initialState) {
      this.setStoreSnapshot(initialState);
    }

    this.setActiveRoom(this.activeRoom);
  }

  applyRoomAssets(roomId, assets = {}) {
    if (roomId === "studio") {
      this._applyStudioAssets(assets);
    } else if (roomId === "bedroom") {
      this._applyBedroomAssets(assets);
    }

    if (assets["hdr_day"]) {
      this.environmentMaps[roomId] = assets["hdr_day"];
      if (roomId === this.activeRoom) {
        this.scene.environment = assets["hdr_day"];
      }
    }
  }

  setActiveRoom(roomId = DEFAULT_ROOM) {
    const nextRoom = roomId || DEFAULT_ROOM;

    if (this.activeRoom !== nextRoom) {
      this.activeRoom = nextRoom;
      this.roomGroups.studio.visible = nextRoom === "studio";
      this.roomGroups.bedroom.visible = nextRoom === "bedroom";
    }

    const env = this.environmentMaps[this.activeRoom];
    this.scene.environment = env || null;
  }

  setStoreSnapshot(snapshot) {
    this.storeSnapshot = snapshot;
    if (snapshot.activeRoom && snapshot.activeRoom !== this.activeRoom) {
      this.setActiveRoom(snapshot.activeRoom);
    }

    const look =
      TIME_OF_DAY_LOOKUPS[snapshot.timeOfDay] || TIME_OF_DAY_LOOKUPS.day;
    this.targetGlow.set(look.glow);
    this.targetGround.set(look.ground);
    this.targetAmbient.set(look.ambient);
    this.targetDirectional.set(look.directional);

    if (this.scene.fog) {
      this.scene.fog.color.set(look.fog);
      this.scene.fog.density = look.fogDensity;
    }

    this.rotationSpeed = snapshot.selectedStory ? 0.00055 : 0.00025;

    if (this.placeholder?.material) {
      this.placeholder.material.emissiveIntensity = snapshot.selectedStory
        ? 0.6
        : 0.35;
    }

    if (this.bookshelf && snapshot.selectedStory) {
      this.bookshelf.children.forEach((child, index) => {
        if (index > 0) {
          child.material.emissive = new THREE.Color(look.glow);
          child.material.emissiveIntensity = 0.1;
        }
      });
    }

    if (this.glassMaterial) {
      const windowColors = {
        day: "#87ceeb",
        evening: "#ff8c42",
        night: "#1a1a3e",
      };
      this.glassMaterial.color.set(
        windowColors[snapshot.timeOfDay] || windowColors.day,
      );
    }

    if (this.starsMaterial) {
      this.starsMaterial.opacity = snapshot.timeOfDay === "night" ? 0.8 : 0;
    }

    if (this.storySpotlight) {
      this.storySpotlight.intensity = snapshot.selectedStory ? 1.0 : 0.3;
    }

    if (this.displayMaterial) {
      this.displayMaterial.opacity = snapshot.selectedStory ? 0.8 : 0.3;
      if (snapshot.selectedStory) {
        this.displayMaterial.color.set("#2a4858");
      }
    }

    // Update story projector
    if (this.projectorScreen && this.projectorLight && this.projectorLens) {
      const hasStory = !!snapshot.selectedStory;
      this.projectorLight.intensity = hasStory ? 0.5 : 0;
      this.projectorScreen.material.emissiveIntensity = hasStory ? 0.1 : 0;
      this.projectorLens.material.emissiveIntensity = hasStory ? 0.5 : 0;
      
      // Update canvas content
      this._updateStoryCanvas(snapshot.selectedStory);
    }
  }

  update({ elapsed, delta }) {
    if (this.activeRoom === "studio") {
      this._updateStudio({ elapsed, delta });
    } else if (this.activeRoom === "bedroom") {
      this._updateBedroom({ elapsed, delta });
    }
  }

  _createLighting() {
    this.rimLight = new THREE.DirectionalLight("#ffe6d5", 0.6);
    this.rimLight.position.set(-4, 6, -4);
    this.rimLight.castShadow = true;
    this.rimLight.shadow.camera.near = 0.1;
    this.rimLight.shadow.camera.far = 20;
    this.rimLight.shadow.mapSize.width = 2048;
    this.rimLight.shadow.mapSize.height = 2048;
    this.scene.add(this.rimLight);

    this.keyLight = new THREE.DirectionalLight("#ffffff", 0.75);
    this.keyLight.position.set(6, 8, 6);
    this.keyLight.castShadow = true;
    this.keyLight.shadow.camera.near = 0.1;
    this.keyLight.shadow.camera.far = 20;
    this.keyLight.shadow.mapSize.width = 2048;
    this.keyLight.shadow.mapSize.height = 2048;
    this.scene.add(this.keyLight);

    this.fillLight = new THREE.HemisphereLight("#ffe5c4", "#1b1e3f", 0.8);
    this.scene.add(this.fillLight);
  }

  _setupFog() {
    this.scene.fog = new THREE.FogExp2("#bce3ff", 0.01);
  }

  _buildStudioFallbacks() {
    this._createGround();
    this._createPlaceholder();
    this._createParticles();
    this._createBookshelf();
    this._createWindow();
    this._createStoryStage();
    this._createReadingLamp();
    this._createToyChest();
    this._createStoryProjector();
    this._createVolumetricLighting();
  }

  _applyStudioAssets(assets) {
    if (this.studioRoom) {
      this.roomGroups.studio.remove(this.studioRoom);
      this._disposeObject(this.studioRoom);
      this.studioRoom = null;
    }

    if (this.studioAccessories) {
      this.roomGroups.studio.remove(this.studioAccessories);
      this._disposeObject(this.studioAccessories);
      this.studioAccessories = null;
    }

    const roomSource = assets.model_room;
    if (roomSource?.scene) {
      this.studioRoom = roomSource.scene.clone(true);
      this.studioRoom.position.y -= 1.1;
      this.roomGroups.studio.add(this.studioRoom);
    } else if (roomSource instanceof THREE.Group) {
      this.studioRoom = roomSource.clone(true);
      this.studioRoom.position.y -= 1.1;
      this.roomGroups.studio.add(this.studioRoom);
    }

    const accessoriesSource = assets.model_accessories;
    if (accessoriesSource?.scene) {
      this.studioAccessories = accessoriesSource.scene.clone(true);
      this.studioAccessories.position.y -= 1.1;
      this.roomGroups.studio.add(this.studioAccessories);
    } else if (accessoriesSource instanceof THREE.Group) {
      this.studioAccessories = accessoriesSource.clone(true);
      this.roomGroups.studio.add(this.studioAccessories);
    }

    if (this.placeholder && this.studioRoom) {
      this.roomGroups.studio.remove(this.placeholder);
      this._disposeObject(this.placeholder, { retainGeometry: true });
      this.placeholder = null;
    }
  }

  _applyBedroomAssets(assets) {
    if (this.bedroomRoom) {
      this.roomGroups.bedroom.remove(this.bedroomRoom);
      this._disposeObject(this.bedroomRoom);
      this.bedroomRoom = null;
    }

    const environmentSource = assets.model_environment || assets.model_room;
    if (environmentSource?.scene) {
      this.bedroomRoom = environmentSource.scene.clone(true);
    } else if (environmentSource instanceof THREE.Group) {
      this.bedroomRoom = environmentSource.clone(true);
    }

    if (this.bedroomRoom) {
      const bbox = new THREE.Box3().setFromObject(this.bedroomRoom);
      const heightOffset = -bbox.min.y;
      this.bedroomRoom.position.y += heightOffset - 1.1;

      const convertMaterial = (material) => {
        if (!material) {
          return material;
        }

        const map = material.map || null;
        if (map) {
          map.colorSpace = THREE.SRGBColorSpace;
        }

        const basic = new THREE.MeshBasicMaterial({
          map,
          color:
            (material.color && material.color.clone()) ||
            new THREE.Color(0xffffff),
          transparent: material.transparent || false,
          opacity: material.opacity ?? 1,
        });
        basic.toneMapped = true;
        basic.side = material.side ?? THREE.FrontSide;

        if (material.map) {
          material.map = null;
        }
        material.dispose?.();

        return basic;
      };

      this.bedroomRoom.traverse((child) => {
        if (child.isMesh) {
          child.castShadow = false;
          child.receiveShadow = true;

          if (Array.isArray(child.material)) {
            child.material = child.material.map((mat) => convertMaterial(mat));
          } else {
            child.material = convertMaterial(child.material);
          }
        }
      });

      this.roomGroups.bedroom.add(this.bedroomRoom);
    }
  }

  _createGround() {
    const geometry = new THREE.CircleGeometry(10, 48);
    const material = new THREE.MeshStandardMaterial({
      color: "#1e2146",
      roughness: 0.8,
      metalness: 0.1,
    });

    this.ground = new THREE.Mesh(geometry, material);
    this.ground.rotation.x = -Math.PI / 2;
    this.ground.position.y = -1.2;
    this.ground.receiveShadow = true;
    this.roomGroups.studio.add(this.ground);
  }

  _createPlaceholder() {
    const geometry = new THREE.TorusKnotGeometry(1.2, 0.35, 120, 18);
    const material = new THREE.MeshStandardMaterial({
      color: "#a36bff",
      roughness: 0.3,
      metalness: 0.25,
      emissive: "#4f6dff",
      emissiveIntensity: 0.4,
    });

    this.placeholder = new THREE.Mesh(geometry, material);
    this.placeholder.position.set(0, 1.4, 0);
    this.placeholder.castShadow = true;
    this.roomGroups.studio.add(this.placeholder);
  }

  _createParticles() {
    const particleGeometry = new THREE.BufferGeometry();
    const particleCount = 100;
    const positions = new Float32Array(particleCount * 3);
    const colors = new Float32Array(particleCount * 3);

    for (let i = 0; i < particleCount * 3; i += 3) {
      positions[i] = (Math.random() - 0.5) * 10;
      positions[i + 1] = Math.random() * 5;
      positions[i + 2] = (Math.random() - 0.5) * 10;

      colors[i] = 0.8 + Math.random() * 0.2;
      colors[i + 1] = 0.6 + Math.random() * 0.3;
      colors[i + 2] = 0.9 + Math.random() * 0.1;
    }

    particleGeometry.setAttribute(
      "position",
      new THREE.BufferAttribute(positions, 3),
    );
    particleGeometry.setAttribute(
      "color",
      new THREE.BufferAttribute(colors, 3),
    );

    const particleMaterial = new THREE.PointsMaterial({
      size: 0.05,
      vertexColors: true,
      transparent: true,
      opacity: 0.6,
      blending: THREE.AdditiveBlending,
    });

    this.particleSystem = new THREE.Points(
      particleGeometry,
      particleMaterial,
    );
    this.roomGroups.studio.add(this.particleSystem);
  }

  _createBookshelf() {
    const shelfGroup = new THREE.Group();

    const shelfGeometry = new THREE.BoxGeometry(3, 4, 0.5);
    const shelfMaterial = new THREE.MeshStandardMaterial({
      color: "#8b4513",
      roughness: 0.8,
      metalness: 0.1,
    });
    const shelf = new THREE.Mesh(shelfGeometry, shelfMaterial);
    shelf.position.set(-3, 2, -2);
    shelf.castShadow = true;
    shelf.receiveShadow = true;
    shelfGroup.add(shelf);

    for (let level = 0; level < 3; level++) {
      const levelGeometry = new THREE.BoxGeometry(2.8, 0.1, 0.4);
      const levelMesh = new THREE.Mesh(levelGeometry, shelfMaterial);
      levelMesh.position.set(-3, 0.5 + level * 1.2, -2);
      levelMesh.castShadow = true;
      levelMesh.receiveShadow = true;
      shelfGroup.add(levelMesh);
    }

    const bookColors = [
      "#ff6b6b",
      "#4ecdc4",
      "#45b7d1",
      "#f9ca24",
      "#6c5ce7",
    ];
    this.books = [];
    for (let level = 0; level < 3; level++) {
      for (let i = 0; i < 5; i++) {
        const bookHeight = 0.8 + Math.random() * 0.4;
        const bookGeometry = new THREE.BoxGeometry(0.2, bookHeight, 0.6);
        const bookMaterial = new THREE.MeshStandardMaterial({
          color: bookColors[i % bookColors.length],
          roughness: 0.7,
          metalness: 0.1,
        });
        const book = new THREE.Mesh(bookGeometry, bookMaterial);
        book.position.set(-3.8 + i * 0.3, 0.9 + level * 1.2, -1.8);
        book.rotation.z = Math.random() * 0.1 - 0.05;
        book.castShadow = true;
        this.books.push(book);
        shelfGroup.add(book);
      }
    }

    this.bookshelf = shelfGroup;
    this.roomGroups.studio.add(this.bookshelf);
  }

  _createWindow() {
    const windowGroup = new THREE.Group();

    const frameGeometry = new THREE.BoxGeometry(2.5, 3, 0.2);
    const frameMaterial = new THREE.MeshStandardMaterial({
      color: "#5a4a3a",
      roughness: 0.8,
      metalness: 0.1,
    });
    const frame = new THREE.Mesh(frameGeometry, frameMaterial);
    frame.position.set(4, 2.5, -3);
    frame.castShadow = true;
    windowGroup.add(frame);

    const glassGeometry = new THREE.PlaneGeometry(2.2, 2.7);
    this.glassMaterial = new THREE.MeshBasicMaterial({
      transparent: true,
      opacity: 0.8,
      color: "#87ceeb",
    });
    const glass = new THREE.Mesh(glassGeometry, this.glassMaterial);
    glass.position.set(4, 2.5, -2.89);
    windowGroup.add(glass);

    const starsGeometry = new THREE.BufferGeometry();
    const starsCount = 50;
    const starsPositions = new Float32Array(starsCount * 3);
    for (let i = 0; i < starsCount * 3; i += 3) {
      starsPositions[i] = 3.5 + Math.random() * 1;
      starsPositions[i + 1] = 2 + Math.random() * 2;
      starsPositions[i + 2] = -2.88;
    }

    starsGeometry.setAttribute(
      "position",
      new THREE.BufferAttribute(starsPositions, 3),
    );

    this.starsMaterial = new THREE.PointsMaterial({
      size: 0.02,
      color: "#ffffff",
      transparent: true,
      opacity: 0,
    });

    this.stars = new THREE.Points(starsGeometry, this.starsMaterial);
    windowGroup.add(this.stars);

    this.window = windowGroup;
    this.roomGroups.studio.add(this.window);
  }

  _createStoryStage() {
    const stageGroup = new THREE.Group();

    const platformGeometry = new THREE.CylinderGeometry(2, 2.2, 0.3, 8);
    const platformMaterial = new THREE.MeshStandardMaterial({
      color: "#8a6b47",
      roughness: 0.7,
      metalness: 0.2,
    });
    const platform = new THREE.Mesh(platformGeometry, platformMaterial);
    platform.position.set(0, -0.9, 2);
    platform.castShadow = true;
    platform.receiveShadow = true;
    stageGroup.add(platform);

    const displayGeometry = new THREE.PlaneGeometry(3, 2);
    this.displayMaterial = new THREE.MeshBasicMaterial({
      color: "#1a1a2e",
      transparent: true,
      opacity: 0.3,
    });
    const display = new THREE.Mesh(displayGeometry, this.displayMaterial);
    display.position.set(0, 1, 2.1);
    stageGroup.add(display);

    this.storySpotlight = new THREE.SpotLight("#ffffff", 0.5);
    this.storySpotlight.position.set(0, 4, 3);
    this.storySpotlight.target.position.set(0, 0, 2);
    this.storySpotlight.angle = Math.PI / 6;
    this.storySpotlight.penumbra = 0.3;
    this.storySpotlight.castShadow = true;
    stageGroup.add(this.storySpotlight);
    stageGroup.add(this.storySpotlight.target);

    this.storyStage = stageGroup;
    this.roomGroups.studio.add(this.storyStage);
  }

  _createReadingLamp() {
    const lampGroup = new THREE.Group();
    
    // Lamp base
    const baseGeometry = new THREE.CylinderGeometry(0.3, 0.35, 0.2, 12);
    const baseMaterial = new THREE.MeshStandardMaterial({
      color: "#2a2a2a",
      metalness: 0.8,
      roughness: 0.2,
    });
    const base = new THREE.Mesh(baseGeometry, baseMaterial);
    lampGroup.add(base);
    
    // Lamp arm
    const armGeometry = new THREE.CylinderGeometry(0.03, 0.03, 1.5, 8);
    const armMaterial = new THREE.MeshStandardMaterial({
      color: "#3a3a3a",
      metalness: 0.7,
      roughness: 0.3,
    });
    const arm = new THREE.Mesh(armGeometry, armMaterial);
    arm.position.y = 0.85;
    arm.rotation.z = Math.PI / 6;
    lampGroup.add(arm);
    
    // Lamp shade
    const shadeGeometry = new THREE.ConeGeometry(0.4, 0.5, 12, 1, true);
    const shadeMaterial = new THREE.MeshStandardMaterial({
      color: "#f5e6d3",
      roughness: 0.8,
      metalness: 0.1,
      side: THREE.DoubleSide,
    });
    const shade = new THREE.Mesh(shadeGeometry, shadeMaterial);
    shade.position.y = 1.5;
    shade.rotation.z = -Math.PI / 6;
    lampGroup.add(shade);
    
    // Reading light (spotlight)
    this.readingLight = new THREE.SpotLight("#ffe4b5", 0.8, 8, Math.PI / 4, 0.5);
    this.readingLight.position.set(0, 1.5, 0);
    this.readingLight.target.position.set(0, -2, 1);
    this.readingLight.castShadow = true;
    this.readingLight.shadow.mapSize.width = 1024;
    this.readingLight.shadow.mapSize.height = 1024;
    lampGroup.add(this.readingLight);
    lampGroup.add(this.readingLight.target);
    
    // Interactive emissive glow when on
    this.lampShade = shade;
    this.lampOn = false;
    
    lampGroup.position.set(3, 0.1, -2);
    this.readingLamp = lampGroup;
    this.roomGroups.studio.add(this.readingLamp);
  }

  _createToyChest() {
    const chestGroup = new THREE.Group();
    
    // Chest body
    const bodyGeometry = new THREE.BoxGeometry(1.8, 0.9, 1.2);
    const bodyMaterial = new THREE.MeshStandardMaterial({
      color: "#8b4513",
      roughness: 0.7,
      metalness: 0.1,
    });
    const body = new THREE.Mesh(bodyGeometry, bodyMaterial);
    body.position.y = 0.45;
    chestGroup.add(body);
    
    // Chest lid (animated)
    const lidGeometry = new THREE.BoxGeometry(1.85, 0.15, 1.25);
    const lidMaterial = new THREE.MeshStandardMaterial({
      color: "#a0522d",
      roughness: 0.6,
      metalness: 0.1,
    });
    this.chestLid = new THREE.Mesh(lidGeometry, lidMaterial);
    this.chestLid.position.set(0, 0.975, -0.6);
    this.chestLid.geometry.translate(0, 0, 0.6);
    chestGroup.add(this.chestLid);
    
    // Metal hinges
    const hingeGeometry = new THREE.CylinderGeometry(0.05, 0.05, 0.2, 8);
    const hingeMaterial = new THREE.MeshStandardMaterial({
      color: "#444444",
      metalness: 0.9,
      roughness: 0.3,
    });
    const hinge1 = new THREE.Mesh(hingeGeometry, hingeMaterial);
    hinge1.rotation.z = Math.PI / 2;
    hinge1.position.set(-0.7, 0.9, -0.6);
    chestGroup.add(hinge1);
    
    const hinge2 = new THREE.Mesh(hingeGeometry, hingeMaterial);
    hinge2.rotation.z = Math.PI / 2;
    hinge2.position.set(0.7, 0.9, -0.6);
    chestGroup.add(hinge2);
    
    // Toy blocks inside (visible when open)
    this.toyBlocks = [];
    const blockColors = ["#ff6b6b", "#4ecdc4", "#45b7d1", "#f9ca24", "#6c5ce7"];
    for (let i = 0; i < 5; i++) {
      const blockGeometry = new THREE.BoxGeometry(0.2, 0.2, 0.2);
      const blockMaterial = new THREE.MeshStandardMaterial({
        color: blockColors[i],
        roughness: 0.3,
        metalness: 0.1,
      });
      const block = new THREE.Mesh(blockGeometry, blockMaterial);
      block.position.set(
        (Math.random() - 0.5) * 0.8,
        0.3 + Math.random() * 0.3,
        (Math.random() - 0.5) * 0.4
      );
      block.rotation.set(
        Math.random() * Math.PI,
        Math.random() * Math.PI,
        Math.random() * Math.PI
      );
      block.visible = false;
      this.toyBlocks.push(block);
      chestGroup.add(block);
    }
    
    // Initial state
    this.chestOpen = false;
    this.chestOpenTarget = 0;
    
    chestGroup.position.set(-3.5, 0, 2);
    this.toyChest = chestGroup;
    this.roomGroups.studio.add(this.toyChest);
  }

  _createStoryProjector() {
    const projectorGroup = new THREE.Group();
    
    // Projector screen (wall-mounted)
    const screenGeometry = new THREE.PlaneGeometry(4, 2.5);
    const screenMaterial = new THREE.MeshStandardMaterial({
      color: "#1a1a1a",
      roughness: 0.9,
      metalness: 0.1,
      emissive: "#ffffff",
      emissiveIntensity: 0,
    });
    this.projectorScreen = new THREE.Mesh(screenGeometry, screenMaterial);
    this.projectorScreen.position.set(0, 2.5, -4.9);
    projectorGroup.add(this.projectorScreen);
    
    // Screen frame
    const frameThickness = 0.1;
    const frameMaterial = new THREE.MeshStandardMaterial({
      color: "#333333",
      roughness: 0.3,
      metalness: 0.7,
    });
    
    // Top frame
    const topFrame = new THREE.Mesh(
      new THREE.BoxGeometry(4.2, frameThickness, 0.1),
      frameMaterial
    );
    topFrame.position.set(0, 3.8, -4.85);
    projectorGroup.add(topFrame);
    
    // Bottom frame
    const bottomFrame = new THREE.Mesh(
      new THREE.BoxGeometry(4.2, frameThickness, 0.1),
      frameMaterial
    );
    bottomFrame.position.set(0, 1.2, -4.85);
    projectorGroup.add(bottomFrame);
    
    // Side frames
    const sideFrame1 = new THREE.Mesh(
      new THREE.BoxGeometry(frameThickness, 2.7, 0.1),
      frameMaterial
    );
    sideFrame1.position.set(-2.05, 2.5, -4.85);
    projectorGroup.add(sideFrame1);
    
    const sideFrame2 = new THREE.Mesh(
      new THREE.BoxGeometry(frameThickness, 2.7, 0.1),
      frameMaterial
    );
    sideFrame2.position.set(2.05, 2.5, -4.85);
    projectorGroup.add(sideFrame2);
    
    // Projector unit (ceiling mounted)
    const projectorGeometry = new THREE.CylinderGeometry(0.2, 0.15, 0.4, 8);
    const projectorMaterial = new THREE.MeshStandardMaterial({
      color: "#2a2a2a",
      roughness: 0.2,
      metalness: 0.8,
    });
    const projector = new THREE.Mesh(projectorGeometry, projectorMaterial);
    projector.position.set(0, 4.5, 2);
    projector.rotation.x = Math.PI / 2;
    projectorGroup.add(projector);
    
    // Projector lens
    const lensGeometry = new THREE.CylinderGeometry(0.12, 0.12, 0.05, 16);
    const lensMaterial = new THREE.MeshStandardMaterial({
      color: "#666666",
      roughness: 0.1,
      metalness: 0.9,
      emissive: "#4488ff",
      emissiveIntensity: 0,
    });
    this.projectorLens = new THREE.Mesh(lensGeometry, lensMaterial);
    this.projectorLens.position.set(0, 4.5, 1.8);
    this.projectorLens.rotation.x = Math.PI / 2;
    projectorGroup.add(this.projectorLens);
    
    // Projector light beam (when active)
    this.projectorLight = new THREE.SpotLight("#88aaff", 0, 8, Math.PI / 8, 0.8);
    this.projectorLight.position.set(0, 4.5, 1.8);
    this.projectorLight.target.position.set(0, 2.5, -4.9);
    projectorGroup.add(this.projectorLight);
    projectorGroup.add(this.projectorLight.target);
    
    // Story text canvas (dynamic content)
    const canvas = document.createElement("canvas");
    canvas.width = 1024;
    canvas.height = 640;
    this.storyCanvas = canvas;
    this.storyContext = canvas.getContext("2d");
    
    // Initialize canvas
    this.storyContext.fillStyle = "#000000";
    this.storyContext.fillRect(0, 0, canvas.width, canvas.height);
    
    const canvasTexture = new THREE.CanvasTexture(canvas);
    canvasTexture.minFilter = THREE.LinearFilter;
    canvasTexture.magFilter = THREE.LinearFilter;
    this.storyTexture = canvasTexture;
    
    // Apply texture to screen
    this.projectorScreen.material.map = this.storyTexture;
    this.projectorScreen.material.needsUpdate = true;
    
    this.storyProjector = projectorGroup;
    this.roomGroups.studio.add(this.storyProjector);
  }

  _updateStoryCanvas(story) {
    if (!this.storyContext || !this.storyTexture) return;
    
    const ctx = this.storyContext;
    const canvas = this.storyCanvas;
    
    // Clear canvas
    ctx.fillStyle = story ? "#0a1929" : "#000000";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    
    if (story) {
      // Add gradient background
      const gradient = ctx.createLinearGradient(0, 0, 0, canvas.height);
      gradient.addColorStop(0, "#0a1929");
      gradient.addColorStop(1, "#1a2332");
      ctx.fillStyle = gradient;
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      
      // Draw title
      ctx.fillStyle = "#ffffff";
      ctx.font = "bold 48px Arial";
      ctx.textAlign = "center";
      ctx.fillText(story.title || "Story Time", canvas.width / 2, 80);
      
      // Draw theme badge
      if (story.theme) {
        ctx.fillStyle = "#4a90e2";
        ctx.roundRect(canvas.width / 2 - 80, 110, 160, 35, 15);
        ctx.fill();
        ctx.fillStyle = "#ffffff";
        ctx.font = "20px Arial";
        ctx.fillText(story.theme, canvas.width / 2, 135);
      }
      
      // Draw story excerpt
      if (story.content) {
        ctx.fillStyle = "#e0e0e0";
        ctx.font = "24px Arial";
        ctx.textAlign = "left";
        const words = story.content.split(" ").slice(0, 50);
        const lines = [];
        let currentLine = "";
        
        words.forEach(word => {
          const testLine = currentLine + word + " ";
          const metrics = ctx.measureText(testLine);
          if (metrics.width > canvas.width - 100 && currentLine) {
            lines.push(currentLine);
            currentLine = word + " ";
          } else {
            currentLine = testLine;
          }
        });
        if (currentLine) lines.push(currentLine);
        
        lines.slice(0, 8).forEach((line, i) => {
          ctx.fillText(line, 50, 220 + i * 35);
        });
      }
      
      // Draw reading time
      if (story.reading_time_minutes) {
        ctx.fillStyle = "#66bb6a";
        ctx.font = "20px Arial";
        ctx.textAlign = "right";
        ctx.fillText(`${story.reading_time_minutes} min read`, canvas.width - 50, canvas.height - 30);
      }
      
      // Add decorative elements
      ctx.strokeStyle = "#4a90e2";
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.moveTo(50, 160);
      ctx.lineTo(canvas.width - 50, 160);
      ctx.stroke();
    }
    
    this.storyTexture.needsUpdate = true;
  }

  _updateStudio({ elapsed, delta }) {
    if (this.activeRoom !== "studio") {
      return;
    }

    if (this.placeholder) {
      this.placeholder.rotation.y += this.rotationSpeed * delta;
      this.placeholder.rotation.x = Math.sin(elapsed * 0.0002) * 0.3;

      this.currentGlow.lerp(this.targetGlow, 0.05);
      this.placeholder.material.emissive.copy(this.currentGlow);
      this.placeholder.material.color.lerp(
        this.currentGlow.clone().offsetHSL(0, -0.3, 0),
        0.03,
      );
    }

    if (this.ground) {
      this.currentGround.lerp(this.targetGround, 0.04);
      this.ground.material.color.copy(this.currentGround);
    }

    if (this.rimLight) {
      this.currentDirectional.lerp(this.targetDirectional, 0.04);
      this.rimLight.color.copy(this.currentDirectional);
    }

    if (this.fillLight) {
      this.currentAmbient.lerp(this.targetAmbient, 0.04);
      this.fillLight.color.copy(this.currentAmbient);
    }

    if (this.particleSystem) {
      this.particleSystem.rotation.y = elapsed * 0.00005;
      const positions = this.particleSystem.geometry.attributes.position.array;
      for (let i = 1; i < positions.length; i += 3) {
        positions[i] += Math.sin(elapsed * 0.001 + i) * 0.001;
      }
      this.particleSystem.geometry.attributes.position.needsUpdate = true;
    }

    if (this.books) {
      this.books.forEach((book, index) => {
        const baseY = 0.9 + Math.floor(index / 5) * 1.2;
        book.position.y = baseY + Math.sin(elapsed * 0.0002 + index) * 0.01;
      });
    }

    if (this.stars && this.starsMaterial.opacity > 0) {
      const twinkle = Math.sin(elapsed * 0.002) * 0.5 + 0.5;
      this.starsMaterial.opacity = 0.3 + twinkle * 0.5;
    }

    if (this.storySpotlight && this.storySpotlight.intensity > 0.5) {
      this.storySpotlight.intensity = 0.8 + Math.sin(elapsed * 0.001) * 0.2;
    }

    // Update reading lamp
    if (this.readingLamp && this.readingLight) {
      // Toggle lamp based on story selection
      const shouldBeOn = !!this.storeSnapshot?.selectedStory;
      if (shouldBeOn !== this.lampOn) {
        this.lampOn = shouldBeOn;
        this.readingLight.intensity = this.lampOn ? 0.8 : 0;
        if (this.lampShade) {
          this.lampShade.material.emissive = new THREE.Color(this.lampOn ? "#ffcc66" : "#000000");
          this.lampShade.material.emissiveIntensity = this.lampOn ? 0.3 : 0;
        }
      }
      
      // Subtle animation when on
      if (this.lampOn) {
        this.readingLight.intensity = 0.7 + Math.sin(elapsed * 0.002) * 0.1;
      }
    }

    // Update toy chest
    if (this.toyChest && this.chestLid) {
      // Open chest when in night mode or when story is playing
      const shouldOpen = this.storeSnapshot?.timeOfDay === "night" || 
                        !!this.storeSnapshot?.selectedStory;
      this.chestOpenTarget = shouldOpen ? 1 : 0;
      
      // Smooth lid animation
      const currentAngle = this.chestLid.rotation.x;
      const targetAngle = -this.chestOpenTarget * Math.PI * 0.6;
      this.chestLid.rotation.x = THREE.MathUtils.lerp(currentAngle, targetAngle, 0.05);
      
      // Show/hide toys based on lid state
      const isOpen = Math.abs(this.chestLid.rotation.x) > 0.3;
      this.toyBlocks?.forEach((block, i) => {
        block.visible = isOpen;
        if (isOpen) {
          // Float toys when chest is open
          block.position.y = 0.3 + Math.sin(elapsed * 0.001 + i) * 0.05;
          block.rotation.y = elapsed * 0.0005 * (i + 1);
        }
      });
    }
    
    // Animate dust particles for cinematic atmosphere
    if (this.dustParticles) {
      const positions = this.dustParticles.geometry.attributes.position.array;
      for (let i = 0; i < positions.length; i += 3) {
        // Gentle floating motion
        positions[i] += Math.sin(elapsed * 0.0001 + i) * 0.001;
        positions[i + 1] += Math.sin(elapsed * 0.0002 + i) * 0.002;
        positions[i + 2] += Math.cos(elapsed * 0.0001 + i) * 0.001;
        
        // Reset particles that float too high
        if (positions[i + 1] > 5) {
          positions[i + 1] = 0;
        }
      }
      this.dustParticles.geometry.attributes.position.needsUpdate = true;
      
      // Rotate the entire dust system slowly
      this.dustParticles.rotation.y = elapsed * 0.00002;
    }
    
    // Animate volumetric lighting
    if (this.volumetricLighting) {
      this.volumetricLighting.children.forEach((child, index) => {
        if (child.geometry && child.geometry.type === "ConeGeometry") {
          // Subtle pulsing for god rays
          const pulse = Math.sin(elapsed * 0.0005 + index * 0.5) * 0.02;
          child.material.opacity = 0.1 + pulse;
        }
      });
    }
  }

  _updateBedroom({ elapsed }) {
    if (this.activeRoom !== "bedroom" || !this.bedroomRoom) {
      return;
    }

    this.bedroomRoom.rotation.y = Math.sin(elapsed * 0.00005) * 0.02;
  }

  _disposeObject(object, { retainGeometry = false } = {}) {
    if (!object) return;

    if (object instanceof THREE.Group) {
      object.traverse((child) => {
        if (!retainGeometry && child.geometry) child.geometry.dispose();
        if (child.material) {
          if (Array.isArray(child.material)) {
            child.material.forEach((mat) => mat.dispose());
          } else {
            child.material.dispose();
          }
        }
      });
    } else {
      if (!retainGeometry) {
        object.geometry?.dispose?.();
      }
      if (object.material) {
        if (Array.isArray(object.material)) {
          object.material.forEach((mat) => mat.dispose());
        } else {
          object.material.dispose();
        }
      }
    }
  }

  
  _createVolumetricLighting() {
    // Add volumetric fog for cinematic depth
    const volumetricGroup = new THREE.Group();
    
    // God rays from window
    const rayGeometry = new THREE.ConeGeometry(2, 6, 4);
    const rayMaterial = new THREE.MeshBasicMaterial({
      color: "#fff8dc",
      transparent: true,
      opacity: 0.1,
      side: THREE.DoubleSide,
    });
    
    for (let i = 0; i < 3; i++) {
      const ray = new THREE.Mesh(rayGeometry, rayMaterial);
      ray.position.set(4 + i * 0.3, 2.5, -1);
      ray.rotation.z = Math.PI;
      ray.rotation.x = 0.3;
      volumetricGroup.add(ray);
    }
    
    // Ambient dust particles for atmosphere
    const dustGeometry = new THREE.BufferGeometry();
    const dustCount = 200;
    const dustPositions = new Float32Array(dustCount * 3);
    const dustSizes = new Float32Array(dustCount);
    
    for (let i = 0; i < dustCount; i++) {
      dustPositions[i * 3] = (Math.random() - 0.5) * 10;
      dustPositions[i * 3 + 1] = Math.random() * 5;
      dustPositions[i * 3 + 2] = (Math.random() - 0.5) * 8;
      dustSizes[i] = Math.random() * 0.02 + 0.005;
    }
    
    dustGeometry.setAttribute(
      "position",
      new THREE.BufferAttribute(dustPositions, 3)
    );
    dustGeometry.setAttribute(
      "size",
      new THREE.BufferAttribute(dustSizes, 1)
    );
    
    const dustMaterial = new THREE.PointsMaterial({
      size: 0.01,
      color: "#ffe4c4",
      transparent: true,
      opacity: 0.4,
      blending: THREE.AdditiveBlending,
    });
    
    this.dustParticles = new THREE.Points(dustGeometry, dustMaterial);
    volumetricGroup.add(this.dustParticles);
    
    this.volumetricLighting = volumetricGroup;
    this.roomGroups.studio.add(this.volumetricLighting);
  }

  destroy() {
    [this.studioRoom, this.studioAccessories, this.bedroomRoom].forEach(
      (object) => {
        if (!object) return;
        this._disposeObject(object);
      },
    );

    const studioExtras = [
      this.ground,
      this.placeholder,
      this.particleSystem,
      this.bookshelf,
      this.window,
      this.storyStage,
      this.volumetricLighting,
    ];

    studioExtras.forEach((object) => {
      if (!object) return;
      this.roomGroups.studio.remove(object);
      this._disposeObject(object);
    });

    Object.values(this.roomGroups).forEach((group) => {
      this.scene.remove(group);
    });

    [this.rimLight, this.keyLight, this.fillLight].forEach((light) => {
      if (light) this.scene.remove(light);
    });
  }
}
