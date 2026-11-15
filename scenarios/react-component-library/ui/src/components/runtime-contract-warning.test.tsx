import { render } from "@testing-library/react";
import { BottomNav } from "@vrooli/react-component-library/BottomNav/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1";
import { Card } from "@vrooli/react-component-library/Card/1";
import { afterEach, describe, expect, it, vi } from "vitest";

describe("versioned RCL development usage warnings", () => {
  afterEach(() => vi.restoreAllMocks());

  it("reports a real component's excess children/items with its version, violated limit, and JSX call site", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    const items = Array.from({ length: 6 }, (_, index) => ({
      id: `route-${index}`,
      label: `Route ${index}`,
      icon: <span aria-hidden="true" />,
    }));
    render(<BottomNav items={items} label="Primary navigation" />);

    const message = warning.mock.calls
      .map(([value]) => String(value))
      .find((value) => value.includes("react-component-library:BottomNav@1.5.7"));
    expect(message).toContain("items violates maximumCount <= 5");
    expect(message).toContain("received 6");
    expect(message).toMatch(/runtime-contract-warning\.test\.tsx:\d+:\d+/);
  });

  it("reports excess RCL child components from a real RCL component contract", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(
      <AsyncBoundary status="success" detectOffline={false}>
        <>
          <Card />
          <Card />
        </>
      </AsyncBoundary>,
    );

    const message = warning.mock.calls
      .map(([value]) => String(value))
      .find((value) => value.includes("react-component-library:AsyncBoundary@1.0.11"));
    expect(message).toContain("children violates maximumCount <= 1");
    expect(message).toContain("received 2");
    expect(message).toMatch(/runtime-contract-warning\.test\.tsx:\d+:\d+/);
  });

  it("does not count text and native elements as versioned RCL child components", () => {
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    render(
      <Button>
        <span>Save</span>
      </Button>,
    );
    expect(warning).not.toHaveBeenCalled();
  });
});
