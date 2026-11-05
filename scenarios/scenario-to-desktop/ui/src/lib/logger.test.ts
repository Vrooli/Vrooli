import { afterEach, describe, expect, it, vi } from "vitest";
import { logger } from "./logger";

describe("logger", () => {
  afterEach(() => vi.restoreAllMocks());

  it("delegates each severity to the matching console method", () => {
    const info = vi.spyOn(console, "info").mockImplementation(() => undefined);
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    const error = vi
      .spyOn(console, "error")
      .mockImplementation(() => undefined);
    const debug = vi
      .spyOn(console, "debug")
      .mockImplementation(() => undefined);

    logger.info("generated", { platform: "linux" });
    logger.warn("preflight warning", "runtime missing");
    logger.error("pipeline failed", new Error("build failed"));
    logger.debug("desktop session", "session-1");

    expect(info).toHaveBeenCalledWith("generated", { platform: "linux" });
    expect(warn).toHaveBeenCalledWith("preflight warning", "runtime missing");
    expect(error).toHaveBeenCalledWith("pipeline failed", expect.any(Error));
    expect(debug).toHaveBeenCalledWith("desktop session", "session-1");
  });
});
