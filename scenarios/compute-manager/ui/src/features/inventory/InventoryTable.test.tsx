import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { InstanceState } from "@vrooli/proto-types/compute-manager/v1/instance/instance_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { InventoryTable } from "../../components/InventoryTable";

const instance = { id: "instance-1", state: InstanceState.RUNNING, region: "fsn1", size: "small", remainingSeconds: 3600n };

describe("InventoryTable", () => {
  afterEach(() => cleanup());

  it("shows a loading state", () => { // [REQ:COMPUTEM-P1-005]
    renderWithProviders(<InventoryTable instances={[]} loading />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("shows a request error", () => { // [REQ:COMPUTEM-P1-005]
    renderWithProviders(<InventoryTable instances={[]} loading={false} error="inventory failed" />);
    expect(screen.getByRole("alert")).toHaveTextContent("inventory failed");
  });

  it("shows an empty state", () => { // [REQ:COMPUTEM-P1-005]
    renderWithProviders(<InventoryTable instances={[]} loading={false} />);
    expect(screen.getByTestId(selectors.pages.dashboardPlaceholder)).toHaveTextContent(strings.pages.dashboard.empty);
  });

  it("shows populated inventory", () => { // [REQ:COMPUTEM-P1-005]
    renderWithProviders(<InventoryTable instances={[instance]} loading={false} />);
    const table = screen.getByTestId(selectors.pages.dashboardPlaceholder);
    expect(table).toHaveTextContent("3");
    expect(table).toHaveTextContent("fsn1");
    expect(table).toHaveTextContent("3600s");
  });
});
