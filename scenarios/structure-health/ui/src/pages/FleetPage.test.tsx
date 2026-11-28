import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { ScanFleetResponseSchema } from "@vrooli/proto-types/structure-health/v1/fleet/fleet_pb";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { i18n, setLocale } from "../i18n";

vi.mock("../api/fleet", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/fleet")>();
  return { ...actual, fleetClient: { scanFleet: vi.fn() } };
});

import { FleetPage } from "./FleetPage";
import { fleetClient } from "../api/fleet";

describe("FleetPage", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the page heading and embeds the fleet view", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(create(ScanFleetResponseSchema, {}));
    renderWithProviders(<FleetPage />);

    expect(screen.getByTestId(selectors.pages.fleet)).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: i18n.t(strings.pages.fleet.title) }),
    ).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.view)).toBeInTheDocument());
  });
});
