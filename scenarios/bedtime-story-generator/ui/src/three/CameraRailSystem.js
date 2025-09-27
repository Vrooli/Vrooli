import * as THREE from "three";
import gsap from "gsap";
import useExperienceStore from "../state/store.js";

export default class CameraRailSystem {
  constructor({ camera, scene }) {
    this.camera = camera;
    this.scene = scene;
    
    // Camera rail paths for cinematic movements
    this.rails = {
      intro: {
        points: [
          new THREE.Vector3(10, 8, 10),
          new THREE.Vector3(8, 6, 8),
          new THREE.Vector3(6.4, 3.9, 6.3)
        ],
        targets: [
          new THREE.Vector3(0, 2, 0),
          new THREE.Vector3(0, 1.8, 0),
          new THREE.Vector3(0, 1.4, 0)
        ],
        duration: 3,
        ease: "power2.inOut"
      },
      bookshelfFocus: {
        points: [
          new THREE.Vector3(6.4, 3.9, 6.3),
          new THREE.Vector3(4, 3.2, 3.5),
          new THREE.Vector3(3.2, 2.9, 2.4)
        ],
        targets: [
          new THREE.Vector3(0, 1.4, 0),
          new THREE.Vector3(-1, 1.2, -1),
          new THREE.Vector3(-3.05, 0.2, -1.4)
        ],
        duration: 2,
        ease: "power2.inOut"
      },
      windowPan: {
        points: [
          new THREE.Vector3(3, 2.5, 3),
          new THREE.Vector3(2, 2.5, 2),
          new THREE.Vector3(0.4, 2.5, 1.1)
        ],
        targets: [
          new THREE.Vector3(0, 2, -2),
          new THREE.Vector3(0, 1.8, -3),
          new THREE.Vector3(0, 1.4, -4.0)
        ],
        duration: 2.5,
        ease: "power3.inOut"
      },
      storyOrbit: {
        points: [],
        targets: [],
        duration: 8,
        ease: "none"
      },
      lampZoom: {
        points: [
          new THREE.Vector3(6.4, 3.9, 6.3),
          new THREE.Vector3(2, 3, 2),
          new THREE.Vector3(1, 2.5, 1)
        ],
        targets: [
          new THREE.Vector3(0, 1.4, 0),
          new THREE.Vector3(1.5, 1.5, -1),
          new THREE.Vector3(2, 1.8, -2)
        ],
        duration: 2,
        ease: "power2.inOut"
      },
      toyChestReveal: {
        points: [
          new THREE.Vector3(6.4, 3.9, 6.3),
          new THREE.Vector3(3, 1.5, 3),
          new THREE.Vector3(2, 1, 2)
        ],
        targets: [
          new THREE.Vector3(0, 1.4, 0),
          new THREE.Vector3(2.5, 0.5, -1),
          new THREE.Vector3(3, 0.5, -2)
        ],
        duration: 2.5,
        ease: "power2.inOut"
      }
    };
    
    // Generate orbital path for story mode
    this._generateOrbitPath();
    
    // Current animation state
    this.currentRail = null;
    this.currentProgress = 0;
    this.isPlaying = false;
    this.currentTween = null;
    
    // Smooth camera position and target
    this.smoothPosition = camera.position.clone();
    this.smoothTarget = new THREE.Vector3(0, 1.4, 0);
    
    // Debug visualization
    this.debugMode = false;
    this.railVisualizations = new Map();
  }
  
  _generateOrbitPath() {
    const radius = 5;
    const height = 3.5;
    const segments = 16;
    
    const points = [];
    const targets = [];
    
    for (let i = 0; i <= segments; i++) {
      const angle = (i / segments) * Math.PI * 2;
      points.push(new THREE.Vector3(
        Math.cos(angle) * radius,
        height + Math.sin(angle * 2) * 0.5,
        Math.sin(angle) * radius
      ));
      targets.push(new THREE.Vector3(0, 1.4, 0));
    }
    
    this.rails.storyOrbit.points = points;
    this.rails.storyOrbit.targets = targets;
  }
  
  playRail(railName, options = {}) {
    const rail = this.rails[railName];
    if (!rail) {
      console.warn(`Camera rail "${railName}" not found`);
      return;
    }
    
    // Stop current animation
    if (this.currentTween) {
      this.currentTween.kill();
    }
    
    this.currentRail = railName;
    this.isPlaying = true;
    
    // Create curve from points
    const positionCurve = new THREE.CatmullRomCurve3(rail.points);
    const targetCurve = new THREE.CatmullRomCurve3(rail.targets);
    
    // Animation progress object
    const progress = { value: 0 };
    
    this.currentTween = gsap.to(progress, {
      value: 1,
      duration: rail.duration,
      ease: rail.ease,
      onUpdate: () => {
        const t = progress.value;
        
        // Sample curves
        const newPosition = positionCurve.getPoint(t);
        const newTarget = targetCurve.getPoint(t);
        
        // Apply to smooth values
        this.smoothPosition.copy(newPosition);
        this.smoothTarget.copy(newTarget);
        
        // Update camera
        this.camera.position.copy(this.smoothPosition);
        this.camera.lookAt(this.smoothTarget);
        
        this.currentProgress = t;
      },
      onComplete: () => {
        this.isPlaying = false;
        if (options.onComplete) {
          options.onComplete();
        }
        
        // Loop for orbital camera
        if (railName === 'storyOrbit' && options.loop) {
          this.playRail('storyOrbit', options);
        }
      }
    });
  }
  
  stopRail() {
    if (this.currentTween) {
      this.currentTween.kill();
      this.currentTween = null;
    }
    this.isPlaying = false;
    this.currentRail = null;
  }
  
  transitionTo(position, target, duration = 2) {
    if (this.currentTween) {
      this.currentTween.kill();
    }
    
    this.currentTween = gsap.to(this.camera.position, {
      x: position.x,
      y: position.y,
      z: position.z,
      duration: duration,
      ease: "power2.inOut",
      onUpdate: () => {
        this.camera.lookAt(target);
      }
    });
    
    gsap.to(this.smoothTarget, {
      x: target.x,
      y: target.y,
      z: target.z,
      duration: duration,
      ease: "power2.inOut"
    });
  }
  
  enableDebug() {
    this.debugMode = true;
    this._createDebugVisualization();
  }
  
  disableDebug() {
    this.debugMode = false;
    this._removeDebugVisualization();
  }
  
  _createDebugVisualization() {
    Object.entries(this.rails).forEach(([name, rail]) => {
      if (rail.points.length < 2) return;
      
      const curve = new THREE.CatmullRomCurve3(rail.points);
      const points = curve.getPoints(50);
      const geometry = new THREE.BufferGeometry().setFromPoints(points);
      const material = new THREE.LineBasicMaterial({
        color: 0x00ff00,
        opacity: 0.5,
        transparent: true
      });
      const line = new THREE.Line(geometry, material);
      
      this.scene.add(line);
      this.railVisualizations.set(name, line);
    });
  }
  
  _removeDebugVisualization() {
    this.railVisualizations.forEach(line => {
      this.scene.remove(line);
      line.geometry.dispose();
      line.material.dispose();
    });
    this.railVisualizations.clear();
  }
  
  update(delta) {
    const autopilot = useExperienceStore.getState().cameraAutopilot;
    if (!autopilot) {
      return;
    }

    if (!this.isPlaying && this.smoothPosition) {
      const lerpFactor = Math.min(delta * 2, 1);
      this.camera.position.lerp(this.smoothPosition, lerpFactor);
      
      // Smooth look-at
      const currentQuaternion = this.camera.quaternion.clone();
      this.camera.lookAt(this.smoothTarget);
      const targetQuaternion = this.camera.quaternion.clone();
      this.camera.quaternion.copy(currentQuaternion);
      this.camera.quaternion.slerp(targetQuaternion, lerpFactor);
    }
  }
  
  destroy() {
    this.stopRail();
    this._removeDebugVisualization();
  }
}
