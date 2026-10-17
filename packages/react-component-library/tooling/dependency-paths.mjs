import { join } from "node:path";

// Compiler path discovery belongs to the package manifest, not to a hand-
// maintained list in the build script. The optional overrides are reserved
// for workspace links whose declaration files intentionally live outside
// this package's node_modules tree.
export function packageDependencyPaths(packageRoot, manifest, overrides = {}) {
  const packageNodeModules = join(packageRoot, "node_modules");
  const names = new Set([
    ...Object.keys(manifest.dependencies ?? {}),
    ...Object.keys(manifest.devDependencies ?? {}),
  ]);
  const paths = Object.fromEntries(
    [...names].map((name) => [name, [join(packageNodeModules, name)]])
  );
  if (names.has("@types/react")) {
    paths.react = [join(packageNodeModules, "@types", "react", "index.d.ts")];
    paths["react/*"] = [join(packageNodeModules, "@types", "react", "*")];
    paths["react/jsx-runtime"] = [join(packageNodeModules, "@types", "react", "jsx-runtime.d.ts")];
  }
  if (names.has("@types/react-dom")) {
    paths["react-dom"] = [join(packageNodeModules, "@types", "react-dom", "index.d.ts")];
  }
  return { ...paths, ...overrides };
}
