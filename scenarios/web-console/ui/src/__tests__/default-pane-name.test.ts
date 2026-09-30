import { describe, expect, it } from "vitest";
import { defaultPaneName } from "../hooks/useSessionManager";

describe("defaultPaneName", () => {
  // A session created through the API or CLI carries display_label but no
  // saved pane. Falling straight through to the shell basename rendered every
  // such session as "bash", so a fully labelled workspace looked like a list
  // of identical shells.
  it("prefers the session's display label", () => {
    expect(defaultPaneName({ shell: "/bin/bash", display_label: "Route solver — time windows" }))
      .toBe("Route solver — time windows");
  });

  it("falls back to the shell basename when there is no label", () => {
    expect(defaultPaneName({ shell: "/bin/bash", display_label: "" })).toBe("bash");
  });

  it("treats a whitespace-only label as absent", () => {
    expect(defaultPaneName({ shell: "/usr/bin/zsh", display_label: "   " })).toBe("zsh");
  });

  it("falls back to terminal when the shell is unusable", () => {
    expect(defaultPaneName({ shell: "", display_label: "" })).toBe("terminal");
  });
});
