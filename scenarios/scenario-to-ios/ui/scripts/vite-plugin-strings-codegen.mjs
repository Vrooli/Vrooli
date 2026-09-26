/**
 * Vite plugin: keep `src/consts/strings.generated.ts` in sync with
 * `src/i18n/locales/en.json` automatically.
 *
 * Hooks:
 *   - `buildStart`: regenerate before any module is resolved. Covers prod
 *     builds (`vite build`), test runs (`vitest`), and cold dev start.
 *   - `configureServer` + watcher: regenerate on every change to en.json
 *     during `vite dev`. The HMR pipeline picks up the changed
 *     `strings.generated.ts` on its own and updates consumers.
 *
 * All actual codegen logic lives in `gen-strings.mjs`. This plugin is
 * deliberately thin so the CLI (`pnpm strings:gen`/`strings:check`) and the
 * build path share one implementation — change codegen behaviour there, not
 * here.
 */
import { SOURCE_PATH, writeIfChanged } from "./gen-strings.mjs";

export default function stringsCodegen() {
  return {
    name: "vrooli-strings-codegen",
    buildStart() {
      writeIfChanged();
    },
    configureServer(server) {
      writeIfChanged();
      server.watcher.add(SOURCE_PATH);
    },
    handleHotUpdate({ file }) {
      if (file === SOURCE_PATH) writeIfChanged();
    },
  };
}
