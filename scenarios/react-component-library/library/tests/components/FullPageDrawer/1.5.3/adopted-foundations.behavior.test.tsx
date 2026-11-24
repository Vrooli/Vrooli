import { useEffect, useRef, useState } from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { useFocusTrap } from "@vrooli/react-component-library/useFocusTrap/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { Pressable } from "@vrooli/react-component-library/Pressable/1";
import { Text } from "@vrooli/react-component-library/Text/1";

function FoundationFixture() {
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  return (
    <>
      <Text data-testid="foundation-text" textStyle="heading">
        Foundation fixture
      </Text>
      <Icon name="check" label="Complete" />
      <Pressable
        pending={pending}
        onClick={() => setPending((value) => !value)}
      >
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
        dismissAffordance="close"
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

function ExpandableDrawerFixture() {
  const [open, setOpen] = useState(true);
  return (
    <FullPageDrawer
      open={open}
      onOpenChange={setOpen}
      title="Expandable drawer"
      closeLabel="Close drawer"
      dismissAffordance="grabber"
      expandable
    >
      Drawer content
    </FullPageDrawer>
  );
}

function swipeGrabber(
  grabber: HTMLElement,
  startY: number,
  endY: number,
  pointerId: number,
) {
  fireEvent.pointerDown(grabber, {
    pointerId,
    pointerType: "touch",
    button: 0,
    clientX: 16,
    clientY: startY,
    timeStamp: 0,
  });
  fireEvent.pointerMove(window, {
    pointerId,
    pointerType: "touch",
    clientX: 16,
    clientY: endY,
    timeStamp: 400,
  });
  fireEvent.pointerUp(window, {
    pointerId,
    pointerType: "touch",
    clientX: 16,
    clientY: endY,
    timeStamp: 400,
  });
}

describe("adopted foundation entry points", () => {
  it("contains focus when a portal assigns the surface ref after the trap effect starts", async () => {
    const user = userEvent.setup();
    render(<DelayedFocusTrapFixture />);

    const first = await screen.findByRole("button", {
      name: "First delayed action",
    });
    const last = screen.getByRole("button", { name: "Last delayed action" });
    last.focus();
    await user.tab();

    expect(first).toHaveFocus();
  });

  it("expose the shared semantics and state transitions", async () => {
    const user = userEvent.setup();
    render(<FoundationFixture />);

    expect(screen.getByTestId("foundation-text")).toHaveTextContent(
      "Foundation fixture",
    );
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

    await user.click(
      screen.getByRole("button", { name: "Open fixture drawer" }),
    );
    expect(
      screen.getByRole("dialog", { name: "Drawer fixture" }),
    ).toBeInTheDocument();
    await user.click(screen.getByTestId("overlays.full-page-drawer.close"));
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Drawer fixture" }),
      ).not.toBeInTheDocument(),
    );
  });

  it("keeps the drawer content available for minimal consumers", () => {
    const metadataCopy = "Styled metadata";
    const stringStyleCopy = "String style";
    const actionLabel = "Create";
    render(
      <>
        <FullPageDrawer open title="Drawer" closeLabel="Close drawer">
          Drawer content
        </FullPageDrawer>
        <Text textStyle="caption" truncate balance numeric>
          Styled metadata
        </Text>
        <Text textStyle="body">{stringStyleCopy}</Text>
        <Icon name="close" size="lg" tone="danger" />
        <EmptyState
          title="Actionable empty"
          icon={<Icon name="plus" aria-hidden />}
          action={<button type="button">{actionLabel}</button>}
        />
      </>,
    );

    expect(screen.getByRole("dialog", { name: "Drawer" })).toHaveTextContent(
      "Drawer content",
    );
    expect(screen.getByText(metadataCopy)).toHaveAttribute(
      "data-text-truncate",
      "true",
    );
    expect(screen.getByText(stringStyleCopy)).toHaveAttribute(
      "data-text-style",
      "body",
    );
    // The open modal correctly makes background siblings inert; this assertion
    // checks that the sibling composition remains mounted without pretending it
    // should be exposed in the accessibility tree while the drawer is open.
    expect(screen.getByText(actionLabel)).toBeInTheDocument();
  });

  it("collapses an expanded drawer before a second downward swipe dismisses it", async () => {
    render(<ExpandableDrawerFixture />);
    const grabber = screen.getByTestId("overlays.full-page-drawer.grabber");
    const root = screen.getByTestId("overlays.full-page-drawer").parentElement;

    swipeGrabber(grabber, 600, 480, 1);
    await waitFor(() => expect(root).toHaveAttribute("data-expanded", "true"));

    swipeGrabber(grabber, 88, 120, 2);
    await waitFor(() => expect(root).toHaveAttribute("data-expanded", "false"));
    expect(
      screen.getByRole("dialog", { name: "Expandable drawer" }),
    ).toBeInTheDocument();

    swipeGrabber(grabber, 0, 120, 3);
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Expandable drawer" }),
      ).not.toBeInTheDocument(),
    );
  });
});
