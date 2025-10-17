import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { BoxGeometry } from 'three';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '..');
const publicDir = path.resolve(projectRoot, 'public', 'prototype');

fs.mkdirSync(publicDir, { recursive: true });

function createBaseGeometry() {
  const geometry = new BoxGeometry(1, 1, 1);
  const positionAttr = geometry.getAttribute('position');
  const normalAttr = geometry.getAttribute('normal');
  const indexAttr = geometry.getIndex();

  const positions = Buffer.from(positionAttr.array.buffer, positionAttr.array.byteOffset, positionAttr.array.byteLength);
  const normals = Buffer.from(normalAttr.array.buffer, normalAttr.array.byteOffset, normalAttr.array.byteLength);
  const indices = Buffer.from(indexAttr.array.buffer, indexAttr.array.byteOffset, indexAttr.array.byteLength);

  const buffer = Buffer.concat([positions, normals, indices]);

  const positionOffset = 0;
  const normalOffset = positions.length;
  const indexOffset = positions.length + normals.length;

  const positionCount = positionAttr.count;

  const computeMinMax = () => {
    const arr = positionAttr.array;
    const min = [Number.POSITIVE_INFINITY, Number.POSITIVE_INFINITY, Number.POSITIVE_INFINITY];
    const max = [Number.NEGATIVE_INFINITY, Number.NEGATIVE_INFINITY, Number.NEGATIVE_INFINITY];
    for (let i = 0; i < arr.length; i += 3) {
      const x = arr[i];
      const y = arr[i + 1];
      const z = arr[i + 2];
      if (x < min[0]) min[0] = x;
      if (y < min[1]) min[1] = y;
      if (z < min[2]) min[2] = z;
      if (x > max[0]) max[0] = x;
      if (y > max[1]) max[1] = y;
      if (z > max[2]) max[2] = z;
    }
    return { min, max };
  };

  const { min, max } = computeMinMax();

  const accessors = [
    {
      bufferView: 0,
      componentType: 5126,
      count: positionCount,
      type: 'VEC3',
      min,
      max,
    },
    {
      bufferView: 1,
      componentType: 5126,
      count: normalAttr.count,
      type: 'VEC3',
    },
    {
      bufferView: 2,
      componentType: 5123,
      count: indexAttr.count,
      type: 'SCALAR',
    },
  ];

  const bufferViews = [
    {
      buffer: 0,
      byteOffset: positionOffset,
      byteLength: positions.length,
      target: 34962,
    },
    {
      buffer: 0,
      byteOffset: normalOffset,
      byteLength: normals.length,
      target: 34962,
    },
    {
      buffer: 0,
      byteOffset: indexOffset,
      byteLength: indices.length,
      target: 34963,
    },
  ];

  return { buffer, bufferViews, accessors };
}

function writeGLB(gltf, binaryBuffer, outputPath) {
  gltf.buffers = [{ byteLength: binaryBuffer.length }];

  const jsonBuffer = Buffer.from(JSON.stringify(gltf), 'utf8');
  const jsonPadding = (4 - (jsonBuffer.length % 4)) % 4;
  const binPadding = (4 - (binaryBuffer.length % 4)) % 4;

  const totalLength = 12 + 8 + jsonBuffer.length + jsonPadding + 8 + binaryBuffer.length + binPadding;
  const glb = Buffer.alloc(totalLength);

  let offset = 0;
  glb.writeUInt32LE(0x46546c67, offset); offset += 4;
  glb.writeUInt32LE(2, offset); offset += 4;
  glb.writeUInt32LE(totalLength, offset); offset += 4;

  glb.writeUInt32LE(jsonBuffer.length + jsonPadding, offset); offset += 4;
  glb.writeUInt32LE(0x4E4F534A, offset); offset += 4;
  jsonBuffer.copy(glb, offset);
  offset += jsonBuffer.length;
  if (jsonPadding) {
    glb.fill(0x20, offset, offset + jsonPadding);
    offset += jsonPadding;
  }

  glb.writeUInt32LE(binaryBuffer.length + binPadding, offset); offset += 4;
  glb.writeUInt32LE(0x004E4942, offset); offset += 4;
  binaryBuffer.copy(glb, offset);
  offset += binaryBuffer.length;
  if (binPadding) {
    glb.fill(0x00, offset, offset + binPadding);
  }

  fs.writeFileSync(outputPath, glb);
}

function createRoomBlockout() {
  const { buffer, bufferViews, accessors } = createBaseGeometry();

  const materials = [
    {
      name: 'Floor',
      pbrMetallicRoughness: {
        baseColorFactor: [0.86, 0.92, 0.98, 1],
        metallicFactor: 0,
        roughnessFactor: 0.9,
      },
    },
    {
      name: 'Wall',
      pbrMetallicRoughness: {
        baseColorFactor: [0.88, 0.90, 0.99, 1],
        metallicFactor: 0,
        roughnessFactor: 0.85,
      },
    },
  ];

  const meshes = [
    {
      name: 'CubeFloor',
      primitives: [
        {
          attributes: { POSITION: 0, NORMAL: 1 },
          indices: 2,
          material: 0,
        },
      ],
    },
    {
      name: 'CubeWall',
      primitives: [
        {
          attributes: { POSITION: 0, NORMAL: 1 },
          indices: 2,
          material: 1,
        },
      ],
    },
  ];

  const nodes = [
    { mesh: 0, translation: [0, -1.1, 0], scale: [6.5, 0.15, 6.5], name: 'Floor' },
    { mesh: 1, translation: [0, 0.5, -3.25], scale: [6.5, 3.2, 0.12], name: 'BackWall' },
    { mesh: 1, translation: [-3.25, 0.5, 0], scale: [0.12, 3.2, 6.5], name: 'LeftWall' },
    { mesh: 1, translation: [3.25, 0.3, -1.45], scale: [0.12, 2.6, 3.6], name: 'RightWall' },
    { mesh: 1, translation: [0, 1.6, -3.2], scale: [2.4, 0.25, 0.16], name: 'WindowLintel' },
    { mesh: 1, translation: [0, 0.3, -3.2], scale: [2.4, 0.2, 0.16], name: 'WindowSill' },
  ];

  const gltf = {
    asset: { version: '2.0', generator: 'blockout-script' },
    scenes: [{ nodes: nodes.map((_, idx) => idx) }],
    nodes,
    meshes,
    materials,
    accessors,
    bufferViews,
  };

  writeGLB(gltf, buffer, path.join(publicDir, 'room_blockout.glb'));
}

function createAccessoriesBlockout() {
  const { buffer, bufferViews, accessors } = createBaseGeometry();

  const materials = [
    {
      name: 'BedFrame',
      pbrMetallicRoughness: {
        baseColorFactor: [0.99, 0.90, 0.54, 1],
        metallicFactor: 0,
        roughnessFactor: 0.7,
      },
    },
    {
      name: 'Mattress',
      pbrMetallicRoughness: {
        baseColorFactor: [0.98, 0.81, 0.91, 1],
        metallicFactor: 0,
        roughnessFactor: 0.8,
      },
    },
    {
      name: 'Bookshelf',
      pbrMetallicRoughness: {
        baseColorFactor: [0.75, 0.95, 0.39, 1],
        metallicFactor: 0,
        roughnessFactor: 0.6,
      },
    },
    {
      name: 'Desk',
      pbrMetallicRoughness: {
        baseColorFactor: [0.94, 0.67, 0.94, 1],
        metallicFactor: 0,
        roughnessFactor: 0.6,
      },
    },
  ];

  const meshes = materials.map((_, index) => ({
    name: `CubeMaterial${index}`,
    primitives: [
      {
        attributes: { POSITION: 0, NORMAL: 1 },
        indices: 2,
        material: index,
      },
    ],
  }));

  const nodes = [
    { mesh: 0, translation: [-1.6, -0.95, -1.2], scale: [2.4, 0.35, 1.6], name: 'BedBase' },
    { mesh: 1, translation: [-1.6, -0.65, -1.2], scale: [2.3, 0.25, 1.5], name: 'BedMattress' },
    { mesh: 2, translation: [2.5, -0.1, -1.6], scale: [0.6, 2.2, 1.2], name: 'Bookshelf' },
    { mesh: 3, translation: [1.4, -0.85, 1.4], scale: [1.6, 0.75, 0.6], name: 'Desk' },
  ];

  const gltf = {
    asset: { version: '2.0', generator: 'blockout-script' },
    scenes: [{ nodes: nodes.map((_, idx) => idx) }],
    nodes,
    meshes,
    materials,
    accessors,
    bufferViews,
  };

  writeGLB(gltf, buffer, path.join(publicDir, 'accessories_blockout.glb'));
}

createRoomBlockout();
createAccessoriesBlockout();

console.log('Generated prototype blockouts in', publicDir);
