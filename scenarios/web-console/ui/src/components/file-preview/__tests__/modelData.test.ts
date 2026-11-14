import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { prepareModel } from "../prepareModel";
import { MAX_MODEL_BYTES, modelCameraDistance, modelFormat } from "../modelData";
const bytes = (text: string) => new TextEncoder().encode(text).buffer;
const embeddedPng = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j0YQAAAAASUVORK5CYII=";
function document() {
  const buffer = new ArrayBuffer(36), f = new Float32Array(buffer); f.set([0, 0, 0, 1, 0, 0, 0, 1, 0]);
  return { asset: { version: "2.0" }, scene: 0, scenes: [{ nodes: [0] }], nodes: [{ mesh: 0 }], meshes: [{ primitives: [{ attributes: { POSITION: 0 } }] }], buffers: [{ byteLength: 36, uri: `data:application/octet-stream;base64,${Buffer.from(buffer).toString("base64")}` }], bufferViews: [{ buffer: 0, byteLength: 36 }], accessors: [{ bufferView: 0, componentType: 5126, count: 3, type: "VEC3" }] };
}
const parse = (value: unknown) => prepareModel(bytes(JSON.stringify(value)), "gltf");
function glbDocument(value: unknown): ArrayBuffer {
  const json = new TextEncoder().encode(JSON.stringify(value)), length = Math.ceil(json.length / 4) * 4;
  const output = new ArrayBuffer(20 + length), header = new DataView(output);
  header.setUint32(0, 0x46546c67, true); header.setUint32(4, 2, true); header.setUint32(8, output.byteLength, true);
  header.setUint32(12, length, true); header.setUint32(16, 0x4e4f534a, true);
  new Uint8Array(output).fill(32, 20); new Uint8Array(output).set(json, 20);
  return output;
}
describe("bounded 3D model preflight", () => {
  it.each(["glb", "gltf", "obj", "stl"] as const)("opens the original %s sculpture fixture", format => {
    const file = readFileSync(`../fixtures/file-preview/orbital-study.${format}`);
    const data = prepareModel(Uint8Array.from(file).buffer, format); expect(data.format).toBe(format);
    if ("positions" in data) expect(data.positions.length / 9).toBe(4768);
  });
  it("preserves geometry and uses the correct framing for narrow stages", () => {
    expect(parse(document()).format).toBe("gltf"); expect(modelCameraDistance(1, 0.5)).toBeGreaterThan(modelCameraDistance(1, 2));
    expect(modelFormat("model.GLB")).toBe("glb"); expect(modelFormat("unknown.zip")).toBeNull();
  });
  it.each(["https://attacker.test/model.bin", "../secrets.bin", "file:///etc/passwd", "blob:https://local/a", "//attacker.test/a"])("rejects external buffer %s", uri => {
    const value = document(); value.buffers[0] = { byteLength: 36, uri }; expect(() => parse(value)).toThrow("external");
  });
  it("rejects external textures and unsupported extensions even when advisory lists omit them", () => {
    expect(() => parse({ ...document(), images: [{ uri: "https://attacker.test/a.png" }] })).toThrow("external");
    expect(() => parse({ ...document(), nodes: [{ mesh: 0, extensions: { EXT_mesh_gpu_instancing: {} } }] })).toThrow("unsupported");
    expect(() => parse({ ...document(), extensionsRequired: ["KHR_draco_mesh_compression"] })).toThrow("unsupported");
    expect(() => parse({ ...document(), skins: [{}] })).toThrow("unsupported");
  });
  it.each(["glb", "gltf"] as const)("preserves authored Blender transmission materials in %s", format => {
    const material = { extensions: { KHR_materials_transmission: { transmissionFactor: 0.82 } } };
    const input = { ...document(), extensionsUsed: ["KHR_materials_transmission"], extensionsRequired: ["KHR_materials_transmission"], materials: [material], meshes: [{ primitives: [{ attributes: { POSITION: 0 }, material: 0 }] }] };
    const prepared = prepareModel(format === "glb" ? glbDocument(input) : bytes(JSON.stringify(input)), format);
    if (!("bytes" in prepared)) throw Error("Expected glTF");
    const json = format === "glb" ? new Uint8Array(prepared.bytes, 20, new DataView(prepared.bytes).getUint32(12, true)) : new Uint8Array(prepared.bytes);
    expect(JSON.parse(new TextDecoder().decode(json))).toMatchObject({ materials: [material], extensionsRequired: ["KHR_materials_transmission"] });
  });
  it.each(["glb", "gltf"] as const)("preserves imported clearcoat/specular materials in %s", format => {
    const extensions = { KHR_materials_clearcoat: { clearcoatFactor: 1, clearcoatRoughnessFactor: 0.06 }, KHR_materials_specular: { specularFactor: 0.1, specularColorFactor: [0.8, 0.9, 1] } };
    const input = { ...document(), materials: [{ extensions }], meshes: [{ primitives: [{ attributes: { POSITION: 0 }, material: 0 }] }] };
    const prepared = prepareModel(format === "glb" ? glbDocument(input) : bytes(JSON.stringify(input)), format);
    if (!("bytes" in prepared)) throw Error("Expected glTF");
    const json = format === "glb" ? new Uint8Array(prepared.bytes, 20, new DataView(prepared.bytes).getUint32(12, true)) : new Uint8Array(prepared.bytes);
    expect(JSON.parse(new TextDecoder().decode(json))).toMatchObject({ materials: [{ extensions }] });
  });
  it("admits optional transmission textures through the existing embedded-resource checks", () => {
    const uri = embeddedPng;
    const material = { extensions: { KHR_materials_transmission: { transmissionTexture: { index: 0 } } } };
    const input = { ...document(), materials: [material], images: [{ uri }], textures: [{ source: 0 }] };
    const prepared = parse(input);
    if (!("bytes" in prepared)) throw Error("Expected glTF");
    expect(prepared.format).toBe("gltf"); expect(prepared.uris).toContain(uri);
    expect(() => parse({ ...input, images: [{ uri: "https://attacker.test/glass.png" }] })).toThrow("external");
    expect(() => parse({ ...input, textures: [{ source: 1 }] })).toThrow("invalid");
  });
  it.each([-0.1, 1.1, "0.5", null])("rejects invalid transmission factor %s", factor => {
    expect(() => parse({ ...document(), materials: [{ extensions: { KHR_materials_transmission: { transmissionFactor: factor } } }] })).toThrow("invalid");
  });
  it.each([{}, { index: -1 }, { index: 0 }, { index: 0.5 }])("rejects invalid or missing transmission texture references %j", texture => {
    expect(() => parse({ ...document(), materials: [{ extensions: { KHR_materials_transmission: { transmissionTexture: texture } } }] })).toThrow("invalid");
  });
  it("rejects malformed transmission payloads and conflicting unlit materials", () => {
    for (const payload of [null, [], "glass"]) expect(() => parse({ ...document(), materials: [{ extensions: { KHR_materials_transmission: payload } }] })).toThrow("invalid");
    expect(() => parse({ ...document(), materials: [{ extensions: { KHR_materials_transmission: {}, KHR_materials_unlit: {} } }] })).toThrow("invalid");
  });
  it.each([
    ["KHR_materials_clearcoat", "clearcoatFactor"],
    ["KHR_materials_clearcoat", "clearcoatRoughnessFactor"],
    ["KHR_materials_specular", "specularFactor"],
  ])("bounds %s %s factors while preserving the endpoints", (extension, field) => {
    const input = (value: unknown) => ({ ...document(), materials: [{ extensions: { [extension]: { [field]: value } } }] });
    for (const value of [0, 1]) expect(parse(input(value)).format).toBe("gltf");
    for (const value of [-0.1, 1.1, "0.5", null]) expect(() => parse(input(value))).toThrow("invalid");
  });
  it.each([
    ["KHR_materials_clearcoat", "clearcoatTexture"],
    ["KHR_materials_clearcoat", "clearcoatRoughnessTexture"],
    ["KHR_materials_clearcoat", "clearcoatNormalTexture"],
    ["KHR_materials_specular", "specularTexture"],
    ["KHR_materials_specular", "specularColorTexture"],
  ])("validates %s %s through the bounded embedded-texture path", (extension, field) => {
    const uri = embeddedPng;
    const input = (texture: unknown) => ({ ...document(), materials: [{ extensions: { [extension]: { [field]: texture } } }], images: [{ uri }], textures: [{ source: 0 }] });
    expect(parse(input({ index: 0, texCoord: 0 })).format).toBe("gltf");
    for (const texture of [null, {}, { index: -1 }, { index: 1 }, { index: 0.5 }, { index: 0, texCoord: -1 }]) expect(() => parse(input(texture))).toThrow("invalid");
    expect(() => parse({ ...input({ index: 0 }), images: [{ uri: "https://attacker.test/coating.png" }] })).toThrow("external");
    const png = new Uint8Array(32), view = new DataView(png.buffer); view.setUint32(0, 0x89504e47); view.setUint32(4, 0x0d0a1a0a); view.setUint32(16, 100000); view.setUint32(20, 100000);
    expect(() => parse({ ...input({ index: 0 }), images: [{ uri: `data:image/png;base64,${Buffer.from(png).toString("base64")}` }] })).toThrow(RangeError);
  });
  it("validates specular RGB shape and channel range", () => {
    const input = (color: unknown) => ({ ...document(), materials: [{ extensions: { KHR_materials_specular: { specularColorFactor: color } } }] });
    for (const color of [[0, 0.5, 1], [2, 3, 1]]) expect(parse(input(color)).format).toBe("gltf");
    for (const color of [null, "white", [], [1, 1], [1, 1, 1, 1], [-0.1, 0, 1], [0, 0, 1e13], [0, "0.5", 1]]) expect(() => parse(input(color))).toThrow();
  });
  it.each(["KHR_materials_clearcoat", "KHR_materials_specular"])("rejects malformed %s payloads and unlit conflicts", extension => {
    for (const payload of [null, [], "coat"]) expect(() => parse({ ...document(), materials: [{ extensions: { [extension]: payload } }] })).toThrow("invalid");
    expect(() => parse({ ...document(), materials: [{ extensions: { [extension]: {}, KHR_materials_unlit: {} } }] })).toThrow("invalid");
    expect(parse({ ...document(), extensionsRequired: [extension], materials: [{ extensions: { [extension]: {} } }] }).format).toBe("gltf");
  });
  it("validates clearcoat normal scale with otherwise valid embedded resources", () => {
    const input = (scale: unknown) => ({ ...document(), images: [{ uri: embeddedPng }], textures: [{ source: 0 }], materials: [{ extensions: { KHR_materials_clearcoat: { clearcoatNormalTexture: { index: 0, scale } } } }] });
    for (const scale of [-1, 0, 1]) expect(parse(input(scale)).format).toBe("gltf");
    for (const scale of [null, "large"]) expect(() => parse(input(scale))).toThrow("invalid");
  });
  it("rejects sparse and oversized accessors before allocating geometry", () => {
    const value = document(); expect(() => parse({ ...value, accessors: [{ ...value.accessors[0], sparse: {} }] })).toThrow("unsupported");
    expect(() => parse({ ...value, accessors: [{ ...value.accessors[0], count: 1000000000 }] })).toThrow();
    expect(() => prepareModel(new ArrayBuffer(MAX_MODEL_BYTES + 1), "stl")).toThrow(RangeError);
  });
  it("rejects cyclic, duplicate-parent and excessively deep graphs", () => {
    expect(() => parse({ ...document(), nodes: [{ children: [0] }] })).toThrow("invalid");
    expect(() => parse({ ...document(), nodes: [{ children: [2] }, { children: [2] }, {}] })).toThrow("invalid");
    expect(() => parse({ ...document(), nodes: Array.from({ length: 70 }, (_, index) => index === 69 ? {} : { children: [index + 1] }) })).toThrow(RangeError);
  });
  it("rejects geometry instance multiplication and malformed accessor views", () => {
    expect(() => parse({ ...document(), nodes: Array.from({ length: 257 }, () => ({ mesh: 0 })) })).toThrow(RangeError);
    expect(() => parse({ ...document(), bufferViews: [{ buffer: 0, byteOffset: 30, byteLength: 36 }] })).toThrow("invalid");
    expect(() => parse({ ...document(), bufferViews: [{ buffer: 0, byteLength: 36, byteStride: 4 }] })).toThrow("invalid");
  });
  it("rejects non-finite binary positions and image decode bombs", () => {
    const value = document(); const f = new Float32Array([NaN, 0, 0, 1, 0, 0, 0, 1, 0]); value.buffers[0] = { byteLength: 36, uri: `data:application/octet-stream;base64,${Buffer.from(f.buffer).toString("base64")}` }; expect(() => parse(value)).toThrow("invalid");
    const png = new Uint8Array(32), view = new DataView(png.buffer); view.setUint32(0, 0x89504e47); view.setUint32(4, 0x0d0a1a0a); view.setUint32(16, 100000); view.setUint32(20, 100000);
    expect(() => parse({ ...document(), images: [{ uri: `data:image/png;base64,${Buffer.from(png).toString("base64")}` }] })).toThrow(RangeError);
  });
  it("rejects malformed GLB chunk lengths rather than trusting its header", () => {
    const glb = new ArrayBuffer(24), v = new DataView(glb); v.setUint32(0, 0x46546c67, true); v.setUint32(4, 2, true); v.setUint32(8, 24, true); v.setUint32(12, 1000000, true); v.setUint32(16, 0x4e4f534a, true);
    expect(() => prepareModel(glb, "glb")).toThrow("invalid");
  });
  it("triangulates negative OBJ indices without loading material libraries", () => {
    const data = prepareModel(bytes("mtllib https://attacker.test/a.mtl\nv 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nf -4 -3 -2 -1\n"), "obj");
    expect(data).toMatchObject({ partial: true }); if (!("positions" in data)) throw Error("Expected geometry"); expect(data.positions).toHaveLength(18);
    expect(() => prepareModel(bytes("v 0 0 NaN\nf 1 1 1"), "obj")).toThrow("invalid");
    expect(() => prepareModel(bytes("v 0 0 0\nf 1 2 3"), "obj")).toThrow("invalid");
  });
  it("retains authored OBJ normals and ignores inline comments", () => {
    const data = prepareModel(bytes("v 0 0 0\nv 1 0 0\nv 0 1 0\nvn 0 0 1\nf 1//1 2//1 3//1 # triangle\n"), "obj");
    expect(data).toMatchObject({ normals: new Float32Array([0, 0, 1, 0, 0, 1, 0, 0, 1]) });
  });
  it("removes clone-amplified publisher metadata and inactive scenes before rendering", () => {
    const input = { ...document(), extras: { unneeded: "x".repeat(10000) }, animations: [{ name: "unused" }], scenes: [{ nodes: [0] }, { nodes: [0] }], meshes: [{ primitives: [{ attributes: { POSITION: 0 } }], extras: { huge: "x".repeat(10000) } }] };
    const data = parse(input); if (!("bytes" in data)) throw Error("Expected glTF");
    const output = JSON.parse(new TextDecoder().decode(data.bytes)) as { extras?: unknown; animations?: unknown; scenes: unknown[]; meshes: { extras?: unknown }[] };
    expect(output.extras).toBeUndefined(); expect(output.animations).toBeUndefined(); expect(output.scenes).toHaveLength(1); expect(output.meshes[0]?.extras).toBeUndefined();
  });
  it("rejects out-of-range geometry indices and oversized cloneable names", () => {
    const input = document(), data = new ArrayBuffer(48); new Float32Array(data, 0, 9).set([0, 0, 0, 1, 0, 0, 0, 1, 0]); new Uint32Array(data, 36, 3).set([0, 1, 500]);
    const indexed = { ...input, buffers: [{ byteLength: 48, uri: `data:application/octet-stream;base64,${Buffer.from(data).toString("base64")}` }], bufferViews: [...input.bufferViews, { buffer: 0, byteOffset: 36, byteLength: 12 }], accessors: [...input.accessors, { bufferView: 1, componentType: 5125, count: 3, type: "SCALAR" }], meshes: [{ primitives: [{ attributes: { POSITION: 0 }, indices: 1 }] }] };
    expect(() => parse(indexed)).toThrow("invalid");
    expect(() => parse({ ...input, nodes: [{ mesh: 0, name: "x".repeat(1025) }] })).toThrow(RangeError);
  });
  it("charges texture allocation for repeated image references", () => {
    const png = new Uint8Array(32), v = new DataView(png.buffer); v.setUint32(0, 0x89504e47); v.setUint32(4, 0x0d0a1a0a); v.setUint32(16, 2048); v.setUint32(20, 2048);
    const image = { uri: `data:image/png;base64,${Buffer.from(png).toString("base64")}` };
    expect(() => parse({ ...document(), images: [image], textures: Array.from({ length: 5 }, () => ({ source: 0 })) })).toThrow(RangeError);
  });
  it("bounds JSON syntax before materializing excessive objects", () => {
    expect(() => prepareModel(bytes('{"nested":' + '['.repeat(50) + '0' + ']'.repeat(50) + '}'), "gltf")).toThrow(RangeError);
    expect(() => prepareModel(bytes('{"values":[' + '0,'.repeat(200001) + '0]}'), "gltf")).toThrow(RangeError);
  });
  it("reads ASCII STL and rejects empty/incomplete/non-finite facets", () => {
    const text = "solid tri\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 1 0 0\nvertex 0 1 0\nendloop\nendfacet\nendsolid tri";
    const data = prepareModel(bytes(text), "stl"); if (!("positions" in data)) throw Error("Expected geometry"); expect([...data.positions]).toEqual([0, 0, 0, 1, 0, 0, 0, 1, 0]);
    expect(() => prepareModel(bytes("solid empty\nendsolid"), "stl")).toThrow("invalid");
    expect(() => prepareModel(bytes(text.replace("vertex 0 1 0", "")), "stl")).toThrow("invalid");
    expect(() => prepareModel(bytes(text.replace("vertex 0 1 0", "vertex 0 1 Infinity")), "stl")).toThrow("invalid");
  });
});
