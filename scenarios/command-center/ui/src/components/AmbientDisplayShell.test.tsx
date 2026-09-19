import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { AmbientDisplayShell } from "./AmbientDisplayShell";

const rail = () => screen.getByTestId("cycle-rail");

describe("cycle rail", () => { // [REQ:CC-P1-008] [REQ:CC-P1-017]
  it("draws each beat's segment and fills the active one to its beat progress", () => {
    renderWithProviders(
      <AmbientDisplayShell theme="ground-control" title="Mission Control" position="ROOM 1 OF 6" beatCount={3} beatIndex={1} beatProgress={0.5} beatDurations={[10, 20, 30]}>
        <div />
      </AmbientDisplayShell>,
    );
    const segments = [0, 1, 2].map((index) => screen.getByTestId(`beat-rail-${index}`));
    expect(segments[0]?.querySelector("span")?.style.transform).toBe("scaleX(1.000)");
    expect(segments[1]?.querySelector("span")?.style.transform).toBe("scaleX(0.500)");
    expect(segments[2]?.querySelector("span")?.style.transform).toBe("scaleX(0.000)");
    expect(segments[1]).toHaveAttribute("data-active");
  });

  it("names the held state so a waiting beat never reads as a frozen one", () => {
    renderWithProviders(
      <AmbientDisplayShell theme="ground-control" title="Mission Control" position="ROOM 1 OF 6" beatCount={2} beatIndex={0} beatProgress={0.999} held>
        <div />
      </AmbientDisplayShell>,
    );
    expect(rail()).toHaveAttribute("data-held");
    expect(rail()).toHaveAttribute("aria-label", "Reading the held section");
  });

  it("names the paused state ahead of the held state", () => {
    renderWithProviders(
      <AmbientDisplayShell theme="ground-control" title="Mission Control" position="ROOM 1 OF 6" beatCount={2} beatIndex={0} beatProgress={0.4} held paused>
        <div />
      </AmbientDisplayShell>,
    );
    expect(rail()).toHaveAttribute("aria-label", "Cycle paused");
    expect(rail()).toHaveAttribute("data-paused");
  });
});
