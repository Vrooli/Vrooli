import { readFileSync } from "node:fs";
import { resolve } from "node:path";

it("references every route assigned to the UI surface", () => {
  const contract = JSON.parse(
    readFileSync(
      resolve(process.cwd(), "../api/testdata/route-surface.json"),
      "utf8",
    ),
  ) as {
    routes: Array<{ method: string; path: string; surfaces: string[] }>;
  };
  const source = readFileSync(resolve(process.cwd(), "src/lib/api.ts"), "utf8");
  for (const route of contract.routes.filter((entry) =>
    entry.surfaces.includes("ui"),
  )) {
    const path = route.path
      .replace(/^\/api\/v1/, "")
      .replace(/^\/api/, "")
      .replace(/\{[^}]+\}/g, "");
    const sourcePath = path.endsWith("/operator-state")
      ? "/operator-state"
      : path;
    expect(source, `${route.method} ${route.path}`).toContain(sourcePath);
  }
});
