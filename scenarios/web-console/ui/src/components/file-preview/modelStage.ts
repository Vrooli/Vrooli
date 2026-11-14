import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";
import { DEFAULT_MODEL_OPTIONS, MAX_MODEL_TRIANGLES, modelCameraDistance, type ModelOptions, type ModelStats, type ModelView, type PreparedModel } from "./modelData";

export interface ModelStage { options: (value: ModelOptions) => void; view: (value: ModelView) => void; zoom: (factor: number) => void; orbit: (x: number, y: number) => void; dispose: () => void; }
function isMesh(value: THREE.Object3D): value is THREE.Mesh { return value instanceof THREE.Mesh; }
function isTexture(value: unknown): value is THREE.Texture { return value instanceof THREE.Texture; }
// Own all resources, including textures cached on non-default glTF scenes.
export function disposeModel(roots: THREE.Object3D[]) {
  const geometries = new Set<THREE.BufferGeometry>(), materials = new Set<THREE.Material>(), textures = new Set<THREE.Texture>(), images = new Set<ImageBitmap>();
  for (const root of roots) root.traverse(object => {
    if (!isMesh(object)) return;
    geometries.add(object.geometry);
    for (const material of Array.isArray(object.material) ? object.material : [object.material]) {
      materials.add(material);
      for (const value of Object.values(material)) if (isTexture(value)) { textures.add(value); if (typeof ImageBitmap !== "undefined" && value.source.data instanceof ImageBitmap) images.add(value.source.data); }
    }
  });
  geometries.forEach(value => { value.dispose(); }); materials.forEach(value => { value.dispose(); }); textures.forEach(value => { value.dispose(); }); images.forEach(value => { value.close(); });
}

async function loadModel(data: PreparedModel): Promise<{ root: THREE.Object3D; roots: THREE.Object3D[] }> {
  if ("positions" in data) {
    const geometry = new THREE.BufferGeometry(); geometry.setAttribute("position", new THREE.BufferAttribute(data.positions, 3)); if (data.normals) geometry.setAttribute("normal", new THREE.BufferAttribute(data.normals, 3)); else geometry.computeVertexNormals();
    const material = new THREE.MeshStandardMaterial({ color: 0x8fbac5, metalness: 0.12, roughness: 0.42, side: THREE.DoubleSide });
    const root = new THREE.Mesh(geometry, material); return { root, roots: [root] };
  }
  const manager = new THREE.LoadingManager();
  // Preflight admits only embedded assets. Defense in depth at the actual loader
  // boundary prevents future loader changes from acquiring network authority.
  const uris = new Set(data.uris);
  manager.setURLModifier(url => { if (uris.has(url) || url.startsWith(`blob:${window.location.origin}/`)) return url; throw Error("external"); });
  const loader = new GLTFLoader(manager);
  const gltf = await loader.parseAsync(data.bytes, "");
  return { root: gltf.scene, roots: gltf.scenes };
}

export async function createModelStage(container: HTMLElement, data: PreparedModel, signal: AbortSignal, report: (value: ModelStats) => void, fail: () => void): Promise<ModelStage | null> {
  let owned: { root: THREE.Object3D; roots: THREE.Object3D[] } | undefined;
  let renderer: THREE.WebGLRenderer | undefined, controls: OrbitControls | undefined, environment: THREE.WebGLRenderTarget | undefined;
  let observer: ResizeObserver | undefined, frame = 0, lastFrame = 0, disposed = false, options = DEFAULT_MODEL_OPTIONS;
  const scene = new THREE.Scene(), camera = new THREE.PerspectiveCamera(40, 1, 0.01, 100);
  const group = new THREE.Group(); scene.add(group);
  const hemisphere = new THREE.HemisphereLight(0xffffff, 0x52627a, 1.4); scene.add(hemisphere);
  const key = new THREE.DirectionalLight(0xfff5e6, 3); key.position.set(3, 5, 4); scene.add(key);
  const fill = new THREE.DirectionalLight(0xc0dcff, 2); fill.position.set(-4, 2, -3); scene.add(fill);
  const grid = new THREE.GridHelper(8, 32, 0x526677, 0x394b5b); scene.add(grid);
  const gridMaterials = [grid.material];
  gridMaterials.forEach(material => { material.transparent = true; material.opacity = 0.28; material.depthWrite = false; });
  const visible = () => !disposed && !signal.aborted && !document.hidden && container.clientWidth > 0 && container.clientHeight > 0;
  const draw = (time: number) => {
    frame = 0; if (!visible() || !renderer || !controls) return;
    try { controls.update(Math.min(0.05, (time - lastFrame) / 1000)); lastFrame = time; renderer.render(scene, camera); if (options.rotate) requestDraw(); } catch { fail(); dispose(); }
  };
  const requestDraw = () => { if (!frame && visible()) frame = requestAnimationFrame(draw); };
  const lost = (event: Event) => { event.preventDefault(); fail(); dispose(); };
  const dispose = () => {
    if (disposed) return; disposed = true; signal.removeEventListener("abort", dispose);
    cancelAnimationFrame(frame); observer?.disconnect(); document.removeEventListener("visibilitychange", requestDraw);
    controls?.dispose(); renderer?.domElement.removeEventListener("webglcontextlost", lost);
    if (owned) disposeModel(owned.roots); grid.geometry.dispose(); gridMaterials.forEach(material => { material.dispose(); }); environment?.dispose();
    renderer?.dispose(); renderer?.forceContextLoss(); renderer?.domElement.remove();
  };
  signal.addEventListener("abort", dispose, { once: true });
  try {
    // Test browser capability before parsing or decoding publisher assets.
    try { renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false, powerPreference: "low-power" }); } catch { throw Error("unavailable"); }
    renderer.setPixelRatio(1); renderer.outputColorSpace = THREE.SRGBColorSpace; renderer.toneMapping = THREE.ACESFilmicToneMapping; renderer.toneMappingExposure = 1.1;
    renderer.domElement.setAttribute("aria-hidden", "true"); renderer.domElement.style.cssText = "display:block;width:100%;height:100%;touch-action:none";
    container.append(renderer.domElement); renderer.domElement.addEventListener("webglcontextlost", lost);
    owned = await loadModel(data);
    if (signal.aborted || !renderer.domElement.isConnected) { disposeModel(owned.roots); return null; }
    const box = new THREE.Box3().setFromObject(owned.root), center = box.getCenter(new THREE.Vector3()), size = box.getSize(new THREE.Vector3()), radius = size.length() / 2;
    if (![...size, ...center, radius].every(Number.isFinite) || radius <= 1e-12 || radius > 1e15) throw Error("invalid");
    let meshes = 0, triangles = 0;
    owned.root.traverse(object => {
      if (isMesh(object)) { meshes++; triangles += (object.geometry.index?.count ?? object.geometry.getAttribute("position").count) / 3; }
    });
    if (!meshes || triangles > MAX_MODEL_TRIANGLES) throw new RangeError("geometry");
    // Normalize without mutating authored geometry, preserving relative layout.
    owned.root.position.sub(center); group.add(owned.root); group.scale.setScalar(1 / radius);
    grid.position.y = -size.y / (2 * radius) - 0.008;
    const generator = new THREE.PMREMGenerator(renderer), room = new RoomEnvironment();
    try { environment = generator.fromScene(room, 0.04, 0.1, 20, { size: 128 }); scene.environment = environment.texture; } finally { generator.dispose(); room.dispose(); }
    controls = new OrbitControls(camera, renderer.domElement); controls.enableDamping = false; controls.minDistance = 1.15; controls.maxDistance = 25; controls.enablePan = true; controls.autoRotateSpeed = 1.3;
    controls.addEventListener("change", requestDraw);
    const view = (value: ModelView) => {
      if (!controls) return;
      const distance = modelCameraDistance(1, camera.aspect), direction = value === "front" ? new THREE.Vector3(0, 0, 1) : value === "right" ? new THREE.Vector3(1, 0, 0) : value === "top" ? new THREE.Vector3(0, 1, 0.0001) : new THREE.Vector3(1, 0.65, 1);
      controls.target.set(0, 0, 0); camera.position.copy(direction.normalize().multiplyScalar(distance)); camera.up.set(0, 1, 0); controls.update(); requestDraw();
    };
    const resize = () => {
      if (!renderer || !controls) return;
      const width = container.clientWidth, height = container.clientHeight; if (!width || !height) return;
      const old = camera.aspect, aspect = width / height;
      camera.position.sub(controls.target).multiplyScalar(modelCameraDistance(1, aspect) / modelCameraDistance(1, old)).add(controls.target);
      camera.aspect = aspect; camera.updateProjectionMatrix();
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.75, Math.sqrt(2000000 / (width * height)))); renderer.setSize(width, height, false); controls.update(); requestDraw();
    };
    observer = new ResizeObserver(resize); observer.observe(container); document.addEventListener("visibilitychange", requestDraw); resize(); view("perspective");
    report({ meshes, triangles: Math.round(triangles), dimensions: [size.x, size.y, size.z] }); requestDraw();
    return {
      options(value) {
        options = value; if (!controls) return; controls.autoRotate = value.rotate;
        scene.background = new THREE.Color(value.background === "light" ? 0xe8edf1 : value.background === "dark" ? 0x0a0e14 : 0x17222e);
        grid.visible = value.grid; gridMaterials.forEach(material => { material.color.set(value.background === "light" ? 0x738493 : 0x728b9b); });
        key.intensity = value.lighting === "soft" ? 1 : 3; fill.intensity = value.lighting === "soft" ? 0.6 : 2; hemisphere.intensity = value.lighting === "soft" ? 2 : 1.4;
        owned?.root.traverse(object => { if (isMesh(object)) for (const material of Array.isArray(object.material) ? object.material : [object.material]) if (material instanceof THREE.MeshStandardMaterial || material instanceof THREE.MeshBasicMaterial) material.wireframe = value.wireframe; }); requestDraw();
      },
      view,
      zoom(factor) { if (!controls) return; const direction = camera.position.clone().sub(controls.target), distance = THREE.MathUtils.clamp(direction.length() * factor, controls.minDistance, controls.maxDistance); camera.position.copy(controls.target).add(direction.normalize().multiplyScalar(distance)); controls.update(); requestDraw(); },
      orbit(x, y) { if (!controls) return; const offset = camera.position.clone().sub(controls.target), spherical = new THREE.Spherical().setFromVector3(offset); spherical.theta += x; spherical.phi = THREE.MathUtils.clamp(spherical.phi + y, 0.01, Math.PI - 0.01); camera.position.copy(controls.target).add(new THREE.Vector3().setFromSpherical(spherical)); controls.update(); requestDraw(); },
      dispose,
    };
  } catch (error) { dispose(); throw error; }
}
