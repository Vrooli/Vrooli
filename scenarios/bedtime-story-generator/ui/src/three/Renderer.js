import * as THREE from "three";

export default class Renderer {
  constructor({ experience }) {
    this.experience = experience;
    this.sizes = experience.sizes;
    this.scene = experience.scene;
    this.camera = experience.camera;
    this.target = experience.target;

    this._createRenderer();
  }

  _createRenderer() {
    this.instance = new THREE.WebGLRenderer({ antialias: true, alpha: false });
    this.instance.outputColorSpace = THREE.SRGBColorSpace;
    this.instance.setClearColor("#0d1022", 1);
    this.instance.setPixelRatio(this.sizes.pixelRatio);
    this.instance.setSize(this.sizes.width, this.sizes.height);

    this.target.appendChild(this.instance.domElement);
  }

  resize() {
    this.instance.setPixelRatio(this.sizes.pixelRatio);
    this.instance.setSize(this.sizes.width, this.sizes.height);
  }

  update() {
    this.instance.render(this.scene, this.camera.instance);
  }

  destroy() {
    this.target?.removeChild?.(this.instance.domElement);
    this.instance.dispose();
  }
}
