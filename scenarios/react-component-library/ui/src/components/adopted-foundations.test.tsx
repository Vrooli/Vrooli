import { useState } from "react";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Drawer } from "./Drawer";
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
      <Drawer open={open} onClose={() => setOpen(false)}>
        <Text as="h2" textStyle="title">
          Drawer fixture
        </Text>
      </Drawer>
    </>
  );
}

describe("adopted foundation entry points", () => {
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
    expect(screen.getByRole("dialog", { name: "Drawer" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.queryByRole("dialog", { name: "Drawer" })).not.toBeInTheDocument();
  });

  it("keeps the drawer fallback content available for minimal consumers", () => {
    renderWithProviders(
      <>
        <Drawer open side="left" presentation="non-modal" />
        <Text style={{ letterSpacing: "0.02em" }} textStyle="caption" truncate balance numeric>
          Styled metadata
        </Text>
        <Text style="body">String style</Text>
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
    expect(screen.getByRole("button", { name: "Create" })).toBeInTheDocument();
  });
});
