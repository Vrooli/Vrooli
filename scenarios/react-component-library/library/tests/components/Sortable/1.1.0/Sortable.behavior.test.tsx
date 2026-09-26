import { describe, expect, it } from "vitest";
import { Sortable } from "@vrooli/react-component-library/Sortable/1.1.0";

describe("Sortable drag integration", () => {
  it("publishes the shared-drag sortable", () => expect(Sortable).toBeDefined());
});
