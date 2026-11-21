import type { JsonObject } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";

import daily from "../../../../../skills/notification-hub/kit/samples/daily-briefing.json?raw";
import roadmap from "../../../../../skills/notification-hub/kit/samples/opportunity-roadmap.json?raw";
import structural from "../../../../../skills/notification-hub/kit/samples/structural-change.json?raw";
import usage from "../../../../../skills/notification-hub/kit/samples/usage-mode.json?raw";
import { renderKitTemplate } from "./kitTemplate";

describe("Hub kit template rendering", () => {
  it.each([
    ["daily-briefing", daily],
    ["opportunity-roadmap", roadmap],
    ["structural-change", structural],
    ["usage-mode", usage],
  ])("embeds validated %s data in its inert JSON slot", (name, source) => {
    const data = JSON.parse(source) as JsonObject;
    const result = renderKitTemplate(name, data);
    expect(result.error).toBe("");
    expect(result.html).toContain('id="hub-data"');
    const slots = [...result.html.matchAll(/<script type="application\/json" id="hub-data">([\s\S]*?)<\/script>/g)];
    expect(slots).toHaveLength(1);
    expect(JSON.parse(slots[0]?.[1] ?? "null")).toEqual(data);
  });

  it("rejects data that does not match the published template schema", () => {
    const result = renderKitTemplate("usage-mode", {});
    expect(result.html).toBe("");
    expect(result.error).toBe("dataInvalid");
  });

  it("reports a template that is absent from this Hub build", () => {
    const result = renderKitTemplate("not-published", {});
    expect(result.html).toBe("");
    expect(result.error).toBe("unavailable");
  });
});
