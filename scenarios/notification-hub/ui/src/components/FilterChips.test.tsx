import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { FilterChips } from "./FilterChips";

const CHIPS = ["all", "awaiting", "answered", "expired"].map((id, index) => ({ id, to: `/history?state=${id}`, label: id, current: index === 0, testId: `chip-${id}` }));
const STRIP = "filter-strip";

function renderChips() {
  return renderWithProviders(
    <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <FilterChips label="States" chips={CHIPS} testId={STRIP} />
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

/** jsdom does no layout: give the strip a width and a longer scroll width. */
function lay(strip: HTMLElement, { client, scroll }: { client: number; scroll: number }) {
  Object.defineProperty(strip, "clientWidth", { configurable: true, value: client });
  Object.defineProperty(strip, "scrollWidth", { configurable: true, value: scroll });
}

afterEach(() => vi.restoreAllMocks());

describe("FilterChips", () => {
  it("is a strip a capture can recognize, with every chip reachable and the current one marked", () => {
    renderChips();
    const strip = screen.getByTestId(STRIP);
    expect(strip).toHaveClass("overflow-x-auto");
    expect(screen.getByTestId("chip-all")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("chip-expired")).toHaveAttribute("href", "/history?state=expired");
  });

  it("shows a fade where more chips wait, and only there", () => {
    const { container } = renderChips();
    const strip = screen.getByTestId(STRIP);
    expect(container.querySelector("[data-edge]")).toBeNull();

    lay(strip, { client: 390, scroll: 600 });
    fireEvent.scroll(strip);
    expect(container.querySelector('[data-edge="after"]')).not.toBeNull();
    expect(container.querySelector('[data-edge="before"]')).toBeNull();

    strip.scrollLeft = 105;
    fireEvent.scroll(strip);
    expect(container.querySelector('[data-edge="after"]')).not.toBeNull();
    expect(container.querySelector('[data-edge="before"]')).not.toBeNull();

    strip.scrollLeft = 210;
    fireEvent.scroll(strip);
    expect(container.querySelector('[data-edge="after"]')).toBeNull();
    expect(container.querySelector('[data-edge="before"]')).not.toBeNull();
  });

  it("reads the distance travelled in a right-to-left page, where scrollLeft is negative", () => {
    const { container } = renderChips();
    const strip = screen.getByTestId(STRIP);
    lay(strip, { client: 390, scroll: 600 });
    strip.scrollLeft = -210;
    fireEvent.scroll(strip);
    expect(container.querySelector('[data-edge="after"]')).toBeNull();
    expect(container.querySelector('[data-edge="before"]')).not.toBeNull();
  });
});
