import { describe, expect, it } from "vitest";
import { Draggable } from "../../../../components/Draggable/versions/1.1.0/Draggable";

describe("Draggable gesture integration", () => {
  it("publishes the shared-drag draggable", () => expect(Draggable).toBeDefined());
});
