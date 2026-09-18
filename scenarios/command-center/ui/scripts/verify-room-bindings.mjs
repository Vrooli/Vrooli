import { readFile, readdir } from "node:fs/promises";
import { join } from "node:path";

const root = new URL("../../config/", import.meta.url);
const load = async (folder, file) => JSON.parse(await readFile(new URL(`${folder}/${file}`, root), "utf8"));
const files = (await readdir(new URL("rooms/", root))).filter((file) => file.endsWith(".json"));
const compositionEntries = await Promise.all((await readdir(new URL("compositions/", root))).filter((file) => file.endsWith(".json") && !file.endsWith(".slots.json")).map((file) => load("compositions", file)));
const compositionMap = Object.fromEntries(compositionEntries.map((entry) => [entry.id, entry]));
const signalFiles = (await readdir(new URL("signals/", root))).filter((file) => file.endsWith(".json"));
const signals = Object.fromEntries(await Promise.all(signalFiles.map(async (file) => { const entry = await load("signals", file); return [entry.id, entry]; })));
const errors = [];
for (const file of files) {
  const room = await load("rooms", file);
  const slots = compositionMap[room.composition]?.slots ?? {};
  for (const [slot, signalId] of Object.entries(room.bind ?? {})) {
    const signal = signals[signalId];
    const spec = slots[slot];
    if (!signal) errors.push(`${room.id}: ${slot} references unknown signal ${signalId}`);
    else if (spec?.shape && signal.shape !== spec.shape) errors.push(`${room.id}: ${slot} requires ${spec.shape}, ${signalId} is ${signal.shape}`);
    for (const [column, definition] of Object.entries(spec?.columns ?? {})) if (!definition.optional && !signal.columns?.[column]) errors.push(`${room.id}: ${signalId} is missing required column ${column}`);
  }
}
if (errors.length) { console.error(errors.join("\n")); process.exit(1); }
console.log(`Room bindings valid: ${files.length} rooms checked.`);
