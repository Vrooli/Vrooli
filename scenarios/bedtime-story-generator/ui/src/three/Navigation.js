import * as THREE from "three";
import normalizeWheel from "normalize-wheel";
import useExperienceStore from "../state/store.js";

const ROOM_NAV_CONFIGS = {
  studio: {
    spherical: {
      radius: 30,
      phi: Math.PI * 0.35,
      theta: -Math.PI * 0.25,
      radiusLimits: { min: 10, max: 50 },
      phiLimits: { min: 0.01, max: Math.PI * 0.5 },
      thetaLimits: { min: -Math.PI * 0.5, max: 0 },
    },
    target: new THREE.Vector3(0, 2, 0),
    targetLimits: {
      x: { min: -4, max: 4 },
      y: { min: 1, max: 6 },
      z: { min: -4, max: 4 },
    },
  },
  bedroom: {
    spherical: {
      radius: 18,
      phi: Math.PI * 0.4,
      theta: -Math.PI * 0.3,
      radiusLimits: { min: 8, max: 28 },
      phiLimits: { min: 0.05, max: Math.PI * 0.55 },
      thetaLimits: { min: -Math.PI * 0.6, max: Math.PI * 0.1 },
    },
    target: new THREE.Vector3(0, 1.6, -1),
    targetLimits: {
      x: { min: -3.5, max: 3.5 },
      y: { min: 0.8, max: 5 },
      z: { min: -4.5, max: 2 },
    },
  },
};

export default class Navigation {
  constructor({ experience, initialRoom = "studio" }) {
    this.experience = experience;
    this.targetElement = experience.target;
    this.camera = experience.camera.instance;
    this.time = experience.time;
    this.store = useExperienceStore;

    this.roomId = initialRoom;
    this.autopilot = this.store.getState().cameraAutopilot;

    this._buildView();
    this._attachEvents();

    this.unsubscribeAutopilot = this.store.subscribe(
      (state) => state.cameraAutopilot,
      (value) => {
        this.autopilot = value;
        if (value) {
          this._syncWithCamera();
        }
      },
    );
  }

  setAutopilot(enabled) {
    this.autopilot = enabled;
    if (enabled) {
      this._syncWithCamera();
    } else {
      this.experience.cameraRailSystem?.stopRail?.();
    }
  }

  updateRoom(roomId) {
    this.roomId = roomId;
    this._applyRoomConfig();
    this._syncWithCamera();
  }

  update(frame) {
    if (this.autopilot) {
      // Keep internal state aligned for smooth transitions when autopilot resumes.
      this._syncWithCamera();
      return;
    }

    const view = this.view;

    view.spherical.value.radius += view.zoom.delta * view.zoom.sensitivity;
    view.spherical.value.radius = THREE.MathUtils.clamp(
      view.spherical.value.radius,
      view.spherical.limits.radius.min,
      view.spherical.limits.radius.max,
    );

    if (view.drag.alternative) {
      const up = new THREE.Vector3(0, 1, 0);
      const right = new THREE.Vector3(-1, 0, 0);

      up.applyQuaternion(this.camera.quaternion);
      right.applyQuaternion(this.camera.quaternion);

      up.multiplyScalar(view.drag.delta.y * 0.01);
      right.multiplyScalar(view.drag.delta.x * 0.01);

      view.target.value.add(up);
      view.target.value.add(right);

      view.target.value.x = THREE.MathUtils.clamp(
        view.target.value.x,
        view.target.limits.x.min,
        view.target.limits.x.max,
      );
      view.target.value.y = THREE.MathUtils.clamp(
        view.target.value.y,
        view.target.limits.y.min,
        view.target.limits.y.max,
      );
      view.target.value.z = THREE.MathUtils.clamp(
        view.target.value.z,
        view.target.limits.z.min,
        view.target.limits.z.max,
      );
    } else {
    const smallestSide = this.experience.config.smallestSide || 600;
      view.spherical.value.theta -=
        (view.drag.delta.x * view.drag.sensitivity) / smallestSide;
      view.spherical.value.phi -=
        (view.drag.delta.y * view.drag.sensitivity) / smallestSide;

      view.spherical.value.theta = THREE.MathUtils.clamp(
        view.spherical.value.theta,
        view.spherical.limits.theta.min,
        view.spherical.limits.theta.max,
      );
      view.spherical.value.phi = THREE.MathUtils.clamp(
        view.spherical.value.phi,
        view.spherical.limits.phi.min,
        view.spherical.limits.phi.max,
      );
    }

    view.drag.delta.x = 0;
    view.drag.delta.y = 0;
    view.zoom.delta = 0;

    view.spherical.smoothed.radius +=
      (view.spherical.value.radius - view.spherical.smoothed.radius) *
      view.spherical.smoothing *
      this.time.delta;
    view.spherical.smoothed.phi +=
      (view.spherical.value.phi - view.spherical.smoothed.phi) *
      view.spherical.smoothing *
      this.time.delta;
    view.spherical.smoothed.theta +=
      (view.spherical.value.theta - view.spherical.smoothed.theta) *
      view.spherical.smoothing *
      this.time.delta;

    view.target.smoothed.x +=
      (view.target.value.x - view.target.smoothed.x) *
      view.target.smoothing *
      this.time.delta;
    view.target.smoothed.y +=
      (view.target.value.y - view.target.smoothed.y) *
      view.target.smoothing *
      this.time.delta;
    view.target.smoothed.z +=
      (view.target.value.z - view.target.smoothed.z) *
      view.target.smoothing *
      this.time.delta;

    const viewPosition = new THREE.Vector3();
    viewPosition.setFromSpherical(view.spherical.smoothed);
    viewPosition.add(view.target.smoothed);

    this.camera.position.copy(viewPosition);
    this.camera.lookAt(view.target.smoothed);
  }

  destroy() {
    this.unsubscribeAutopilot?.();
    this._detachEvents();
  }

  _buildView() {
    this.view = {
      spherical: {
        value: new THREE.Spherical(),
        smoothed: new THREE.Spherical(),
        smoothing: 0.005,
        limits: {
          radius: { min: 10, max: 50 },
          phi: { min: 0.01, max: Math.PI * 0.5 },
          theta: { min: -Math.PI * 0.5, max: 0 },
        },
      },
      target: {
        value: new THREE.Vector3(),
        smoothed: new THREE.Vector3(),
        smoothing: 0.005,
        limits: {
          x: { min: -4, max: 4 },
          y: { min: 1, max: 6 },
          z: { min: -4, max: 4 },
        },
      },
      drag: {
        delta: { x: 0, y: 0 },
        previous: { x: 0, y: 0 },
        sensitivity: 1,
        alternative: false,
      },
      zoom: {
        sensitivity: 0.01,
        delta: 0,
      },
    };

    this._applyRoomConfig();
    this._syncWithCamera();
  }

  _applyRoomConfig() {
    const config = ROOM_NAV_CONFIGS[this.roomId] || ROOM_NAV_CONFIGS.studio;
    const spherical = this.view.spherical;
    const target = this.view.target;

    spherical.value.radius = config.spherical.radius;
    spherical.value.phi = config.spherical.phi;
    spherical.value.theta = config.spherical.theta;
    spherical.smoothed.copy(spherical.value);
    spherical.limits.radius = config.spherical.radiusLimits;
    spherical.limits.phi = config.spherical.phiLimits;
    spherical.limits.theta = config.spherical.thetaLimits;

    target.value.copy(config.target);
    target.smoothed.copy(config.target);
    target.limits = config.targetLimits;
  }

  _syncWithCamera() {
    const position = this.camera.position.clone();
    const target = new THREE.Vector3();
    target.copy(this.view.target.smoothed);

    if (!this.autopilot) {
      const config = ROOM_NAV_CONFIGS[this.roomId] || ROOM_NAV_CONFIGS.studio;
      target.copy(config.target);
    }

    const offset = position.clone().sub(target);
    const spherical = new THREE.Spherical().setFromVector3(offset);

    this.view.spherical.value.radius = spherical.radius;
    this.view.spherical.value.phi = spherical.phi;
    this.view.spherical.value.theta = spherical.theta;
    this.view.spherical.smoothed.copy(this.view.spherical.value);

    if (!this.autopilot) {
      this.view.target.value.copy(target);
      this.view.target.smoothed.copy(target);
    }
  }

  _attachEvents() {
    this._onMouseDown = (event) => {
      event.preventDefault();
      this.view.drag.alternative =
        event.button === 2 || event.button === 1 || event.ctrlKey || event.shiftKey;
      this.view.drag.previous.x = event.clientX;
      this.view.drag.previous.y = event.clientY;

      if (this.autopilot) {
        this.store.getState().setCameraAutopilot(false);
      }

      window.addEventListener("mouseup", this._onMouseUp);
      window.addEventListener("mousemove", this._onMouseMove);
    };

    this._onMouseMove = (event) => {
      event.preventDefault();
      this.view.drag.delta.x += event.clientX - this.view.drag.previous.x;
      this.view.drag.delta.y += event.clientY - this.view.drag.previous.y;
      this.view.drag.previous.x = event.clientX;
      this.view.drag.previous.y = event.clientY;
    };

    this._onMouseUp = (event) => {
      event.preventDefault();
      window.removeEventListener("mouseup", this._onMouseUp);
      window.removeEventListener("mousemove", this._onMouseMove);
    };

    this._onWheel = (event) => {
      event.preventDefault();
      if (this.autopilot) {
        this.store.getState().setCameraAutopilot(false);
      }
      const normalized = normalizeWheel(event);
      this.view.zoom.delta += normalized.pixelY;
    };

    this._onTouchStart = (event) => {
      event.preventDefault();
      this.view.drag.alternative = event.touches.length > 1;
      const touch = event.touches[0];
      this.view.drag.previous.x = touch.clientX;
      this.view.drag.previous.y = touch.clientY;

      if (this.autopilot) {
        this.store.getState().setCameraAutopilot(false);
      }

      window.addEventListener("touchend", this._onTouchEnd);
      window.addEventListener("touchmove", this._onTouchMove);
    };

    this._onTouchMove = (event) => {
      event.preventDefault();
      const touch = event.touches[0];
      this.view.drag.delta.x += touch.clientX - this.view.drag.previous.x;
      this.view.drag.delta.y += touch.clientY - this.view.drag.previous.y;
      this.view.drag.previous.x = touch.clientX;
      this.view.drag.previous.y = touch.clientY;
    };

    this._onTouchEnd = (event) => {
      event.preventDefault();
      window.removeEventListener("touchend", this._onTouchEnd);
      window.removeEventListener("touchmove", this._onTouchMove);
    };

    this._onContextMenu = (event) => {
      event.preventDefault();
    };

    this.targetElement.addEventListener("mousedown", this._onMouseDown);
    window.addEventListener("mousewheel", this._onWheel, { passive: false });
    window.addEventListener("wheel", this._onWheel, { passive: false });
    window.addEventListener("touchstart", this._onTouchStart, {
      passive: false,
    });
    window.addEventListener("contextmenu", this._onContextMenu);
  }

  _detachEvents() {
    this.targetElement.removeEventListener("mousedown", this._onMouseDown);
    window.removeEventListener("mousewheel", this._onWheel);
    window.removeEventListener("wheel", this._onWheel);
    window.removeEventListener("touchstart", this._onTouchStart);
    window.removeEventListener("contextmenu", this._onContextMenu);
    window.removeEventListener("mouseup", this._onMouseUp);
    window.removeEventListener("mousemove", this._onMouseMove);
    window.removeEventListener("touchend", this._onTouchEnd);
    window.removeEventListener("touchmove", this._onTouchMove);
  }
}
