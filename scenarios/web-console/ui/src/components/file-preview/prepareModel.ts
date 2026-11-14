import { MAX_MODEL_BYTES, MAX_MODEL_TRIANGLES, MAX_MODEL_VERTICES, type ModelFormat, type PreparedModel } from "./modelData";
import { rasterPixels } from "./rasterPixels";

const decoder = new TextDecoder("utf-8", { fatal: true });
const invalid = (): never => { throw Error("invalid"); };
function number(value: unknown, min = 0, max = Number.MAX_SAFE_INTEGER): number {
  if (typeof value !== "number" || !Number.isFinite(value) || value < min || value > max) return invalid();
  return value;
}
function integer(value: unknown, max = Number.MAX_SAFE_INTEGER): number { const n = number(value, 0, max); if (!Number.isSafeInteger(n)) return invalid(); return n; }
function record(value: unknown): Record<string, unknown> { if (!value || typeof value !== "object" || Array.isArray(value)) return invalid(); return value as Record<string, unknown>; }
function list(value: unknown, max: number): unknown[] { if (value === undefined) return []; if (!Array.isArray(value)) return invalid(); if (value.length > max) throw new RangeError("elements"); return value; }
function budget(vertices: number, triangles: number) { if (vertices > MAX_MODEL_VERTICES || triangles > MAX_MODEL_TRIANGLES) throw new RangeError("geometry"); }
function finiteCoordinates(values: ArrayLike<number>) { for (let i = 0; i < values.length; i++) number(values[i], -1e12, 1e12); }
function dataBytes(uri: unknown, image = false): Uint8Array {
  if (typeof uri !== "string" || !uri.startsWith("data:")) throw Error("external");
  const match = /^(?:data:)(application\/(?:octet-stream|gltf-buffer)|image\/(?:png|jpeg|webp));base64,([A-Za-z0-9+/]*={0,2})$/.exec(uri);
  if (!match || image !== !!match[1]?.startsWith("image/")) throw Error("unsupported");
  const raw = atob(match[2] ?? ""); if (raw.length > MAX_MODEL_BYTES) throw new RangeError("bytes");
  return Uint8Array.from(raw, char => char.charCodeAt(0));
}

// These loader-native material features add bounded shading inputs, not new
// asset acquisition or compression decoders. Share admission and validation.
const materialFeatures = [
  { name: "KHR_materials_transmission", factors: ["transmissionFactor"], textures: ["transmissionTexture"] },
  { name: "KHR_materials_clearcoat", factors: ["clearcoatFactor", "clearcoatRoughnessFactor"], textures: ["clearcoatTexture", "clearcoatRoughnessTexture", "clearcoatNormalTexture"] },
  { name: "KHR_materials_specular", factors: ["specularFactor"], textures: ["specularTexture", "specularColorTexture"] },
] as const;

function validateMaterialFeatures(materials: unknown[], textureCount: number) {
  for (const item of materials) {
    const material = record(item), extensions = material.extensions === undefined ? {} : record(material.extensions);
    for (const feature of materialFeatures) {
      if (extensions[feature.name] === undefined) continue;
      const properties = record(extensions[feature.name]);
      if (extensions.KHR_materials_unlit !== undefined) return invalid();
      for (const field of feature.factors) if (properties[field] !== undefined) number(properties[field], 0, 1);
      for (const field of feature.textures) {
        if (properties[field] === undefined) continue;
        const texture = record(properties[field]);
        integer(texture.index, textureCount - 1);
        if (texture.texCoord !== undefined) integer(texture.texCoord);
        if (field === "clearcoatNormalTexture" && texture.scale !== undefined) number(texture.scale, -1e12, 1e12);
      }
      if (feature.name === "KHR_materials_specular" && properties.specularColorFactor !== undefined) {
        const color = list(properties.specularColorFactor, 3);
        if (color.length !== 3) return invalid();
        // The spec permits HDR specular colors above 1; keep the existing
        // finite numeric ceiling without restricting them to SDR channels.
        for (const value of color) number(value, 0, 1e12);
      }
    }
  }
}

// Bound syntax before JSON.parse/Object.entries can materialize a huge tree.
function jsonDocument(text: string): Record<string, unknown> {
  let quoted = false, escaped = false, depth = 0, tokens = 0;
  for (const char of text) {
    if (quoted) { if (escaped) escaped = false; else if (char === "\\") escaped = true; else if (char === '"') quoted = false; continue; }
    if (char === '"') quoted = true;
    else if (char === "{" || char === "[") { depth++; tokens++; }
    else if (char === "}" || char === "]") depth--;
    else if (char === "," || char === ":") tokens++;
    if (depth > 48 || tokens > 200000) throw new RangeError("structure");
    if (depth < 0) return invalid();
  }
  return record(JSON.parse(text, (key: string, value: unknown) => {
    // Object3D cloning deep-copies extras. Unused metadata and animation data
    // must not multiply with scene instances on the main thread.
    return key === "extras" || key === "animations" ? undefined : value;
  }));
}

function gltf(bytes: ArrayBuffer, format: "glb" | "gltf"): PreparedModel {
  let json: Record<string, unknown> | undefined, binary: Uint8Array | undefined;
  if (format === "glb") {
    const view = new DataView(bytes);
    if (bytes.byteLength < 20 || view.getUint32(0, true) !== 0x46546c67 || view.getUint32(4, true) !== 2 || view.getUint32(8, true) !== bytes.byteLength) return invalid();
    let offset = 12, chunk = 0;
    while (offset + 8 <= bytes.byteLength) {
      const length = view.getUint32(offset, true), type = view.getUint32(offset + 4, true); offset += 8;
      if (length % 4 || offset + length > bytes.byteLength || chunk > 1) return invalid();
      if (chunk === 0 && type === 0x4e4f534a) json = jsonDocument(decoder.decode(new Uint8Array(bytes, offset, length)));
      else if (chunk === 1 && type === 0x004e4942) binary = new Uint8Array(bytes, offset, length);
      else return invalid();
      offset += length; chunk++;
    }
    if (offset !== bytes.byteLength || !chunk) return invalid();
    if (!json) return invalid();
  } else json = jsonDocument(decoder.decode(bytes));
  if (record(json.asset).version !== "2.0") throw Error("unsupported");
  // Reject extensions everywhere, not just the advisory extensionsUsed array.
  const allowed = new Set(["KHR_materials_unlit", "KHR_texture_transform", "KHR_mesh_quantization", "KHR_materials_emissive_strength", ...materialFeatures.map(feature => feature.name)]);
  const stack: { value: unknown; depth: number }[] = [{ value: json, depth: 0 }]; let elements = 0;
  while (stack.length) {
    const entry = stack.pop(); if (!entry) break;
    if (++elements > 200000 || entry.depth > 48) throw new RangeError("structure");
    if (typeof entry.value === "number") number(entry.value, -1e12, 1e12);
    if (entry.value && typeof entry.value === "object") for (const [key, value] of Object.entries(entry.value)) {
      if (key === "name" && (typeof value !== "string" || value.length > 1024)) throw new RangeError("label");
      if (key === "extensions") for (const name of Object.keys(record(value))) if (!allowed.has(name)) throw Error("unsupported");
      stack.push({ value, depth: entry.depth + 1 });
    }
  }
  for (const name of list(json.extensionsRequired, 16)) if (typeof name !== "string" || !allowed.has(name)) throw Error("unsupported");
  if (list(json.skins, 128).length) throw Error("unsupported");
  const buffers = list(json.buffers, 32).map((item, i) => {
    const source = record(item), length = integer(source.byteLength, MAX_MODEL_BYTES);
    const value = source.uri === undefined && i === 0 && binary ? binary : dataBytes(source.uri);
    if (length > value.length || value.length - length > (source.uri === undefined ? 3 : 0)) return invalid(); return value;
  });
  if (buffers.reduce((sum, value) => sum + value.length, 0) > MAX_MODEL_BYTES) throw new RangeError("buffers");
  const views = list(json.bufferViews, 4096).map(item => {
    const v = record(item), buffer = buffers[integer(v.buffer)], start = integer(v.byteOffset ?? 0), length = integer(v.byteLength);
    if (!buffer || start + length > buffer.length) return invalid();
    if (v.byteStride !== undefined && (!Number.isInteger(v.byteStride) || number(v.byteStride, 4, 252) % 4)) return invalid();
    return { value: buffer.subarray(start, start + length), stride: v.byteStride as number | undefined };
  });
  let accessorBytes = 0;
  const accessors = list(json.accessors, 4096).map(item => {
    const a = record(item); if (a.sparse !== undefined) throw Error("unsupported");
    const width = ({ SCALAR: 1, VEC2: 2, VEC3: 3, VEC4: 4, MAT2: 4, MAT3: 9, MAT4: 16 } as Record<string, number>)[String(a.type)];
    const size = ({ 5120: 1, 5121: 1, 5122: 2, 5123: 2, 5125: 4, 5126: 4 } as Record<string, number>)[String(a.componentType)];
    const count = integer(a.count, MAX_MODEL_VERTICES), start = integer(a.byteOffset ?? 0), v = views[integer(a.bufferView)];
    if (!width || !size || !v || !count) return invalid();
    const stride = v.stride ?? width * size;
    if (stride < width * size || start % size || start + (count - 1) * stride + width * size > v.value.length) return invalid();
    accessorBytes += count * width * size; if (accessorBytes > 32 * 1024 * 1024) throw new RangeError("accessors");
    if (a.componentType === 5126) {
      const d = new DataView(v.value.buffer, v.value.byteOffset, v.value.length);
      for (let i = 0; i < count; i++) for (let c = 0; c < width; c++) number(d.getFloat32(start + i * stride + c * size, true), -1e12, 1e12);
    }
    let maximum = -1;
    if (a.type === "SCALAR" && [5121, 5123, 5125].includes(Number(a.componentType))) {
      const d = new DataView(v.value.buffer, v.value.byteOffset, v.value.length);
      for (let i = 0; i < count; i++) { const offset = start + i * stride; maximum = Math.max(maximum, size === 1 ? d.getUint8(offset) : size === 2 ? d.getUint16(offset, true) : d.getUint32(offset, true)); }
    }
    return { count, type: a.type, component: a.componentType, maximum };
  });
  const meshes = list(json.meshes, 256).map(item => {
    let vertices = 0, triangles = 0;
    const primitives = list(record(item).primitives, 256);
    for (const raw of primitives) {
      const p = record(raw); if (list(p.targets, 8).length || p.mode !== undefined && p.mode !== 4) throw Error("unsupported");
      const position = accessors[integer(record(p.attributes).POSITION)], index = p.indices === undefined ? position : accessors[integer(p.indices)];
      if (!position || position.type !== "VEC3" || !index || index.count % 3) return invalid();
      if (p.indices !== undefined && (index.type !== "SCALAR" || ![5121, 5123, 5125].includes(Number(index.component)) || index.maximum >= position.count)) return invalid();
      for (const reference of Object.values(record(p.attributes))) { const attribute = accessors[integer(reference)]; if (!attribute || attribute.count !== position.count) return invalid(); }
      vertices += position.count; triangles += index.count / 3;
    }
    budget(vertices, triangles); return { vertices, triangles, draws: primitives.length };
  });
  const nodes = list(json.nodes, 2048).map(record), parents = new Set<number>(); let vertices = 0, triangles = 0, draws = 0;
  for (const node of nodes) {
    if (node.mesh !== undefined) { const mesh = meshes[integer(node.mesh)]; if (!mesh) return invalid(); vertices += mesh.vertices; triangles += mesh.triangles; draws += mesh.draws; }
    for (const child of list(node.children, 2048)) { const index = integer(child, nodes.length - 1); if (parents.has(index)) return invalid(); parents.add(index); }
    for (const key of ["matrix", "rotation", "translation", "scale"]) if (node[key] !== undefined) { const values = list(node[key], 16); if (values.length !== ({ matrix: 16, rotation: 4, translation: 3, scale: 3 } as Record<string, number>)[key]) return invalid(); for (const value of values) number(value, -1e6, 1e6); }
  }
  budget(vertices, triangles); if (draws > 256) throw new RangeError("draws");
  // A forest avoids cycles and exponential scene cloning before Three sees it.
  const visited = new Set<number>();
  const walk = (index: number, depth: number) => {
    if (depth > 64) throw new RangeError("depth"); if (visited.has(index)) return invalid(); visited.add(index);
    for (const child of list(nodes[index]?.children, 2048)) walk(integer(child, nodes.length - 1), depth + 1);
  };
  nodes.forEach((_, index) => { if (!parents.has(index)) walk(index, 0); }); if (visited.size !== nodes.length) return invalid();
  const scenes = list(json.scenes, 8); if (!scenes.length || nodes.length * scenes.length > 4096) throw new RangeError("scenes");
  integer(json.scene ?? 0, scenes.length - 1);
  for (const scene of scenes) { const roots = new Set<number>(); for (const root of list(record(scene).nodes, 2048)) { const index = integer(root, nodes.length - 1); if (parents.has(index) || roots.has(index)) return invalid(); roots.add(index); } }
  const materials = list(json.materials, 256), textures = list(json.textures, 32), samplers = list(json.samplers, 32);
  validateMaterialFeatures(materials, textures.length);
  let pixels = 0; const imagePixels: number[] = [];
  for (const item of list(json.images, 32)) {
    const image = record(item), uri = image.uri;
    const value = uri === undefined ? views[integer(image.bufferView)]?.value : dataBytes(uri, true);
    if (!value) return invalid();
    // Inspect bytes, never trust a publisher-supplied MIME label.
    const kind = value[0] === 0x89 ? "png" : value[0] === 0xff ? "jpg" : value[0] === 0x52 && value[8] === 0x57 && value[9] === 0x45 && value[10] === 0x42 && value[11] === 0x50 ? "webp" : null;
    if (!kind) throw Error("unsupported");
    const mime = ({ png: "image/png", jpg: "image/jpeg", webp: "image/webp" })[kind];
    if (uri === undefined ? image.mimeType !== mime : typeof uri !== "string" || !uri.startsWith(`data:${mime};base64,`)) throw Error("unsupported");
    const area = rasterPixels(value, `texture.${kind}`, 4096, 8 * 1024 * 1024); imagePixels.push(area); pixels += area;
    if (pixels > 16 * 1024 * 1024) throw new RangeError("pixels");
  }
  let texturePixels = 0;
  for (const item of textures) {
    const texture = record(item), area = imagePixels[integer(texture.source, imagePixels.length - 1)]; if (!area) return invalid();
    if (texture.sampler !== undefined) integer(texture.sampler, samplers.length - 1);
    // Different samplers can upload the same source image repeatedly. Charge
    // each texture, not only unique decoded images, before any GPU allocation.
    texturePixels += area; if (texturePixels > 16 * 1024 * 1024) throw new RangeError("textures");
  }
  const uris = [...list(json.buffers, 32), ...list(json.images, 32)].map(item => record(item).uri).filter((uri): uri is string => typeof uri === "string");
  // Only the selected scene is inspected. Keep the GLB binary chunk but publish
  // a sanitized JSON chunk, so the renderer never receives cloneable extras.
  json.scenes = [scenes[integer(json.scene ?? 0, scenes.length - 1)]]; json.scene = 0;
  const encoded = new TextEncoder().encode(JSON.stringify(json));
  if (format === "gltf") return { format, bytes: encoded.buffer, uris };
  const jsonLength = Math.ceil(encoded.length / 4) * 4;
  const output = new ArrayBuffer(20 + jsonLength + (binary ? 8 + binary.length : 0)), header = new DataView(output);
  header.setUint32(0, 0x46546c67, true); header.setUint32(4, 2, true); header.setUint32(8, output.byteLength, true); header.setUint32(12, jsonLength, true); header.setUint32(16, 0x4e4f534a, true);
  const content = new Uint8Array(output); content.fill(32, 20, 20 + jsonLength); content.set(encoded, 20);
  if (binary) { const offset = 20 + jsonLength; header.setUint32(offset, binary.length, true); header.setUint32(offset + 4, 0x004e4942, true); content.set(binary, offset + 8); }
  return { format, bytes: output, uris };
}

function obj(bytes: ArrayBuffer): PreparedModel {
  const vertices: number[][] = [], sourceNormals: number[][] = [], positions: number[] = [], normals: number[] = []; let partial = false, lines = 0, completeNormals = true;
  for (const match of decoder.decode(bytes).matchAll(/[^\r\n]+/g)) {
    const raw = match[0].split("#", 1)[0] ?? "";
    if (++lines > 800000 || raw.length > 65536) throw new RangeError("lines");
    const tokens = raw.trim().split(/\s+/), kind = tokens.shift();
    if (kind === "v") {
      if (tokens.length < 3) return invalid();
      const vertex = tokens.slice(0, 3).map(Number); finiteCoordinates(vertex); vertices.push(vertex);
      if (vertices.length > MAX_MODEL_VERTICES) throw new RangeError("vertices");
    } else if (kind === "vn") {
      if (tokens.length !== 3) return invalid();
      const normal = tokens.map(Number); finiteCoordinates(normal); sourceNormals.push(normal);
      if (sourceNormals.length > MAX_MODEL_VERTICES) throw new RangeError("normals");
    } else if (kind === "f") {
      if (tokens.length < 3 || tokens.length > 1024) return invalid();
      const face = tokens.map(token => {
        const fields = token.split("/"), n = Number(fields[0]); if (!Number.isInteger(n) || !n) return invalid();
        const vertex = vertices[n < 0 ? vertices.length + n : n - 1]; if (!vertex) return invalid();
        let normal: number[] | undefined;
        if (fields[2]) { const index = Number(fields[2]); if (!Number.isInteger(index) || !index) return invalid(); normal = sourceNormals[index < 0 ? sourceNormals.length + index : index - 1]; if (!normal) return invalid(); }
        return { vertex, normal };
      });
      budget(positions.length / 3 + (face.length - 2) * 3, positions.length / 9 + face.length - 2);
      for (let i = 1; i < face.length - 1; i++) for (const vertex of [face[0], face[i], face[i + 1]]) { if (!vertex) return invalid(); positions.push(...vertex.vertex); if (vertex.normal) normals.push(...vertex.normal); else completeNormals = false; }
    } else if (kind === "mtllib" || kind === "usemtl" || kind === "l" || kind === "p") partial = true;
  }
  if (!positions.length) return invalid(); return { format: "obj", positions: new Float32Array(positions), ...(completeNormals ? { normals: new Float32Array(normals) } : {}), partial };
}
function stl(bytes: ArrayBuffer): PreparedModel {
  const view = new DataView(bytes); let positions: Float32Array;
  if (bytes.byteLength >= 84 && 84 + view.getUint32(80, true) * 50 === bytes.byteLength) {
    const count = view.getUint32(80, true); budget(count * 3, count); positions = new Float32Array(count * 9);
    for (let face = 0; face < count; face++) for (let n = 0; n < 9; n++) positions[face * 9 + n] = view.getFloat32(84 + face * 50 + 12 + n * 4, true);
  } else {
    const text = decoder.decode(bytes); if (!/^\s*solid\b/.test(text) || !/\bendsolid\b/.test(text)) return invalid();
    const values: number[] = []; let facets = 0;
    for (const match of text.matchAll(/\bfacet\s+normal\b([\s\S]*?)\bendfacet\b/g)) {
      if (++facets > MAX_MODEL_TRIANGLES) throw new RangeError("geometry");
      const vertices = [...(match[1] ?? "").matchAll(/\bvertex\s+(\S+)\s+(\S+)\s+(\S+)/g)]; if (vertices.length !== 3) return invalid();
      for (const vertex of vertices) values.push(Number(vertex[1]), Number(vertex[2]), Number(vertex[3]));
    }
    positions = new Float32Array(values);
  }
  if (!positions.length) return invalid(); finiteCoordinates(positions); return { format: "stl", positions, partial: false };
}
export function prepareModel(bytes: ArrayBuffer, format: ModelFormat): PreparedModel {
  if (bytes.byteLength > MAX_MODEL_BYTES) throw new RangeError("bytes");
  if (format === "glb" || format === "gltf") return gltf(bytes, format);
  return format === "obj" ? obj(bytes) : stl(bytes);
}
