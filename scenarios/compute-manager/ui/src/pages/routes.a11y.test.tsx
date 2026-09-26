import { cleanup, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InstanceState } from "@vrooli/proto-types/compute-manager/v1/instance/instance_pb";

import { selectors } from "../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { TestAppRouter } from "../app/routes";

vi.mock("../api/compute", () => ({
  fetchOpenFindings: vi.fn().mockResolvedValue({ findings: [] }),
  fetchInstance: vi.fn().mockResolvedValue({ instance: { id: "instance-1", state: InstanceState.RUNNING, provider: "fake", address: "198.51.100.1" } }),
  fetchInstances: vi.fn().mockResolvedValue({ instances: [] }),
}));

describe("compute routes accessibility", () => {
  afterEach(() => cleanup());

  it("renders findings without axe violations", async () => { // [REQ:COMPUTEM-P1-005]
    const { container } = renderWithProviders(<TestAppRouter initialEntries={["/findings"]} />, { withoutRouter: true });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.pages.findings)).toBeInTheDocument();
      expect(screen.getByText("pages.findings.empty")).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });

  it("renders an instance detail without axe violations", async () => { // [REQ:COMPUTEM-P1-005]
    const { container } = renderWithProviders(<TestAppRouter initialEntries={["/instances/instance-1"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByTestId(selectors.pages.instance)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
