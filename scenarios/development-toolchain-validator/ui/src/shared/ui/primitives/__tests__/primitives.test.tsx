/**
 * Primitive smoke tests.
 *
 * Per the testing plan: one assertion per primitive that the declared
 * variant renders the expected `data-variant` attribute and key class
 * tokens. Not snapshot tests — we don't want every Tailwind class change
 * to churn the file.
 */
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { Badge } from "../Badge";
import { Button } from "../Button";
import { Card } from "../Card";
import { Dialog, DialogTitle } from "../Dialog";
import { Input } from "../Input";
import { Popover } from "../Popover";
import { ScrollArea } from "../ScrollArea";
import { Select } from "../Select";
import { Sheet, SheetTitle } from "../Sheet";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../Tabs";
import { Textarea } from "../Textarea";
import { Tooltip } from "../Tooltip";

const wrap = (ui: React.ReactElement) =>
  render(<MemoryRouter>{ui}</MemoryRouter>);

describe("Button primitive", () => {
  it("emits the declared variant on data-variant", () => {
    wrap(<Button variant="danger" data-testid="b">x</Button>);
    expect(screen.getByTestId("b")).toHaveAttribute("data-variant", "danger");
  });

  it("defaults to the primary variant", () => {
    wrap(<Button data-testid="b">x</Button>);
    expect(screen.getByTestId("b")).toHaveAttribute("data-variant", "default");
  });
});

describe("Badge primitive", () => {
  it("emits the verdict variant on data-variant", () => {
    wrap(<Badge variant="verdict-pass" data-testid="bg">pass</Badge>);
    expect(screen.getByTestId("bg")).toHaveAttribute("data-variant", "verdict-pass");
  });
});

describe("Card primitive", () => {
  it("emits the surface variant on data-variant", () => {
    wrap(<Card surface="raised" data-testid="c">x</Card>);
    expect(screen.getByTestId("c")).toHaveAttribute("data-variant", "raised");
  });
});

describe("Input primitive", () => {
  it("renders an input with the focus-ring class token", () => {
    wrap(<Input data-testid="i" />);
    expect(screen.getByTestId("i").className).toMatch(/focus-visible:ring-app-accent/);
  });
});

describe("Textarea primitive", () => {
  it("renders a textarea with the input surface token", () => {
    wrap(<Textarea data-testid="t" />);
    expect(screen.getByTestId("t").className).toMatch(/bg-app-surface-input/);
  });
});

describe("Select primitive", () => {
  it("renders a native <select> with token classes", () => {
    wrap(
      <Select data-testid="s">
        <option value="a">a</option>
      </Select>,
    );
    const el = screen.getByTestId("s");
    expect(el.tagName.toLowerCase()).toBe("select");
    expect(el.className).toMatch(/bg-app-surface-input/);
  });
});

describe("ScrollArea primitive", () => {
  it("applies overflow-auto and the optional max-height inline style", () => {
    wrap(<ScrollArea data-testid="sc" maxHeight="200px">x</ScrollArea>);
    const el = screen.getByTestId("sc");
    expect(el.className).toMatch(/overflow-auto/);
    expect(el.style.maxHeight).toBe("200px");
  });
});

describe("Tabs primitive", () => {
  it("renders the active panel only", () => {
    wrap(
      <Tabs value="a" onValueChange={() => undefined}>
        <TabsList>
          <TabsTrigger value="a">A</TabsTrigger>
          <TabsTrigger value="b">B</TabsTrigger>
        </TabsList>
        <TabsContent value="a"><span data-testid="panel-a">A body</span></TabsContent>
        <TabsContent value="b"><span data-testid="panel-b">B body</span></TabsContent>
      </Tabs>,
    );
    expect(screen.getByTestId("panel-a")).toBeInTheDocument();
    expect(screen.queryByTestId("panel-b")).not.toBeInTheDocument();
  });
});

describe("Dialog primitive", () => {
  it("does not render when closed", () => {
    wrap(<Dialog open={false} onOpenChange={() => undefined}><DialogTitle>hi</DialogTitle></Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders when open", () => {
    wrap(<Dialog open onOpenChange={() => undefined} ariaLabel="x"><DialogTitle>hi</DialogTitle></Dialog>);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});

describe("Sheet primitive", () => {
  it("renders with the side attribute when open", () => {
    wrap(
      <Sheet open onOpenChange={() => undefined} side="bottom" ariaLabel="x">
        <SheetTitle>x</SheetTitle>
      </Sheet>,
    );
    expect(screen.getByRole("dialog")).toHaveAttribute("data-side", "bottom");
  });
});

describe("Popover primitive", () => {
  it("renders content when open", () => {
    wrap(
      <Popover
        open
        onOpenChange={() => undefined}
        trigger={<button type="button">open</button>}
      >
        <span data-testid="popover-content">x</span>
      </Popover>,
    );
    expect(screen.getByTestId("popover-content")).toBeInTheDocument();
  });
});

describe("Tooltip primitive", () => {
  it("renders the tooltip content when defaultOpen is true", () => {
    wrap(
      <Tooltip content="explain" defaultOpen>
        <button type="button">trigger</button>
      </Tooltip>,
    );
    expect(screen.getByRole("tooltip")).toHaveTextContent("explain");
  });
});
