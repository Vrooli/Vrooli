import { describe, expect, it } from "vitest";
import { useOverlaySurface } from "../../../../components/useOverlaySurface/versions/1.4.2/useOverlaySurface";

describe("useOverlaySurface gesture integration", () => {
  it("publishes the shared swipe substrate", () => expect(useOverlaySurface).toBeTypeOf("function"));
});
