import { readFileSync, writeFileSync } from "node:fs";
import { execFileSync } from "node:child_process";

const files = execFileSync("rg", ["--files", "library"], { cwd: new URL("..", import.meta.url).pathname })
  .toString()
  .trim()
  .split("\n")
  .filter((file) => file.endsWith("/story.json"));

const visibleExpectation = { kind: "visible", selector: "body" };
let migrated = 0;
let addedFrames = 0;

for (const file of files) {
  const contract = JSON.parse(readFileSync(file, "utf8"));
  if (contract.schemaVersion !== 4 && contract.schemaVersion !== 5) continue;
  contract.schemaVersion = 5;
  const enumFields = (contract.args?.fields ?? []).filter((field) => field.kind === "enum");
  const stories = contract.stories ?? [];
  if (stories.length === 0) continue;

  stories[0].role = "anatomy";
  if (!Array.isArray(stories[0].expect) || stories[0].expect.length === 0) {
    stories[0].expect = [visibleExpectation];
  }
  for (const story of stories) {
    if (!Array.isArray(story.expect) || story.expect.length === 0) story.expect = [visibleExpectation];
  }

  const usedIds = new Set(stories.map((story) => story.id));
  for (let index = 0; index < enumFields.length; index += 1) {
    const field = enumFields[index];
    let axisStory = stories.find((story) => story.role === "axis" && story.axis === field.path);
    if (!axisStory) {
      axisStory = stories[index + 1];
      if (!axisStory || axisStory.role === "anatomy" || axisStory.axis) {
        const base = `axis-${field.path.replaceAll(".", "-")}`;
        let id = base;
        let suffix = 2;
        while (usedIds.has(id)) id = `${base}-${suffix++}`;
        axisStory = JSON.parse(JSON.stringify(stories[0]));
        axisStory.id = id;
        axisStory.name = `${field.label || field.path} matrix`;
        stories.push(axisStory);
        usedIds.add(id);
        addedFrames += 1;
      }
    }
    axisStory.role = "axis";
    axisStory.axis = field.path;
    axisStory.covers = { ...(axisStory.covers ?? {}), [field.path]: field.options };
  }
  for (const story of stories) {
    if (!story.role) story.role = "boundary";
  }
  contract.stories = stories;
  writeFileSync(file, `${JSON.stringify(contract, null, 2)}\n`);
  migrated += 1;
}

console.log(JSON.stringify({ migrated, addedFrames }, null, 2));
