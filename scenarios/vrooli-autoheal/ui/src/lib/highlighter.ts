import { createHighlighter, type Highlighter, type BundledLanguage } from "shiki";

const bundledLanguages: BundledLanguage[] = ["json", "bash", "yaml", "markdown", "dockerfile"];

let highlighterInstance: Highlighter | null = null;
let highlighterPromise: Promise<Highlighter> | null = null;

export async function getHighlighter(): Promise<Highlighter> {
  if (highlighterInstance) {
    return highlighterInstance;
  }

  if (highlighterPromise) {
    return highlighterPromise;
  }

  highlighterPromise = createHighlighter({
    themes: ["github-dark"],
    langs: bundledLanguages,
  }).then((instance) => {
    highlighterInstance = instance;
    return instance;
  });

  return highlighterPromise;
}

export async function highlightCode(code: string, lang: string): Promise<string> {
  const highlighter = await getHighlighter();
  const loadedLangs = highlighter.getLoadedLanguages();

  let langToUse = lang || "text";

  if (langToUse !== "text" && !loadedLangs.includes(langToUse as BundledLanguage)) {
    try {
      await highlighter.loadLanguage(langToUse as BundledLanguage);
    } catch {
      langToUse = "text";
    }
  }

  return highlighter.codeToHtml(code, {
    lang: langToUse as BundledLanguage | "text",
    theme: "github-dark",
  });
}
