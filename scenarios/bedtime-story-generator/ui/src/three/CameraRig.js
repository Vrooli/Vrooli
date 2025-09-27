import * as THREE from "three";
import useExperienceStore from "../state/store.js";

const CAMERA_PRESETS = {
  studio: {
    wide: {
      position: new THREE.Vector3(6.4, 3.9, 6.3),
      target: new THREE.Vector3(0, 1.4, 0),
    },
    bookshelf: {
      position: new THREE.Vector3(3.2, 2.9, 2.4),
      target: new THREE.Vector3(3.05, 0.2, -1.4),
    },
    bed: {
      position: new THREE.Vector3(-1.8, 2.6, 4.1),
      target: new THREE.Vector3(-1.6, 0.6, -1.1),
    },
    window: {
      position: new THREE.Vector3(0.4, 2.5, 1.1),
      target: new THREE.Vector3(0, 1.4, -4.0),
    },
    mobile: {
      position: new THREE.Vector3(-3.5, 3.1, 3.4),
      target: new THREE.Vector3(-2.4, 2.1, 1.2),
    },
  },
  bedroom: {
    wide: {
      position: new THREE.Vector3(8.5, 4.2, 5.8),
      target: new THREE.Vector3(0, 1.6, -1),
    },
    bookshelf: {
      position: new THREE.Vector3(3.2, 2.6, -2.6),
      target: new THREE.Vector3(2.2, 1.0, -2.4),
    },
    bed: {
      position: new THREE.Vector3(-3.4, 2.7, 3.2),
      target: new THREE.Vector3(-2.4, 0.9, -0.8),
    },
    window: {
      position: new THREE.Vector3(0.6, 2.8, 3.8),
      target: new THREE.Vector3(0, 1.8, -2.8),
    },
    mobile: {
      position: new THREE.Vector3(-5.0, 3.4, 0.6),
      target: new THREE.Vector3(-1.5, 2.0, -0.2),
    },
  },
};

const DEFAULT_PRESET = {
  studio: "wide",
  bedroom: "wide",
};

export default class CameraRig {
  constructor({ experience, initialRoom = "studio" }) {
    this.camera = experience.camera.instance;
    this.currentPosition = this.camera.position.clone();
    this.currentTarget = new THREE.Vector3(0, 1.4, 0);
    this.desiredPosition = this.currentPosition.clone();
    this.desiredTarget = this.currentTarget.clone();
    this.lag = 0.08;

    this.roomId = initialRoom;
    this.store = useExperienceStore;
    this.autopilot = this.store.getState().cameraAutopilot;

    this.unsubscribeAutopilot = this.store.subscribe(
      (state) => state.cameraAutopilot,
      (value) => {
        this.autopilot = value;
        if (value) {
          this.currentPosition.copy(this.camera.position);
          this.desiredPosition.copy(this.camera.position);
        }
      },
    );

    const { cameraFocus } = this.store.getState();
    this.updateFocus(cameraFocus, this.roomId);
  }

  updateRoom(roomId) {
    const nextRoom = roomId || this.roomId;
    this.roomId = nextRoom;
    const currentFocus = this.store.getState().cameraFocus;
    this.updateFocus(currentFocus, nextRoom);
  }

  updateFocus(focusId, roomId = this.roomId) {
    const presets = CAMERA_PRESETS[roomId] || CAMERA_PRESETS.studio;
    const preset = presets[focusId] || presets[DEFAULT_PRESET[roomId] || "wide"];
    this.desiredPosition.copy(preset.position);
    this.desiredTarget.copy(preset.target);
  }

  update(frame) {
    if (!this.autopilot) {
      this.currentPosition.copy(this.camera.position);
      return;
    }

    const smoothing = this.lag * Math.min(frame.delta * 0.06, 1);

    this.currentPosition.lerp(this.desiredPosition, smoothing);
    this.currentTarget.lerp(this.desiredTarget, smoothing);

    this.camera.position.copy(this.currentPosition);
    this.camera.lookAt(this.currentTarget);
  }

  destroy() {
    this.unsubscribeAutopilot?.();
  }
}
