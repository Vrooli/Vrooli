const labels: Record<string, string> = {
  scenario: "Scenarios",
  resource: "Resources",
  package: "Packages",
  "control-plane": "Control plane",
  tool: "Tools",
  safeguard: "Safeguards",
  team: "Teams",
  docs: "Docs",
  project: "Project",
};

export function groupKindLabel(kind?: string, source?: string): string {
  if (source === "manual") return "Manual rules";
  if (source === "builtin") return "Other";
  if (!kind) return "Other";
  return labels[kind] ?? kind.replace(/\b\w/g, (letter) => letter.toUpperCase());
}
