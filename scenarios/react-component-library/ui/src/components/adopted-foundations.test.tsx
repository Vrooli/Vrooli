import { readFileSync, readdirSync } from "node:fs";
import { dirname, join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { useEffect, useRef, useState } from "react";
import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { useFocusTrap } from "@vrooli/react-component-library/useFocusTrap/1";
import { EmptyState } from "./EmptyState";
import { Icon } from "./Icon";
import { Pressable } from "./Pressable";
import { Text } from "./Text";
import { renderWithProviders } from "../test-utils";

function FoundationFixture() {
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  return (
    <>
      <Text data-testid="foundation-text" textStyle="heading">
        Foundation fixture
      </Text>
      <Icon name="check" label="Complete" />
      <Pressable pending={pending} onClick={() => setPending((value) => !value)}>
        Save
      </Pressable>
      <EmptyState title="Nothing here" description="Create the first item." />
      <button type="button" onClick={() => setOpen(true)}>
        Open fixture drawer
      </button>
      <FullPageDrawer
        open={open}
        onClose={() => setOpen(false)}
        title="Drawer fixture"
        closeLabel="Close drawer"
      >
        <Text as="h2" textStyle="title">
          Drawer fixture
        </Text>
      </FullPageDrawer>
    </>
  );
}

function DelayedFocusTrapFixture() {
  const panelRef = useRef<HTMLDivElement>(null);
  const [portalMounted, setPortalMounted] = useState(false);
  useFocusTrap(true, panelRef);
  useEffect(() => setPortalMounted(true), []);

  return portalMounted ? (
    <div ref={panelRef} role="dialog" aria-label="Delayed portal">
      <button type="button">First delayed action</button>
      <button type="button">Last delayed action</button>
    </div>
  ) : null;
}

describe("adopted foundation entry points", () => {
  it("contains focus when a portal assigns the surface ref after the trap effect starts", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DelayedFocusTrapFixture />);

    const first = await screen.findByRole("button", { name: "First delayed action" });
    const last = screen.getByRole("button", { name: "Last delayed action" });
    last.focus();
    await user.tab();

    expect(first).toHaveFocus();
  });

  it("expose the shared semantics and state transitions", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FoundationFixture />);

    expect(screen.getByTestId("foundation-text")).toHaveTextContent("Foundation fixture");
    expect(screen.getByRole("img", { name: "Complete" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toHaveAttribute(
      "data-rcl-pending",
      "false",
    );

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(screen.getByRole("button", { name: "Working…" })).toHaveAttribute(
      "data-rcl-pending",
      "true",
    );

    await user.click(screen.getByRole("button", { name: "Open fixture drawer" }));
    expect(screen.getByRole("dialog", { name: "Drawer fixture" })).toBeInTheDocument();
    await user.click(screen.getByTestId("overlays.full-page-drawer.close"));
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Drawer fixture" })).not.toBeInTheDocument(),
    );
  });

  it("keeps the drawer content available for minimal consumers", () => {
    renderWithProviders(
      <>
        <FullPageDrawer open title="Drawer" closeLabel="Close drawer">
          Drawer content
        </FullPageDrawer>
        <Text textStyle="caption" truncate balance numeric>
          Styled metadata
        </Text>
        <Text textStyle="body">String style</Text>
        <Icon name="close" size="lg" tone="danger" />
        <EmptyState
          title="Actionable empty"
          icon={<Icon name="plus" aria-hidden />}
          action={<button type="button">Create</button>}
        />
      </>,
    );

    expect(screen.getByRole("dialog", { name: "Drawer" })).toHaveTextContent("Drawer content");
    expect(screen.getByText("Styled metadata")).toHaveAttribute("data-text-truncate", "true");
    expect(screen.getByText("String style")).toHaveAttribute("data-text-style", "body");
    // The open modal correctly makes background siblings inert; this assertion
    // checks that the sibling composition remains mounted without pretending it
    // should be exposed in the accessibility tree while the drawer is open.
    expect(screen.getByText("Create")).toBeInTheDocument();
  });
});

/**
 * Adoption guard.
 *
 * The rule is deliberately not "never render a DOM element": a primitive has to
 * bottom out in a real element somewhere, or the library would be turtles all the
 * way down. What the guard enforces is that a *composition* — a component in this
 * folder that is not itself the versioned primitive layer — may not hand-roll an
 * interactive element when `Button` / `IconButton` / `Pressable` / `ControlBase` /
 * `Input` / `Select` / `Textarea` already own that control's tap target, hover,
 * :focus-visible, disabled treatment and token-backed motion.
 *
 * A new component that renders a raw interactive element fails this test until
 * someone either composes the primitive or writes down, here, why it cannot.
 * Prefer the former.
 */
const RAW_INTERACTIVE_ELEMENTS = ["button", "select", "textarea", "input"] as const;

/**
 * Files under `versions/` directories ARE the primitive layer: extracted, version-pinned
 * assets whose whole job is to turn a token contract into a DOM element. They are
 * exempt by category, not by oversight.
 */
const PRIMITIVE_LAYER = /[\\/]versions[\\/]/;

/**
 * Compositions that legitimately render a raw interactive element. Every entry
 * needs a reason a reviewer can disagree with. Adding an entry is the fallback;
 * composing the primitive is the fix.
 */
const JUSTIFIED_RAW_ELEMENTS: Record<string, string> = {
  "markdown-harvest/CodeBlock.tsx":
    "Self-contained harvest asset: its @deps header declares react + shiki only, so " +
    "importing a sibling library primitive would break standalone extraction. The " +
    "control treatment is expressed in shared tokens instead.",
  "markdown-harvest/InlineCode.tsx":
    "Same harvest boundary as CodeBlock, plus the affordances sit inside a line of " +
    "prose, where ControlBase's 44px tap-target box would break text flow.",
  "markdown-harvest/MermaidDiagram.tsx":
    "Same harvest boundary as CodeBlock: its @deps header declares react + mermaid " +
    "only, so the toolbar controls carry the token-backed treatment inline rather " +
    "than importing the library Button.",
  "color-picker-harvest/ColorPicker.tsx":
    "Self-contained harvest asset, and the swatch IS the colour: its background is a " +
    "runtime value and it must not inherit a control variant's own background.",
};

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) return sourceFiles(full);
    if (!entry.name.endsWith(".tsx")) return [];
    if (entry.name.endsWith(".test.tsx")) return [];
    return [full];
  });
}

/** Comments describe the rule; they must not trip it. */
function stripComments(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, "");
}

function rawElementsIn(source: string): string[] {
  const body = stripComments(source);
  return RAW_INTERACTIVE_ELEMENTS.filter((tag) => new RegExp(`<${tag}(\\s|>|$)`, "m").test(body));
}

describe("raw interactive element adoption", () => {
  const root = dirname(fileURLToPath(import.meta.url));
  const compositions = sourceFiles(root)
    .filter((file) => !PRIMITIVE_LAYER.test(file))
    .map((file) => ({ key: relative(root, file).split(sep).join("/"), file }));

  it("finds the composition layer it is supposed to police", () => {
    expect(compositions.length).toBeGreaterThan(20);
  });

  it("lets no composition hand-roll an interactive element without a written reason", () => {
    const unexplained = compositions
      .map(({ key, file }) => ({ key, tags: rawElementsIn(readFileSync(file, "utf8")) }))
      .filter(({ key, tags }) => tags.length > 0 && !JUSTIFIED_RAW_ELEMENTS[key])
      .map(({ key, tags }) => `${key} renders <${tags.join(">, <")}>`);

    expect(
      unexplained,
      "Compose Button/IconButton/Pressable/ControlBase/Input/Select/Textarea instead, " +
        "or add a justification to JUSTIFIED_RAW_ELEMENTS in this file.",
    ).toEqual([]);
  });

  it("keeps every justification live, so the exemption list cannot rot", () => {
    const stale = Object.keys(JUSTIFIED_RAW_ELEMENTS).filter((key) => {
      const match = compositions.find((entry) => entry.key === key);
      return !match || rawElementsIn(readFileSync(match.file, "utf8")).length === 0;
    });

    expect(
      stale,
      "These files no longer render a raw interactive element — drop the entry.",
    ).toEqual([]);
  });

  it("holds every justification to a real explanation", () => {
    for (const [key, reason] of Object.entries(JUSTIFIED_RAW_ELEMENTS)) {
      expect(reason.length, `${key} needs a reason a reviewer can disagree with`).toBeGreaterThan(
        60,
      );
    }
  });
});
