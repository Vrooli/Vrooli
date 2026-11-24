import { create } from "@bufbuild/protobuf";
import { ComponentSchema } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { node, slotNode, type Template } from "../templates/spec";
import { InContextStage, type InContextReport } from "./InContextStage";

const asset: CatalogAsset = create(ComponentSchema, {
  id: "asset-metric",
  libraryId: "react-component-library:MonetizationAccount",
  displayName: "MonetizationAccount",
  slug: "MonetizationAccount",
  description: "Monthly recurring revenue metric",
  version: "1.2.0",
  latestVersion: "1.2.0",
  category: "analytics",
  catalogKind: "component",
  catalogId: "analytics.monetization",
  slot: "metric",
});

const story: WorkbenchStory = {
  key: "story-1",
  id: "default",
  name: "Default",
  role: "primary",
  args: {},
  fieldsJson: "{}",
  version: "1.2.0",
  sourcePath: "library/components/MonetizationAccount/versions/1.2.0/MonetizationAccount.tsx",
  contractId: "contract-1",
};

function renderStage(overrides: Partial<Parameters<typeof InContextStage>[0]> = {}) {
  const onReport = vi.fn<(report: InContextReport) => void>();
  const utils = renderWithProviders(
    <InContextStage
      asset={asset}
      story={story}
      version="1.2.0"
      template={DEFAULT_TEMPLATE}
      templates={[
        { id: DEFAULT_TEMPLATE.id, name: DEFAULT_TEMPLATE.name, summary: DEFAULT_TEMPLATE.tagline },
      ]}
      onTemplateChange={vi.fn()}
      device="desktop"
      onDeviceChange={vi.fn()}
      slotChoice={null}
      onSlotChoice={vi.fn()}
      onReport={onReport}
      states={null}
      {...overrides}
    />,
  );
  return { ...utils, onReport };
}

const lastReport = (onReport: ReturnType<typeof vi.fn<(report: InContextReport) => void>>) =>
  onReport.mock.calls.at(-1)?.[0];

/**
 * jsdom has no layout, so every probe reports a zero scroll width. Declaring
 * the probe's own width as its scroll width is the minimum that makes the
 * measurement real; a zero width stays unmeasured by design.
 */
function stubProbeLayout(extraWidth = 0): () => void {
  const spy = vi.spyOn(Element.prototype, "scrollWidth", "get").mockImplementation(function (
    this: Element,
  ) {
    const probe = this instanceof HTMLElement ? this.dataset.icProbe : undefined;
    if (!probe) return 0;
    return Number(probe) + extraWidth;
  });
  return () => spy.mockRestore();
}

let restores: Array<() => void> = [];

beforeEach(async () => {
  await setLocale("en");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("<html></html>", { status: 200 })));
});

afterEach(() => {
  for (const restore of restores) restore();
  restores = [];
  vi.unstubAllGlobals();
});

describe("InContextStage measurements", () => {
  it("announces the placement and reports measured rows", async () => {
    restores.push(stubProbeLayout());
    const { onReport } = renderStage();

    expect(screen.getByTestId("in-context-announce").textContent).toContain("Placed in");

    await act(async () => {
      await Promise.resolve();
    });

    const report = lastReport(onReport);
    expect(report).toBeDefined();
    const overflow = report?.rows.find((row) => row.id === "overflow");
    expect(overflow?.verdict).toBe("pass");
    expect(overflow?.evidence).toBe("measured");
  });

  it("marks overflow unmeasured when the probe has no layout", async () => {
    const { onReport } = renderStage();
    await act(async () => {
      await Promise.resolve();
    });
    const overflow = lastReport(onReport)?.rows.find((row) => row.id === "overflow");
    expect(overflow?.verdict).toBe("unmeasured");
    expect(overflow?.evidence).toBe("none");
  });

  it("fails overflow when a reference width is wider than its page", async () => {
    restores.push(stubProbeLayout(12));
    const { onReport } = renderStage();
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastReport(onReport)?.rows.find((row) => row.id === "overflow")?.verdict).toBe("fail");
  });

  it("renders a clear state with a next action when there is no story", () => {
    const { container } = renderStage({ story: undefined });
    expect(screen.getByText(/No story selected/i)).toBeInTheDocument();
    expect(container.querySelector("iframe")).toBeNull();
  });

  it("applies page typography to the empty story slot after measurement", async () => {
    const { container } = renderStage({ story: undefined });

    await waitFor(() => {
      const emptySlot = container.querySelector<HTMLElement>(".ic-empty-slot");
      expect(emptySlot?.getAttribute("style")).toBeTruthy();
      expect(emptySlot?.style.color).toBeTruthy();
    });
  });

  it("renders a clear state when the page declares no slot", () => {
    const slotless: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-slotless",
      name: "Slotless",
      spec: node("slotless.page", "page", { pad: 28 }, [
        node("slotless.title", "text", { text: "No slot here", variant: "display" }),
      ]),
    };
    renderStage({ template: slotless });

    expect(screen.getByText(/No slot on this page/i)).toBeInTheDocument();
    expect(screen.getByTestId("in-context-announce").textContent).toContain("No slot");
  });

  it("cancels measurement work when it unmounts mid-measurement", async () => {
    restores.push(stubProbeLayout());
    const cancel = vi.spyOn(window, "cancelAnimationFrame");
    const clear = vi.spyOn(window, "clearTimeout");
    restores.push(
      () => cancel.mockRestore(),
      () => clear.mockRestore(),
    );
    const { unmount } = renderStage();

    unmount();
    await act(async () => {
      await Promise.resolve();
    });

    expect(cancel).toHaveBeenCalled();
    expect(clear).toHaveBeenCalled();
  });

  it("keeps the squint check unmeasured when Blur is on", async () => {
    restores.push(stubProbeLayout());
    const { onReport } = renderStage();
    await act(async () => {
      await Promise.resolve();
    });

    act(() => {
      screen.getByRole("button", { name: "Blur" }).click();
    });

    const squint = lastReport(onReport)?.rows.find((row) => row.id === "squint");
    expect(squint?.verdict).toBe("unmeasured");
    expect(squint?.scored).toBe(false);
  });
});

describe("InContextStage slot typing", () => {
  it("warns instead of passing a slot that declares no accepted kinds", async () => {
    const open: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-open-slot",
      name: "Open slot",
      spec: node("open.page", "page", { pad: 28 }, [
        node("open.title", "text", { text: "Open", variant: "display" }),
        node("open.slot", "slot", { hint: "Anything fits", minH: 48 }),
      ]),
    };
    const { onReport } = renderStage({ template: open });
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastReport(onReport)?.rows.find((row) => row.id === "slot")?.verdict).toBe("warn");
  });

  it("renders a next action when no slot accepts the component's kind", async () => {
    const unsupported: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-unsupported-slot",
      name: "Action only",
      spec: node("unsupported.page", "page", { pad: 28 }, [
        node("unsupported.title", "text", { text: "Action only", variant: "display" }),
        slotNode("unsupported.slot", {
          hint: "Header action",
          minH: 48,
          role: "action",
          accepts: ["button"],
        }),
      ]),
    };
    renderStage({ template: unsupported });
    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText(/No slot on this page declares metric/i)).toBeInTheDocument();
  });

  it("passes a typed slot that accepts the component's kind", async () => {
    const typed: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-typed-slot",
      name: "Typed slot",
      spec: node("typed.page", "page", { pad: 28 }, [
        node("typed.title", "text", { text: "Typed", variant: "display" }),
        slotNode("typed.slot", { hint: "KPI card", minH: 48, role: "metric", accepts: ["metric"] }),
      ]),
    };
    const { onReport } = renderStage({ template: typed });
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastReport(onReport)?.rows.find((row) => row.id === "slot")?.verdict).toBe("pass");
  });
});

describe("InContextStage interactions", () => {
  it("supports keyboard toggling instruments and reports their enabled state", async () => {
    const user = userEvent.setup();
    const { onReport } = renderStage();
    const guides = screen.getByRole("button", { name: "Guides" });

    await user.click(guides);
    expect(guides).toHaveAttribute("aria-pressed", "false");
    await user.keyboard(" ");
    expect(guides).toHaveAttribute("aria-pressed", "true");

    await user.click(screen.getByRole("button", { name: "Targets" }));
    await waitFor(() => expect(lastReport(onReport)?.instruments.targets).toBe(false));
    expect(lastReport(onReport)?.rows.find((row) => row.id === "touch")?.detail).toContain(
      "Turn on Targets",
    );

    await user.click(screen.getByRole("button", { name: "Contrast" }));
    await waitFor(() => expect(lastReport(onReport)?.instruments.contrast).toBe(false));
    expect(lastReport(onReport)?.rows.find((row) => row.id === "contrast")?.detail).toContain(
      "Turn on Contrast",
    );
  });

  it("updates the contrast probe when a text node in the page is clicked", async () => {
    const { container, onReport } = renderStage();
    const heading = container.querySelector<HTMLElement>('[data-node-id="workspace.title"]');
    expect(heading).not.toBeNull();

    fireEvent.click(heading!);

    await waitFor(() => {
      expect(lastReport(onReport)?.rows.find((row) => row.id === "contrast")?.detail).toContain(
        "Start with one clear idea.",
      );
    });
  });

  it("lets a manually pinned slot be changed and reset to automatic placement", async () => {
    const multiSlot: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-multiple-slots",
      name: "Multiple slots",
      spec: node("multi.page", "page", { pad: 28 }, [
        node("multi.title", "text", { text: "Choose a slot", variant: "display" }),
        slotNode("multi.first", { hint: "First metric", role: "metric", accepts: ["metric"] }),
        slotNode("multi.second", { hint: "Second metric", role: "metric", accepts: ["metric"] }),
      ]),
    };
    const onSlotChoice = vi.fn();
    renderStage({ template: multiSlot, slotChoice: "multi.second", onSlotChoice });

    const choices = screen.getAllByRole("button", { name: /metric/i });
    expect(choices).toHaveLength(2);
    expect(choices[1]).toHaveAttribute("aria-pressed", "true");
    const firstChoice = choices[0];
    if (!firstChoice) throw new Error("Expected the first metric slot choice");
    await userEvent.setup().click(firstChoice);
    expect(onSlotChoice).toHaveBeenCalledWith("multi.first");

    await userEvent.setup().click(screen.getByRole("button", { name: "Reset to authored slot" }));
    expect(onSlotChoice).toHaveBeenLastCalledWith(null);
  });

  it("explains when a manually pinned slot differs from the best scoring slot", async () => {
    const multiSlot: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-manual-placement-explanation",
      name: "Multiple slots",
      spec: node("multi.page", "page", { pad: 28 }, [
        node("multi.title", "text", { text: "Choose a slot", variant: "display" }),
        slotNode("multi.first", { hint: "First metric", role: "metric", accepts: ["metric"] }),
        slotNode("multi.second", { hint: "Second metric", role: "metric", accepts: ["metric"] }),
      ]),
    };
    renderStage({ template: multiSlot, slotChoice: "multi.second" });
    await act(async () => {
      await Promise.resolve();
    });

    expect(screen.getByText(/Second metric was chosen by hand/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Second metric/i })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("explains when the manually pinned slot is already the best scoring slot", async () => {
    const multiSlot: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-pinned-best-placement",
      name: "Multiple slots",
      spec: node("multi.page", "page", { pad: 28 }, [
        node("multi.title", "text", { text: "Choose a slot", variant: "display" }),
        slotNode("multi.first", { hint: "First metric", role: "metric", accepts: ["metric"] }),
        slotNode("multi.second", { hint: "Second metric", role: "metric", accepts: ["metric"] }),
      ]),
    };
    renderStage({ template: multiSlot, slotChoice: "multi.first" });
    await act(async () => {
      await Promise.resolve();
    });

    expect(
      screen.getByText(/First metric is pinned by hand; it is also the best score/i),
    ).toBeInTheDocument();
  });

  it("updates the tested viewport when a width preset is selected", async () => {
    const user = userEvent.setup();
    const { onReport } = renderStage();
    const viewport = screen.getByRole("combobox", { name: "Viewport width" });

    await user.selectOptions(viewport, "320");

    expect(viewport).toHaveValue("320");
    await waitFor(() => expect(lastReport(onReport)?.viewport).toBe(320));
    expect(screen.getByText(/320 × 640/)).toBeInTheDocument();
  });

  it("follows the selected device's natural viewport after a custom width", async () => {
    const user = userEvent.setup();
    const onDeviceChange = vi.fn();
    const view = renderStage({ onDeviceChange });

    await user.selectOptions(screen.getByRole("combobox", { name: "Viewport width" }), "320");
    await user.click(screen.getByRole("button", { name: "Phone" }));
    expect(onDeviceChange).toHaveBeenCalledWith("phone");

    view.rerender(
      <InContextStage
        asset={asset}
        story={story}
        version="1.2.0"
        template={DEFAULT_TEMPLATE}
        templates={[
          {
            id: DEFAULT_TEMPLATE.id,
            name: DEFAULT_TEMPLATE.name,
            summary: DEFAULT_TEMPLATE.tagline,
          },
        ]}
        onTemplateChange={vi.fn()}
        device="phone"
        onDeviceChange={onDeviceChange}
        slotChoice={null}
        onSlotChoice={vi.fn()}
        onReport={view.onReport}
        states={null}
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole("combobox", { name: "Viewport width" })).toHaveValue("390");
    });
    expect(lastReport(view.onReport)?.device).toBe("phone");
    expect(lastReport(view.onReport)?.viewport).toBe(390);
  });

  it("shows a recoverable error when the harness request fails and retries on reload", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new Error("network unavailable"));
    vi.stubGlobal("fetch", fetchMock);
    const { onReport } = renderStage();

    expect(await screen.findByText(/The specimen could not load/i)).toBeInTheDocument();
    await waitFor(() => expect(lastReport(onReport)?.status).toBe("error"));

    await userEvent.setup().click(await screen.findByRole("button", { name: "Try again" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(await screen.findByText(/The specimen could not load/i)).toBeInTheDocument();

    const reloadButtons = screen.getAllByRole("button", { name: "Reload specimen" });
    await userEvent.setup().click(reloadButtons.at(-1)!);
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
    expect(await screen.findByText(/The specimen could not load/i)).toBeInTheDocument();
  });

  it("explains an unavailable harness response and offers a retry", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("unavailable", { status: 503 })));
    const { onReport } = renderStage();

    expect(await screen.findByText(/preview harness returned HTTP 503/i)).toBeInTheDocument();
    expect(screen.getByRole("alert")).toBeInTheDocument();
    await waitFor(() => expect(lastReport(onReport)?.status).toBe("error"));
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });

  it("reports an empty story slot without mounting a specimen frame", async () => {
    const { container, onReport } = renderStage({ story: undefined });

    await waitFor(() => expect(lastReport(onReport)?.hasStory).toBe(false));
    expect(lastReport(onReport)?.storyId).toBeNull();
    expect(container.querySelector("iframe")).toBeNull();
    expect(screen.getByText(/No story selected/i)).toBeInTheDocument();
  });
});
