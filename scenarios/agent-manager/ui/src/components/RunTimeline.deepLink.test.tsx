import { expect, test, vi } from "vitest";
import { focusLinkedEvent } from "./RunTimeline";

test("deep-link focus resolves exact event identity and falls back to sequence", () => {
  const container = document.createElement("div");
  const target = document.createElement("div");
  target.tabIndex = -1;
  target.dataset.eventIds = "event-6 event-7";
  target.dataset.eventSequences = "6 7";
  target.scrollIntoView = vi.fn();
  container.append(target);
  document.body.append(container);

  expect(focusLinkedEvent(container, "event-7", "7")).toBe(true);
  expect(document.activeElement).toBe(target);
  expect(target.scrollIntoView).toHaveBeenCalledWith({ block: "center", behavior: "smooth" });
  expect(focusLinkedEvent(container, "deleted-event", "7")).toBe(true);
  expect(focusLinkedEvent(container, "deleted-event", "999")).toBe(false);
});
