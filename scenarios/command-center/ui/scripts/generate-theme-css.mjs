import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const uiDir = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const themesDir = path.join(uiDir, "../config/themes");
const output = path.join(uiDir, "src/themes/catalog.generated.css");
let css = "/* Generated from config/themes/*.json; do not hand edit. */\n";
for (const file of fs.readdirSync(themesDir).filter((entry) => entry.endsWith(".json")).sort()) {
  const theme = JSON.parse(fs.readFileSync(path.join(themesDir, file), "utf8"));
  css += `[data-theme="${theme.id}"] {\n`;
  for (const [name, value] of Object.entries(theme.tokens ?? {})) css += `  ${name}: ${value};\n`;
  css += "}\n";
}
fs.writeFileSync(output, css);
