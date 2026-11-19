import { create } from "@bufbuild/protobuf";
import { ComponentSchema } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { LOCAL_TEMPLATE_KEY } from "../templates/local";
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
  const utils = render(
    <InContextStage
      asset={asset}
      story={story}
      version="1.2.0"
      templateId={DEFAULT_TEMPLATE.id}
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
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response("<html></html>", { status: 200 })),
  );
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

  it("renders a clear state when the page declares no slot", () => {
    const slotless: Template = {
      ...DEFAULT_TEMPLATE,
      id: "test-slotless",
      name: "Slotless",
      spec: node("slotless.page", "page", { pad: 28 }, [
        node("slotless.title", "text", { text: "No slot here", variant: "display" }),
      ]),
    };
    window.localStorage.setItem(LOCAL_TEMPLATE_KEY, JSON.stringify([{ ...slotless, local: true }]));

    renderStage({ templateId: slotless.id });

    expect(screen.getByText(/No slot on this page/i)).toBeInTheDocument();
    expect(screen.getByTestId("in-context-announce").textContent).toContain("No slot");
  });

  it("cancels measurement work when it unmounts mid-measurement", async () => {
    restores.push(stubProbeLayout());
    const cancel = vi.spyOn(window, "cancelAnimationFrame");
    const clear = vi.spyOn(window, "clearTimeout");
    restores.push(() => cancel.mockRestore(), () => clear.mockRestore());
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
    window.localStorage.setItem(LOCAL_TEMPLATE_KEY, JSON.stringify([{ ...open, local: true }]));

    const { onReport } = renderStage({ templateId: open.id });
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
    window.localStorage.setItem(
      LOCAL_TEMPLATE_KEY,
      JSON.stringify([{ ...unsupported, local: true }]),
    );

    renderStage({ templateId: unsupported.id });
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
    window.localStorage.setItem(LOCAL_TEMPLATE_KEY, JSON.stringify([{ ...typed, local: true }]));

    const { onReport } = renderStage({ templateId: typed.id });
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastReport(onReport)?.rows.find((row) => row.id === "slot")?.verdict).toBe("pass");
  });
});
