import { readFile } from "node:fs/promises";
import { buildGoalBrief, validateGoalBrief } from "@vrooli/goal-brief";

const filename = process.argv[2];
if (!filename) {
  console.error("Usage: node scripts/generate-goal-brief.mjs <input.json>");
  process.exit(2);
}

let input;
try {
  input = JSON.parse(await readFile(filename, "utf8"));
} catch (error) {
  console.error(`Could not read the goal brief input JSON: ${error.message}`);
  process.exit(2);
}

if (!input || typeof input !== "object" || ["destination", "stance", "charterPath"].some((key) => typeof input[key] !== "string")) {
  console.error("Input JSON must contain string fields: destination, stance, charterPath");
  process.exit(2);
}

const draft = buildGoalBrief(input);
const errors = validateGoalBrief(draft);
if (errors.length > 0) {
  for (const error of errors) console.error(`FAIL: ${error}`);
  process.exit(1);
}

console.log(JSON.stringify(draft, null, 2));
