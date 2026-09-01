import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render } from "@testing-library/react";
import { DashboardLayout } from "./DashboardLayout";
import { expectNoA11yViolations } from "../test-utils/a11y";

afterEach(() => cleanup());

describe("DashboardLayout accessibility", () => {
  it("keeps the room landmarks free of axe violations", async () => {
    const { container } = render(
      <DashboardLayout themeKey="ground-control" title="Mission Control">
        <section aria-label="Room scene"><p>Instrument surface</p></section>
      </DashboardLayout>,
    );
    await expectNoA11yViolations(container);
    expect(container.querySelector("main")).toBeTruthy();
  });
});
