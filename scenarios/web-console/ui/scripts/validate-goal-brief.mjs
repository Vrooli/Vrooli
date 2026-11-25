import { readFile } from "node:fs/promises";
import { validateGoalBrief } from "@vrooli/goal-brief";

const filename = process.argv[2];
if (!filename) {
  console.error("Usage: node scripts/validate-goal-brief.mjs <draft.json>");
  process.exit(2);
}

let draft;
try {
  draft = JSON.parse(await readFile(filename, "utf8"));
} catch (error) {
  console.error(`Could not read the goal brief JSON: ${error.message}`);
  process.exit(2);
}

const required = ["destination", "stance", "charterPath", "stanceMessage", "kernel", "message"];
if (!draft || typeof draft !== "object" || required.some((key) => typeof draft[key] !== "string")) {
  console.error(`Draft JSON must contain string fields: ${required.join(", ")}`);
  process.exit(2);
}

const errors = validateGoalBrief(draft);
if (errors.length > 0) {
  for (const error of errors) console.error(`FAIL: ${error}`);
  process.exit(1);
}

console.log(`PASS: kernel=${Array.from(draft.kernel).length}/600 characters; exact stance preserved; charter path present`);
