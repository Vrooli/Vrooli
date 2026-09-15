import { describe, expect, it } from "vitest";

import { themesClient } from "./themes";

describe("themes API client", () => {
  it("exports the generated client used by theme controls", () => {
    expect(themesClient).toBeDefined();
    expect(Object.keys(themesClient)).not.toHaveLength(0);
  });
});
