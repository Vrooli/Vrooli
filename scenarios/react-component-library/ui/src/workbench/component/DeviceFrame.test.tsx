import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { DeviceFrame } from "./DeviceFrame";

describe("DeviceFrame", () => {
  afterEach(() => cleanup());

  it("draws browser chrome and no status bar on desktop", () => {
    const { container } = renderWithProviders(
      <DeviceFrame device="desktop" width={1180} height={760} label="Desktop preview">
        <div>page</div>
      </DeviceFrame>,
    );
    expect(container.querySelector(".df-browser")).not.toBeNull();
    expect(container.querySelector(".df-status")).toBeNull();
    expect(screen.getByRole("group", { name: "Desktop preview" })).toBeTruthy();
  });

  it("draws status bar, island and home indicator on phone", () => {
    const { container } = renderWithProviders(
      <DeviceFrame device="phone" width={390} height={844} label="Phone preview">
        <div>page</div>
      </DeviceFrame>,
    );
    expect(container.querySelector(".df-status")).not.toBeNull();
    expect(container.querySelector(".df-island")).not.toBeNull();
    expect(container.querySelector(".df-home")).not.toBeNull();
  });

  it("scales the logical viewport once", () => {
    const { container } = renderWithProviders(
      <DeviceFrame device="phone" width={390} height={844} scale={0.5} label="Phone preview">
        <div>page</div>
      </DeviceFrame>,
    );
    const viewport = container.querySelector<HTMLElement>(".df-viewport");
    const content = container.querySelector<HTMLElement>(".df-content");
    expect(viewport?.style.width).toBe("195px");
    expect(viewport?.style.height).toBe("422px");
    expect(content?.style.width).toBe("390px");
    expect(content?.style.transform).toBe("scale(0.5)");
  });

  it("exposes selection as a real button, not a clickable div", () => {
    const { container } = renderWithProviders(
      <DeviceFrame device="phone" width={390} height={844} label="Phone preview" onSelect={() => {}}>
        <div>page</div>
      </DeviceFrame>,
    );
    const select = container.querySelector(".df-select");
    expect(select?.tagName).toBe("BUTTON");
    expect(select?.getAttribute("aria-label")).toBe("workbench.studio.selectFrame");
  });

  it("shows instruments only when guides are enabled", () => {
    const { container, rerender } = renderWithProviders(
      <DeviceFrame device="tablet" width={834} height={1112} label="Tablet preview">
        <div>page</div>
      </DeviceFrame>,
    );
    expect(container.querySelector(".df-guides")).toBeNull();

    rerender(
      <DeviceFrame device="tablet" width={834} height={1112} label="Tablet preview" guides>
        <div>page</div>
      </DeviceFrame>,
    );
    expect(container.querySelector(".df-safe")).not.toBeNull();
    expect(container.querySelector(".df-thumb")).not.toBeNull();
  });
});
