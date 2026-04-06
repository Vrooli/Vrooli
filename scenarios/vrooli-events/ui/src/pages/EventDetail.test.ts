// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectorsManifest } from "../consts/selectors";

// [REQ:REQ-UI-004] Event detail view — full detail panel selector coverage
describe("event detail view selectors", () => {
  it("includes all event detail fields", () => {
    expect(selectorsManifest.selectors["eventDetail.panel"]).toBeDefined();
    expect(selectorsManifest.selectors["eventDetail.closeButton"]).toBeDefined();
    expect(selectorsManifest.selectors["eventDetail.eventId"]).toBeDefined();
    expect(selectorsManifest.selectors["eventDetail.eventType"]).toBeDefined();
    expect(selectorsManifest.selectors["eventDetail.payload"]).toBeDefined();
    expect(selectorsManifest.selectors["eventDetail.metadata"]).toBeDefined();
  });

  it("event detail selectors produce correct data-testid format", () => {
    const panel = selectorsManifest.selectors["eventDetail.panel"];
    expect(panel?.selector).toBe('[data-testid="event-detail-panel"]');
    const close = selectorsManifest.selectors["eventDetail.closeButton"];
    expect(close?.selector).toBe('[data-testid="event-detail-close"]');
  });

  it("event detail panel has id and type fields for inspection", () => {
    const eventId = selectorsManifest.selectors["eventDetail.eventId"];
    expect(eventId?.testId).toBe("event-detail-id");
    const eventType = selectorsManifest.selectors["eventDetail.eventType"];
    expect(eventType?.testId).toBe("event-detail-type");
  });
});
