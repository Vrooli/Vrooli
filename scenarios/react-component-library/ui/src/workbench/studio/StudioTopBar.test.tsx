import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import type { DesignDocument } from "../design-types";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { emptyStack, type CommandStack } from "./history";
import { StudioTopBar } from "./StudioTopBar";

function design(id: string, name: string): DesignDocument {
  return {
    id,
    name,
    updatedAt: "2026-11-20T10:00:00.000Z",
    revision: 3,
    frames: [],
    theme: { ...DEFAULT_TEMPLATE.palette },
  };
}

function setup(overrides: Partial<React.ComponentProps<typeof StudioTopBar>> = {}) {
  const props: React.ComponentProps<typeof StudioTopBar> = {
    designs: [design("open", "Landing page"), design("saved", "Pricing page")],
    openIds: ["open"],
    activeId: "open",
    onSelect: vi.fn(),
    onClose: vi.fn(),
    onNew: vi.fn(),
    onImportHtml: vi.fn(),
    onOpen: vi.fn(),
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
    canUndo: true,
    canRedo: false,
    onUndo: vi.fn(),
    onRedo: vi.fn(),
    history: emptyStack(),
    onJump: vi.fn(),
    device: "desktop",
    onDevice: vi.fn(),
    onShare: vi.fn(),
    onPresent: vi.fn(),
    dirty: true,
    ...overrides,
  };
  renderWithProviders(<StudioTopBar {...props} />);
  return props;
}

describe("StudioTopBar", () => {
  beforeEach(async () => {
    vi.useFakeTimers();
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("opens saved designs and creates or imports from the add menu", () => {
    const props = setup();

    fireEvent.click(screen.getByRole("button", { name: "Open a design" }));
    expect(screen.getByRole("dialog", { name: "Open a design" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Pricing page/ }));
    expect(props.onOpen).toHaveBeenCalledWith("saved");
    expect(screen.queryByRole("dialog", { name: "Open a design" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Open a design" }));
    fireEvent.click(screen.getByRole("button", { name: "Blank design" }));
    expect(props.onNew).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("button", { name: "Open a design" }));
    fireEvent.click(screen.getByRole("button", { name: "Import an HTML page" }));
    expect(props.onImportHtml).toHaveBeenCalledOnce();
  });

  it("routes tab, device, history, undo, redo, share, and present actions", () => {
    const history = {
      index: 1,
      entries: [
        {
          id: "edit-1",
          label: "Changed heading",
          at: Date.now(),
          before: design("open", "Landing page"),
          after: design("open", "Landing page"),
        },
      ],
    } as CommandStack<DesignDocument>;
    const props = setup({
      designs: [design("open", "Landing page"), design("other", "Dashboard")],
      openIds: ["open", "other"],
      canRedo: true,
      history,
    });

    fireEvent.click(screen.getByRole("tab", { name: /Dashboard/ }));
    expect(props.onSelect).toHaveBeenCalledWith("other");
    fireEvent.click(screen.getByRole("button", { name: "Close Dashboard" }));
    expect(props.onClose).toHaveBeenCalledWith("other");
    fireEvent.click(screen.getByRole("button", { name: "Undo" }));
    fireEvent.click(screen.getByRole("button", { name: "Redo" }));
    expect(props.onUndo).toHaveBeenCalledOnce();
    expect(props.onRedo).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: "Tablet" }));
    expect(props.onDevice).toHaveBeenCalledWith("tablet");

    fireEvent.click(screen.getByRole("button", { name: "History" }));
    expect(screen.getByRole("dialog", { name: "History" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Changed heading/ }));
    expect(props.onJump).toHaveBeenCalledWith(1);
    expect(screen.queryByRole("dialog", { name: "History" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Duplicate this design" }));
    fireEvent.click(screen.getByRole("button", { name: "Share" }));
    fireEvent.click(screen.getByRole("button", { name: "Present" }));
    expect(props.onDuplicate).toHaveBeenCalledOnce();
    expect(props.onShare).toHaveBeenCalledOnce();
    expect(props.onPresent).toHaveBeenCalledOnce();
  });

  it("requires a second delete action, times out confirmation, and protects the last design", async () => {
    const props = setup();
    const deleteButton = screen.getByRole("button", { name: "Delete this design" });
    fireEvent.click(deleteButton);
    expect(props.onDelete).not.toHaveBeenCalled();
    expect(screen.getByRole("status")).toHaveTextContent("Click again to delete");
    fireEvent.click(screen.getByRole("button", { name: "Confirm delete design" }));
    expect(props.onDelete).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("button", { name: "Delete this design" }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(4_000);
    });
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete this design" })).toBeInTheDocument();

    cleanup();
    setup({ designs: [design("only", "Only design")], openIds: ["only"], activeId: "only" });
    expect(screen.getByRole("button", { name: "Delete this design" })).toBeDisabled();
  });
});
