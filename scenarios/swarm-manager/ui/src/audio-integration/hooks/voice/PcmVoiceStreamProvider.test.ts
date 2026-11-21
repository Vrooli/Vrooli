import { describe, expect, it } from "vitest";
import { openVoiceStream } from "../../api/voice";

describe("scenario voice stream adapter", () => {
  it("exposes a Connect stream opener to the shared provider", () => {
    expect(openVoiceStream).toBeTypeOf("function");
  });
});
