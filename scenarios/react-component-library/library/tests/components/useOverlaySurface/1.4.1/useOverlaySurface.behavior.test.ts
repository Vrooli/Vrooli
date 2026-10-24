import { describe, expect, it } from "vitest";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.4.2";

describe("useOverlaySurface gesture integration", () => {
  it("publishes the shared swipe substrate", () => expect(useOverlaySurface).toBeTypeOf("function"));
});
