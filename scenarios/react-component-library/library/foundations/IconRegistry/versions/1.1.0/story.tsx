// Preview contract exports for IconRegistry.
// The registry is a data module, so its specimen renders every entry as a
// bare glyph. That makes the contract assert what the module actually
// promises: that each declared name resolves to a drawable path.
import { ICON_REGISTRY, iconSize, type IconName } from "./IconRegistry";

export function Default() {
  const names = Object.keys(ICON_REGISTRY) as IconName[];
  return (
    <ul
      data-testid="story-registry"
      style={{
        display: "flex",
        flexWrap: "wrap",
        gap: "var(--space-sm, 16px)",
        listStyle: "none",
        margin: 0,
        padding: 0,
      }}
    >
      {names.map((name) => {
        const definition = ICON_REGISTRY[name];
        return (
          <li key={name} data-testid={`story-icon-${name}`}>
            <svg
              viewBox={definition.viewBox}
              fill="none"
              stroke="currentColor"
              strokeWidth="1.75"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
              style={{ inlineSize: iconSize("md"), blockSize: iconSize("md") }}
            >
              <path d={definition.path} />
            </svg>
          </li>
        );
      })}
    </ul>
  );
}
