import { describe, expect, it } from "vitest";
import {
  countableTypesForKind,
  starterCardBadgeSpec,
  starterSuggestionsForKind,
} from "./session-starter-suggestions";

describe("operating-mode-authoring starter suggestions", () => {
  const suggestions = starterSuggestionsForKind("operating_mode_authoring");
  const byId = new Map(suggestions.map((s) => [s.id, s]));

  it("leads with a warm describe-first card that needs no attached context", () => {
    const lead = suggestions[0];
    expect(lead?.id).toBe("mode-describe");
    expect(lead?.requirements ?? []).toHaveLength(0);
    expect(lead?.text.toLowerCase()).toContain("phase graph");
  });

  it("offers a reuse-first start-from card gated on an operating_mode", () => {
    const startFrom = byId.get("mode-start-from");
    expect(startFrom).toBeDefined();
    const spec = starterCardBadgeSpec(startFrom!);
    expect(spec).toEqual({ type: "operating_mode", filterKey: undefined, gating: true });
  });

  it("counts every gating context type its cards require", () => {
    const countable = countableTypesForKind("operating_mode_authoring");
    expect(countable).toContain("operating_mode");
    expect(countable).toContain("initiative");
  });
});
