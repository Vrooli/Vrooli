import test from "node:test";
import assert from "node:assert/strict";
import { packageDependencyPaths } from "./dependency-paths.mjs";

test("new manifest dependencies resolve without changing the build script", () => {
  const paths = packageDependencyPaths("/tmp/rcl", {
    dependencies: { "new-renderer": "^1.0.0" },
    devDependencies: { "@types/react": "^19.0.0" },
  });
  assert.deepEqual(paths["new-renderer"], ["/tmp/rcl/node_modules/new-renderer"]);
  assert.deepEqual(paths.react, ["/tmp/rcl/node_modules/@types/react/index.d.ts"]);
});
