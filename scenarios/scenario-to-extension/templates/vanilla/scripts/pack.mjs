// Package dist/ as a distributable extension .zip.
//
// Written against Node's zlib rather than the `zip` binary: `zip` is absent on
// Windows and is not something a generated extension should require. Adding an
// archive library would put a dependency on every extension this template
// generates, so the ~60 lines of container format live here instead.
import { deflateRawSync } from "node:zlib";
import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const dist = join(packageRoot, "dist");
const archiveName = process.argv[2];

if (!archiveName) {
  console.error("Usage: node scripts/pack.mjs <archive-name.zip>");
  process.exit(1);
}

const crcTable = (() => {
  const table = new Int32Array(256);
  for (let i = 0; i < 256; i++) {
    let c = i;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    table[i] = c;
  }
  return table;
})();

function crc32(buffer) {
  let crc = -1;
  for (const byte of buffer) crc = (crc >>> 8) ^ crcTable[(crc ^ byte) & 0xff];
  return (crc ^ -1) >>> 0;
}

function walk(dir) {
  return readdirSync(dir).flatMap((name) => {
    const full = join(dir, name);
    return statSync(full).isDirectory() ? walk(full) : [full];
  });
}

// MS-DOS date/time, as the ZIP container stores it.
const now = new Date();
const dosTime = ((now.getHours() << 11) | (now.getMinutes() << 5) | (now.getSeconds() / 2)) & 0xffff;
const dosDate = (((now.getFullYear() - 1980) << 9) | ((now.getMonth() + 1) << 5) | now.getDate()) & 0xffff;

const files = walk(dist).sort();
if (files.length === 0) {
  console.error(`Nothing to pack: ${dist} is empty. Run the build first.`);
  process.exit(1);
}

const local = [];
const central = [];
let offset = 0;

for (const file of files) {
  // ZIP paths are always forward-slashed, whatever the host separator is.
  const name = Buffer.from(relative(dist, file).split(sep).join("/"), "utf8");
  const raw = readFileSync(file);
  const deflated = deflateRawSync(raw);
  // Storing beats deflating when compression does not pay for itself.
  const useDeflate = deflated.length < raw.length;
  const body = useDeflate ? deflated : raw;
  const method = useDeflate ? 8 : 0;
  const crc = crc32(raw);

  const localHeader = Buffer.alloc(30);
  localHeader.writeUInt32LE(0x04034b50, 0);
  localHeader.writeUInt16LE(20, 4); // version needed
  localHeader.writeUInt16LE(0, 6); // flags
  localHeader.writeUInt16LE(method, 8);
  localHeader.writeUInt16LE(dosTime, 10);
  localHeader.writeUInt16LE(dosDate, 12);
  localHeader.writeUInt32LE(crc, 14);
  localHeader.writeUInt32LE(body.length, 18);
  localHeader.writeUInt32LE(raw.length, 22);
  localHeader.writeUInt16LE(name.length, 26);
  localHeader.writeUInt16LE(0, 28); // extra length
  local.push(localHeader, name, body);

  const centralHeader = Buffer.alloc(46);
  centralHeader.writeUInt32LE(0x02014b50, 0);
  centralHeader.writeUInt16LE(20, 4); // version made by
  centralHeader.writeUInt16LE(20, 6); // version needed
  centralHeader.writeUInt16LE(0, 8); // flags
  centralHeader.writeUInt16LE(method, 10);
  centralHeader.writeUInt16LE(dosTime, 12);
  centralHeader.writeUInt16LE(dosDate, 14);
  centralHeader.writeUInt32LE(crc, 16);
  centralHeader.writeUInt32LE(body.length, 20);
  centralHeader.writeUInt32LE(raw.length, 24);
  centralHeader.writeUInt16LE(name.length, 28);
  centralHeader.writeUInt16LE(0, 30); // extra
  centralHeader.writeUInt16LE(0, 32); // comment
  centralHeader.writeUInt16LE(0, 34); // disk number
  centralHeader.writeUInt16LE(0, 36); // internal attrs
  centralHeader.writeUInt32LE(0o644 << 16, 38); // external attrs
  centralHeader.writeUInt32LE(offset, 42);
  central.push(centralHeader, name);

  offset += localHeader.length + name.length + body.length;
}

const centralBuffer = Buffer.concat(central);
const end = Buffer.alloc(22);
end.writeUInt32LE(0x06054b50, 0);
end.writeUInt16LE(0, 4); // disk
end.writeUInt16LE(0, 6); // central dir start disk
end.writeUInt16LE(files.length, 8);
end.writeUInt16LE(files.length, 10);
end.writeUInt32LE(centralBuffer.length, 12);
end.writeUInt32LE(offset, 16);
end.writeUInt16LE(0, 20); // comment length

const archive = join(packageRoot, archiveName);
writeFileSync(archive, Buffer.concat([...local, centralBuffer, end]));
console.log(`Packed ${files.length} file(s) into ${archiveName}`);
