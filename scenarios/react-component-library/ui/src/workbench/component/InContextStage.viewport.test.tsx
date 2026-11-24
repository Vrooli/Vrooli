import { create } from "@bufbuild/protobuf";
import { ComponentSchema } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { InContextStage, type Device, type InContextReport } from "./InContextStage";

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

function ResponsiveStage({ onReport }: { onReport: (report: InContextReport) => void }) {
  const [device, setDevice] = useState<Device>("desktop");

  return (
    <InContextStage
      asset={asset}
      story={story}
      version="1.2.0"
      template={DEFAULT_TEMPLATE}
      templates={[
        { id: DEFAULT_TEMPLATE.id, name: DEFAULT_TEMPLATE.name, summary: DEFAULT_TEMPLATE.tagline },
      ]}
      onTemplateChange={vi.fn()}
      device={device}
      onDeviceChange={setDevice}
      slotChoice={null}
      onSlotChoice={vi.fn()}
      onReport={onReport}
    />
  );
}

beforeEach(async () => {
  await setLocale("en");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("<html></html>", { status: 200 })));
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("InContextStage responsive viewport", () => {
  it("resets a custom width to the selected device's natural viewport", async () => {
    const onReport = vi.fn<(report: InContextReport) => void>();
    renderWithProviders(<ResponsiveStage onReport={onReport} />);

    const viewport = screen.getByRole("combobox", { name: "Viewport width" });
    fireEvent.change(viewport, { target: { value: "320" } });
    await waitFor(() => expect(onReport.mock.calls.at(-1)?.[0].viewport).toBe(320));

    fireEvent.click(screen.getByRole("button", { name: "Phone" }));

    await waitFor(() => {
      expect(viewport).toHaveValue("390");
      expect(screen.getByTestId("device-frame")).toHaveAttribute("data-device", "phone");
      expect(onReport.mock.calls.at(-1)?.[0]).toMatchObject({ device: "phone", viewport: 390 });
    });
  });
});
