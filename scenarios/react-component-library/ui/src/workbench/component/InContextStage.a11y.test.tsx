import { create } from "@bufbuild/protobuf";
import { ComponentSchema } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { act, cleanup } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import type { ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { expectNoA11yViolations } from "../../test-utils";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { InContextStage } from "./InContextStage";

/**
 * The harness is an iframe with its own load protocol; the stage's own DOM is
 * what this file audits, so the specimen is replaced with a plain block.
 */
vi.mock("./FixtureFrame", () => ({
  FIXTURE_TIMEOUT_MS: 8_000,
  FixtureFrame: ({ title }: { title: string }) => <div data-testid="fixture-stub">{title}</div>,
}));

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

function stage(overrides: { story?: WorkbenchStory; device?: "desktop" | "phone" } = {}) {
  return (
    <InContextStage
      asset={asset}
      story={overrides.story ?? story}
      version="1.2.0"
      template={DEFAULT_TEMPLATE}
      templates={[
        { id: DEFAULT_TEMPLATE.id, name: DEFAULT_TEMPLATE.name, summary: DEFAULT_TEMPLATE.tagline },
      ]}
      onTemplateChange={vi.fn()}
      device={overrides.device ?? "desktop"}
      onDeviceChange={vi.fn()}
      slotChoice={null}
      onSlotChoice={vi.fn()}
      onReport={vi.fn()}
      states={{ total: 3, covered: 3, status: "fresh" }}
    />
  );
}

/**
 * The stage measures on a `requestAnimationFrame` and again on a 120 ms timer.
 * Settling inside `act` keeps those updates inside the test instead of leaking
 * act warnings into the console contract.
 */
async function settledRender(element: ReactElement) {
  const utils = renderWithProviders(element);
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 220));
  });
  return utils;
}

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
});

describe("InContextStage accessibility", () => {
  it("renders without axe violations", async () => {
    const { container } = await settledRender(stage());
    // axe probes focus and visibility, which can wake the stage's layout
    // observers; act keeps those updates out of the console contract.
    await act(async () => {
      await expectNoA11yViolations(container);
    });
  });

  it("renders the empty state without axe violations", async () => {
    const { container } = await settledRender(stage({ story: undefined, device: "phone" }));
    await act(async () => {
      await expectNoA11yViolations(container);
    });
  });

  it("exposes the state of the device switch and instrument toggles", async () => {
    await settledRender(stage());

    const desktop = document.querySelector<HTMLButtonElement>(
      ".wb-seg button[aria-pressed='true']",
    );
    expect(desktop?.textContent).toBe("Desktop");
    const pressed = Array.from(document.querySelectorAll(".wb-seg button[aria-pressed]")).map(
      (button) => button.getAttribute("aria-pressed"),
    );
    expect(pressed).toEqual(["true", "false", "false"]);

    const guides = Array.from(document.querySelectorAll<HTMLButtonElement>(".ic-tool")).find(
      (button) => button.textContent.includes("Guides"),
    );
    expect(guides?.getAttribute("aria-pressed")).toBe("true");

    const slotRows = Array.from(document.querySelectorAll<HTMLButtonElement>(".ic-slot-row"));
    expect(slotRows.length).toBeGreaterThan(0);
    expect(slotRows.filter((row) => row.getAttribute("aria-pressed") === "true")).toHaveLength(1);
  });

  it("announces placement in a polite live region", async () => {
    await settledRender(stage());
    const region = document.querySelector('[data-testid="in-context-announce"]');
    expect(region?.getAttribute("role")).toBe("status");
    expect(region?.getAttribute("aria-live")).toBe("polite");
    expect(region?.textContent).toMatch(/Placed in .+, \d+% affinity/);
  });

  it("keeps probe copies out of the accessibility tree and tab order", async () => {
    await settledRender(stage());
    const probe = document.querySelector(".ic-probe");
    expect(probe?.hasAttribute("inert")).toBe(true);
  });
});
