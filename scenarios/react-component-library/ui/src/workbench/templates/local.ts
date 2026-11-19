import { useEffect, useState } from "react";

import { DEFAULT_TEMPLATE, TEMPLATES } from "./library";
import { node, type Template, type TemplateNode } from "./spec";

/**
 * Browser-local templates. The studio must be able to author a composition
 * without publishing a library release, so a duplicated or new template lives
 * in localStorage until it is promoted through the governed asset flow. This
 * module is the only reader/writer of that store.
 */
export const LOCAL_TEMPLATE_KEY = "rcl.workbench.templates.v1";
const EVENT = "rcl:workbench-templates";

function isTemplate(value: unknown): value is Template {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return (
    typeof record.id === "string" &&
    typeof record.name === "string" &&
    typeof record.spec === "object" &&
    record.spec !== null &&
    typeof record.palette === "object" &&
    record.palette !== null
  );
}

export function readLocalTemplates(): Template[] {
  try {
    const raw = window.localStorage.getItem(LOCAL_TEMPLATE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isTemplate).map((template) => ({ ...template, local: true }));
  } catch {
    return [];
  }
}

function writeLocalTemplates(templates: Template[]): void {
  try {
    window.localStorage.setItem(LOCAL_TEMPLATE_KEY, JSON.stringify(templates));
    window.dispatchEvent(new Event(EVENT));
  } catch {
    /* storage can be full or disabled; the shelf simply stays in memory */
  }
}

export function upsertLocalTemplate(template: Template): Template[] {
  const next = [
    ...readLocalTemplates().filter((entry) => entry.id !== template.id),
    { ...template, local: true },
  ];
  writeLocalTemplates(next);
  return next;
}

export function removeLocalTemplate(id: string): Template[] {
  const next = readLocalTemplates().filter((entry) => entry.id !== id);
  writeLocalTemplates(next);
  return next;
}

export function localTemplateId(name: string): string {
  const base = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 32);
  return `${base || "template"}-${Date.now().toString(36)}`;
}

export function useLocalTemplates(): Template[] {
  const [templates, setTemplates] = useState<Template[]>(() => readLocalTemplates());
  useEffect(() => {
    const sync = () => setTemplates(readLocalTemplates());
    window.addEventListener(EVENT, sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener(EVENT, sync);
      window.removeEventListener("storage", sync);
    };
  }, []);
  return templates;
}

export function templateShelf(locals: Template[] = []): Template[] {
  return [...TEMPLATES, ...locals];
}

export function resolveTemplate(id: string, locals: Template[] = []): Template | undefined {
  return templateShelf(locals).find((template) => template.id === id);
}

export function cloneTemplate(template: Template, name?: string): Template {
  const stamp = Date.now().toString(36);
  const rename = (current: TemplateNode): TemplateNode => ({
    ...current,
    id: `${template.id}.${stamp}.${current.id}`,
    children: current.children?.map(rename),
  });
  return {
    ...template,
    id: `${template.id}-copy-${stamp}`,
    name: name ?? `${template.name} copy`,
    tagline: template.tagline,
    spec: rename(template.spec),
    local: true,
  };
}

/** A deliberately quiet starter: one page, one stack, one title and a slot. */
export function blankTemplate(name = "Untitled page"): Template {
  const stamp = Date.now().toString(36);
  return {
    id: `untitled-${stamp}`,
    name,
    tagline: "Blank composition",
    category: DEFAULT_TEMPLATE.category,
    slotHint: "Place the component where the story needs it",
    idea: "Start from one clear idea, then add evidence around it.",
    palette: { ...DEFAULT_TEMPLATE.palette },
    display: DEFAULT_TEMPLATE.display,
    radius: DEFAULT_TEMPLATE.radius,
    local: true,
    spec: node(
      `untitled.${stamp}.page`,
      "page",
      { pad: 40 },
      [
        node(`untitled.${stamp}.stack`, "stack", { gap: 18 }, [
          node(`untitled.${stamp}.eyebrow`, "text", {
            text: "NEW PAGE",
            variant: "eyebrow",
            tone: "accent",
          }),
          node(`untitled.${stamp}.title`, "text", {
            text: "Name the one idea this page makes inevitable.",
            variant: "display",
          }),
          node(`untitled.${stamp}.body`, "text", {
            text: "Add the supporting evidence, then place a real component in the slot.",
            variant: "body",
            tone: "muted",
          }),
          node(`untitled.${stamp}.slot`, "slot", {
            hint: "Place the component where the story needs it",
            minH: 48,
          }),
        ]),
      ],
    ),
  };
}

export const TEMPLATE_PALETTES: Array<{
  id: string;
  label: string;
  palette: Template["palette"];
}> = [
  {
    id: "nocturne",
    label: "Nocturne",
    palette: DEFAULT_TEMPLATE.palette,
  },
  {
    id: "ivory",
    label: "Ivory",
    palette: {
      bg: "#f7f3ea",
      surface: "#fffdf8",
      surfaceAlt: "#efe8da",
      ink: "#1e1a16",
      muted: "#6d6458",
      line: "#ded4c3",
      accent: "#a8452a",
      accentInk: "#fffaf4",
      accentSoft: "#f2dfd6",
      secondary: "#234c44",
    },
  },
  {
    id: "cobalt",
    label: "Cobalt",
    palette: {
      bg: "#070d1f",
      surface: "#0e1730",
      surfaceAlt: "#152243",
      ink: "#f1f5ff",
      muted: "#9fb0d4",
      line: "#24345c",
      accent: "#7aa2ff",
      accentInk: "#07101f",
      accentSoft: "#1d2f5c",
      secondary: "#6fe3c4",
    },
  },
  {
    id: "clay",
    label: "Clay",
    palette: {
      bg: "#f3e7dc",
      surface: "#fbf3ea",
      surfaceAlt: "#ead9c9",
      ink: "#2a1c14",
      muted: "#7a6151",
      line: "#dcc5ae",
      accent: "#c4562f",
      accentInk: "#fff8f2",
      accentSoft: "#f0d3c1",
      secondary: "#3d5a4c",
    },
  },
  {
    id: "grove",
    label: "Grove",
    palette: {
      bg: "#0b1510",
      surface: "#112018",
      surfaceAlt: "#172c21",
      ink: "#eef7ef",
      muted: "#9db8a5",
      line: "#254433",
      accent: "#8ede6b",
      accentInk: "#0b1510",
      accentSoft: "#22452f",
      secondary: "#e6c37a",
    },
  },
];

export function withPalette(template: Template, palette: Template["palette"]): Template {
  return { ...template, palette: { ...palette } };
}
