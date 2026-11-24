import { CharacterCount } from "./CharacterCount";

export function Default() {
  return (
    <div style={{ display: "grid", gap: "var(--space-xs)", maxInlineSize: "28rem" }}>
      <CharacterCount count="214 / 500 characters" />
      <CharacterCount
        count="642 / 500 characters"
        overLimit
        message="Shorten this entry or move details into a file."
      />
    </div>
  );
}
