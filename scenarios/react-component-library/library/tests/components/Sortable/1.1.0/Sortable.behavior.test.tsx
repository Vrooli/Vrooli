import { describe, expect, it } from "vitest";
import { Sortable } from "../../../../components/Sortable/versions/1.1.0/Sortable";

describe("Sortable drag integration", () => {
  it("publishes the shared-drag sortable", () => expect(Sortable).toBeDefined());
});
