import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";

import { TabList, TabPanel } from "./Tabs";

type Key = "a" | "b" | "c";
const items: { value: Key; label: string }[] = [
  { value: "a", label: "A" },
  { value: "b", label: "B" },
  { value: "c", label: "C" },
];

function Harness({ value }: { value: Key }) {
  const [v, setV] = useState<Key>(value);
  return (
    <>
      <TabList
        idPrefix="x"
        value={v}
        onChange={setV}
        items={items}
        aria-label="demo"
      />
      <TabPanel idPrefix="x" value="a" active={v}>
        <span data-testid="x-content-a">a-content</span>
      </TabPanel>
      <TabPanel idPrefix="x" value="b" active={v}>
        <span data-testid="x-content-b">b-content</span>
      </TabPanel>
      <TabPanel idPrefix="x" value="c" active={v}>
        <span data-testid="x-content-c">c-content</span>
      </TabPanel>
    </>
  );
}

describe("TabList / TabPanel", () => {
  afterEach(() => cleanup());

  it("renders each tab with aria-selected reflecting the active value", () => {
    render(<Harness value="b" />);
    expect(screen.getByTestId("x-tab-a")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("x-tab-b")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("x-tab-c")).toHaveAttribute("aria-selected", "false");
  });

  it("renders only the active panel", () => {
    render(<Harness value="a" />);
    expect(screen.getByTestId("x-content-a")).toBeInTheDocument();
    expect(screen.queryByTestId("x-content-b")).not.toBeInTheDocument();
  });

  it("calls onChange when a different tab is clicked", async () => {
    const onChange = vi.fn();
    render(
      <TabList
        idPrefix="y"
        value="a"
        onChange={onChange}
        items={items}
        aria-label="demo"
      />,
    );
    const user = userEvent.setup();
    await user.click(screen.getByTestId("y-tab-c"));
    expect(onChange).toHaveBeenLastCalledWith("c");
  });

  it("ArrowRight cycles to the next tab", () => {
    const onChange = vi.fn();
    render(
      <TabList
        idPrefix="z"
        value="a"
        onChange={onChange}
        items={items}
        aria-label="demo"
      />,
    );
    fireEvent.keyDown(screen.getByTestId("z-tab-a"), { key: "ArrowRight" });
    expect(onChange).toHaveBeenLastCalledWith("b");
  });

  it("ArrowLeft wraps to the last tab", () => {
    const onChange = vi.fn();
    render(
      <TabList
        idPrefix="z2"
        value="a"
        onChange={onChange}
        items={items}
        aria-label="demo"
      />,
    );
    fireEvent.keyDown(screen.getByTestId("z2-tab-a"), { key: "ArrowLeft" });
    expect(onChange).toHaveBeenLastCalledWith("c");
  });

  it("Home and End jump to the boundary tabs", () => {
    const onChange = vi.fn();
    render(
      <TabList
        idPrefix="z3"
        value="b"
        onChange={onChange}
        items={items}
        aria-label="demo"
      />,
    );
    fireEvent.keyDown(screen.getByTestId("z3-tab-b"), { key: "Home" });
    expect(onChange).toHaveBeenLastCalledWith("a");
    fireEvent.keyDown(screen.getByTestId("z3-tab-b"), { key: "End" });
    expect(onChange).toHaveBeenLastCalledWith("c");
  });

  it("roving tabindex: only the active tab is in the tab order", () => {
    render(
      <TabList
        idPrefix="z4"
        value="b"
        onChange={() => {}}
        items={items}
        aria-label="demo"
      />,
    );
    expect(screen.getByTestId("z4-tab-a")).toHaveAttribute("tabindex", "-1");
    expect(screen.getByTestId("z4-tab-b")).toHaveAttribute("tabindex", "0");
    expect(screen.getByTestId("z4-tab-c")).toHaveAttribute("tabindex", "-1");
  });
});
