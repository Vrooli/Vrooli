import { exportSelectorManifest } from "@vrooli/ui-selectors/export";
await exportSelectorManifest({ check: process.argv.includes("--check") });
