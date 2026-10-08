import type { ReactNode } from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
// This is a scenario-owned harness, not a published library asset. Keep the
// exact path so the contract exercises the harness implementation directly.
import { NavigationContext } from "../../harnesses/navigation-context/versions/1.0.0/NavigationContext";

vi.mock("@vrooli/react-component-library/useMediaQuery/1", async original => ({
  ...await original<object>(), useBreakpoint: () => true,
}));
afterEach(cleanup);

describe("navigation preview context", () => {
  it("preserves children supplied as subject arguments", () => {
    const Subject = ({ children }: Record<string, unknown>) => <button>{children as ReactNode}</button>;
    render(<NavigationContext subject={Subject} args={{ children: "Argument label" }} />);
    expect(screen.getByRole("button", { name: "Argument label" })).toBeInTheDocument();
  });
  it("renders the injected subject and preserves its callbacks inside library chrome", () => {
    const activate = vi.fn();
    const Subject = ({ onActivate }: Record<string, unknown>) => <button onClick={onActivate as () => void}>People</button>;
    const { container } = render(<NavigationContext subject={Subject} args={{ onActivate: activate }} config={{ title: "Directory" }} />);
    const subject = screen.getByRole("button", { name: "People" });
    expect(subject.closest("[data-rcl-app-shell-utility]")).not.toBeNull();
    expect(container.querySelectorAll("[data-rcl-app-shell]")).toHaveLength(1);
    expect(screen.getByRole("heading", { name: "Directory" })).toBeInTheDocument();
    fireEvent.click(subject);
    expect(activate).toHaveBeenCalledOnce();
  });

  it("can place a content specimen in the main region without duplicating it", () => {
    const Subject = () => <button>Inspect split</button>;
    render(<NavigationContext subject={Subject} config={{ region: "content" }} />);
    const subject = screen.getByRole("button", { name: "Inspect split" });
    expect(subject.closest("main")).not.toBeNull();
    expect(screen.getAllByRole("button", { name: "Inspect split" })).toHaveLength(1);
  });
});
