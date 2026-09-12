import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const uiRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const sourceRoot = path.join(uiRoot, "src");

function filesUnder(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) return filesUnder(absolute);
    return [absolute];
  });
}

test("react-vite template does not copy governed component sources", () => {
  const copied = filesUnder(sourceRoot).filter((file) => /\.(js|jsx|ts|tsx)$/.test(file)).filter((file) => fs.readFileSync(file, "utf8").includes("@vrooliComponentSource"));
  assert.deepEqual(copied, [], "template UI must link package subpaths instead of copying governed sources");
});

test("react-vite template declares the governed component package", () => {
  const packageJSON = JSON.parse(fs.readFileSync(path.join(uiRoot, "package.json"), "utf8"));
  assert.equal(packageJSON.dependencies["@vrooli/react-component-library"], "file:../../../packages/react-component-library");
});
