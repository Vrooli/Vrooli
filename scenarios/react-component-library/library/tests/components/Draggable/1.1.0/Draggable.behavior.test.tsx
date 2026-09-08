import { describe, expect, it } from "vitest";
import { Draggable } from "@vrooli/react-component-library/Draggable/1.1.0";

describe("Draggable gesture integration", () => {
  it("publishes the shared-drag draggable", () => expect(Draggable).toBeDefined());
});
