import { node, slotNode, type Template } from "./spec";

/** A neutral starting composition for new workspaces and HTML import defaults. */
export const DEFAULT_TEMPLATE: Template = {
  id: "blank-workspace",
  name: "Untitled page",
  tagline: "A clean page to build on.",
  category: "operations",
  idea: "Start with one clear message and add the evidence it needs.",
  slotHint: "Place a component where the page needs it",
  display: "serif",
  radius: 16,
  palette: {
    bg: "#111318",
    surface: "#191c23",
    surfaceAlt: "#222630",
    ink: "#f4f5f7",
    muted: "#9ca3af",
    line: "#343945",
    accent: "#c7f36b",
    accentInk: "#18210b",
    accentSoft: "rgba(199,243,107,.14)",
    secondary: "#9bb6ff",
  },
  spec: node("workspace.page", "page", { pad: 32 }, [
    node("workspace.intro", "stack", { gap: 12 }, [
      node("workspace.eyebrow", "text", {
        text: "A NEW PAGE",
        variant: "eyebrow",
        tone: "accent",
      }),
      node("workspace.title", "text", {
        text: "Start with one clear idea.",
        variant: "display",
      }),
      node("workspace.body", "text", {
        text: "Shape the layout, add a real component, and save each state in your workspace.",
        variant: "body",
        tone: "muted",
      }),
    ]),
    node("workspace.section", "card", { pad: 20 }, [
      node("workspace.sectionTitle", "text", {
        text: "Build the next section",
        variant: "subtitle",
      }),
      slotNode("workspace.component", {
        hint: "Place a component where the page needs it",
        minH: 80,
        role: "hero",
        accepts: ["action", "button", "input", "search", "metric", "stat", "card", "text"],
      }),
    ]),
  ]),
};
