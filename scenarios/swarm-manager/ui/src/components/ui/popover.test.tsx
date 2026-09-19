import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useRef, useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { Popover } from "./popover";

function AnchoredPopover() {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setIsOpen(true)}
      >
        Open
      </button>
      <Popover
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        triggerRef={triggerRef}
        testId="anchored-popover"
      >
        Anchored content
      </Popover>
    </>
  );
}

describe("Popover", () => {
  it("anchors to a trigger and closes on Escape", async () => {
    const user = userEvent.setup();
    const rectSpy = vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function getRect(this: HTMLElement) {
        if (this.textContent === "Open") {
          return {
            x: 24,
            y: 32,
            top: 32,
            right: 104,
            bottom: 64,
            left: 24,
            width: 80,
            height: 32,
            toJSON: () => ({}),
          };
        }
        return {
          x: 0,
          y: 0,
          top: 0,
          right: 180,
          bottom: 80,
          left: 0,
          width: 180,
          height: 80,
          toJSON: () => ({}),
        };
      });

    render(<AnchoredPopover />);

    await user.click(screen.getByRole("button", { name: "Open" }));

    const popover = screen.getByTestId("anchored-popover");
    expect(popover).toHaveTextContent("Anchored content");
    // Measure-then-reveal: the menu is revealed only after its anchored
    // position is written, so it is never painted at the (0,0) top-left origin.
    expect(popover).toHaveStyle({ left: "24px", top: "72px" });
    expect(popover.style.left).not.toBe("0px");
    expect(popover.style.visibility).not.toBe("hidden");

    await user.keyboard("{Escape}");
    expect(screen.queryByTestId("anchored-popover")).not.toBeInTheDocument();

    rectSpy.mockRestore();
  });

  it("closes on outside click", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();

    render(
      <>
        <button type="button">Outside</button>
        <Popover isOpen onClose={onClose} x={20} y={30} testId="popover">
          Menu
        </Popover>
      </>,
    );

    await user.click(screen.getByRole("button", { name: "Outside" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("reveals a fixed-position (context-menu) popover at its exact coordinates", () => {
    render(
      <Popover isOpen onClose={() => {}} x={20} y={30} testId="fixed-popover">
        Menu
      </Popover>,
    );

    const popover = screen.getByTestId("fixed-popover");
    expect(popover).toHaveStyle({ left: "20px", top: "30px" });
    expect(popover.style.visibility).not.toBe("hidden");
  });
});
