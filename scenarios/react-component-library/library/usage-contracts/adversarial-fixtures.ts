import type { OverlayActionList } from "../components/OverlayActionBar/versions/1.0.0/OverlayActionBar";

const german = [
  "Änderungen dauerhaft speichern",
  "Arbeitsbereich sicher verlassen",
  "Einstellungen zurücksetzen",
  "Änderungen zur Überprüfung senden",
  "Diese Sitzung schließen und verwerfen",
];

/** Standard pseudo-localisation transform: accent letters and add expansion brackets. */
export function pseudoLocalise(text: string): string {
  const map: Record<string, string> = { a: "à", e: "ë", i: "ï", o: "ô", u: "ü", A: "À", E: "Ë", I: "Ï", O: "Ô", U: "Û" };
  const expanded = Array.from(text, (char) => map[char] ?? char).join("");
  return `⟦${expanded}${"~".repeat(Math.ceil(expanded.length * 0.25))}⟧`;
}

export const fiveGermanOverlayActions: OverlayActionList = german.map((label, index) => ({
  id: `de-action-${index + 1}`,
  label,
  onSelect: () => undefined,
  primary: index === 0 ? true : undefined,
})) as unknown as OverlayActionList;

export const pseudoLocalisedOverlayActions: OverlayActionList = fiveGermanOverlayActions.map((action) => ({ ...action, id: `pseudo-${action.id}`, label: pseudoLocalise(action.label) })) as unknown as OverlayActionList;

// This malformed input is intentionally outside the type system to exercise the development check.
export const missingIDOverlayActions = [{ label: "Continue", primary: true, onSelect: () => undefined }] as unknown as OverlayActionList;
