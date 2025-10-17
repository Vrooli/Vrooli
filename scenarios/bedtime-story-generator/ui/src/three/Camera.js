import * as THREE from "three";

export default class Camera {
  constructor({ experience }) {
    this.experience = experience;
    this.config = experience.config;
    this.sizes = experience.sizes;
    this.scene = experience.scene;

    this._createInstance();
  }

  _createInstance() {
    const aspect = this.sizes.width / this.sizes.height;
    this.instance = new THREE.PerspectiveCamera(36, aspect, 0.1, 200);
    this.instance.position.set(6.5, 3.2, 6.5);
    this.instance.lookAt(0, 1.5, 0);
    this.scene.add(this.instance);
  }

  resize() {
    this.instance.aspect = this.sizes.width / this.sizes.height;
    this.instance.updateProjectionMatrix();
  }

  update() {}

  destroy() {
    this.scene.remove(this.instance);
  }
}
