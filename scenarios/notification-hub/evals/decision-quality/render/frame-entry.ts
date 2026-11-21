// The decision-quality eval renders a page the way the Hub's frame does: this
// bundle exposes the UI's own sealing (document.ts) and theme reading
// (theme.ts) to the headless browser run.py drives. esbuild bundles it; it is
// never part of the UI build.
import { buildExplainerDocument, findReferences, planInlining } from "../../../ui/src/features/packet/explainer/document";
import type { ExplainerAttachment } from "../../../ui/src/features/packet/explainer/document";
import { pageColorScheme, pageTheme } from "../../../ui/src/features/packet/explainer/theme";

interface FrameFile {
  name: string;
  mediaType: string;
  sizeBytes: number;
  dataUrl: string;
}

interface FrameInput {
  html: string;
  kitCss: string;
  kitJs: string;
  bridgeJs: string;
  theme: "light" | "dark";
  files: FrameFile[];
}

function build(input: FrameInput): { srcdoc: string; scheme: string; frameTheme: string } {
  const attachments: ExplainerAttachment[] = input.files.map((file) => ({
    name: file.name,
    folder: false,
    sizeBytes: file.sizeBytes,
    mediaType: file.mediaType,
    entries: [],
  }));
  const { inline, deferred } = planInlining(findReferences(input.html, attachments));
  const byName = new Map(input.files.map((file) => [file.name, file.dataUrl]));
  const dataUrls = new Map<string, string>();
  for (const reference of inline) {
    const url = byName.get(reference.value);
    if (url) dataUrls.set(reference.value, url);
  }
  const scheme = pageColorScheme(input.html, input.kitCss);
  const srcdoc = buildExplainerDocument(input.html, {
    kitCss: input.kitCss,
    kitJs: input.kitJs,
    bridgeJs: input.bridgeJs,
    theme: input.theme,
    dataUrls,
    deferred,
  });
  return { srcdoc, scheme, frameTheme: pageTheme(scheme, input.theme) };
}

(window as unknown as { HubFrame: unknown }).HubFrame = { build };
