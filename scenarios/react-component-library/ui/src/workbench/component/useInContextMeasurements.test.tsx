import { useRef } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { useInContextMeasurements } from "./useInContextMeasurements";

class ResizeObserverStub {
  static instances: ResizeObserverStub[] = [];
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();

  constructor(_callback: ResizeObserverCallback) {
    ResizeObserverStub.instances.push(this);
  }
}

function MeasurementHarness() {
  const pageRef = useRef<HTMLDivElement>(null);
  const slotWrapRef = useRef<HTMLDivElement>(null);
  const probeRef = useRef<HTMLDivElement>(null);

  useInContextMeasurements({
    pageRef,
    slotWrapRef,
    probeRef,
    scale: 1,
    viewport: { id: "1024", width: 1024, height: 768, label: "1024" },
    safeTop: 0,
    safeBottom: 0,
    overflowEnabled: true,
    targetsEnabled: true,
    contrastEnabled: true,
    renderSpec: DEFAULT_TEMPLATE.spec,
    templateId: DEFAULT_TEMPLATE.id,
    themeSignature: "default",
    status: "ready",
    storyId: "default",
  });

  return (
    <div ref={pageRef} data-testid="measurement-page">
      <div className="tpl">
        <div className="tpl-page">
          <span ref={slotWrapRef} className="tpl-slot" data-node-id="title">
            Heading
          </span>
          <button type="button">Action</button>
        </div>
      </div>
      <div ref={probeRef}>
        <div data-ic-probe="320" />
      </div>
    </div>
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  ResizeObserverStub.instances = [];
});

describe("useInContextMeasurements lifecycle", () => {
  it("disconnects resize observation and removes the contrast pick listener on unmount", () => {
    vi.stubGlobal("ResizeObserver", ResizeObserverStub);
    const addListener = vi.spyOn(HTMLElement.prototype, "addEventListener");
    const removeListener = vi.spyOn(HTMLElement.prototype, "removeEventListener");

    const { getByTestId, unmount } = renderWithProviders(<MeasurementHarness />);
    const page = getByTestId("measurement-page");
    const observer = ResizeObserverStub.instances[0];

    expect(observer?.observe).toHaveBeenCalledWith(page);
    expect(observer?.observe).toHaveBeenCalledWith(page.querySelector(".tpl"));
    const clickRegistration = addListener.mock.calls.find(
      ([type], index) => type === "click" && addListener.mock.contexts[index] === page,
    );
    expect(clickRegistration).toBeDefined();

    unmount();

    expect(observer?.disconnect).toHaveBeenCalledOnce();
    const removedClickListener = removeListener.mock.calls.some(
      ([type, listener], index) =>
        type === "click" &&
        listener === clickRegistration?.[1] &&
        removeListener.mock.contexts[index] === page,
    );
    expect(removedClickListener).toBe(true);
  });
});
