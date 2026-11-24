import test from "node:test";
import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import { contrastRatio, auditThemeContrast } from "./theme-contrast-core.mjs";

test("uses the WCAG relative-luminance ratio", () => {
  assert.equal(Number(contrastRatio("#fff", "#000").toFixed(2)), 21);
  assert.ok(contrastRatio("#777", "#080808") < 4.5);
});

test("audits every active board room theme and fails an unreadable text token", async () => {
  const config = new URL("../../config/", import.meta.url);
  async function entries(path) {
    const root = new URL(path, config);
    return Promise.all((await readdir(root)).filter((file) => file.endsWith(".json")).map(async (file) => JSON.parse(await readFile(new URL(file, root), "utf8"))));
  }
  const [board, rooms, themes] = await Promise.all([
    JSON.parse(await readFile(new URL("board.json", config), "utf8")),
    entries("rooms/"),
    entries("themes/"),
  ]);
  const audit = auditThemeContrast({ board, rooms, themes });
  assert.equal(audit.length, 6);
  assert.ok(audit.every((theme) => theme.pairs.length === 176));
  assert.ok(audit.every((theme) => theme.failures.length === 0));
  assert.ok(audit.every((theme) => theme.minimum >= 4.5));

  const lowContrast = themes.map((theme) => theme.id === "ground-control"
    ? { ...theme, tokens: { ...theme.tokens, "--color-muted-foreground": "#777777" } }
    : theme);
  const broken = auditThemeContrast({ board, rooms, themes: lowContrast });
  assert.ok(broken.find((theme) => theme.theme === "ground-control").failures.length > 0);
});
