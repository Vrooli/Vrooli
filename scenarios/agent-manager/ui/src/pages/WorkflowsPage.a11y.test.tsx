import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup } from "@testing-library/react";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "../test-utils";
import { WorkflowsPage } from "./WorkflowsPage";

vi.mock("../hooks/useApi", () => ({
  useWorkflowExecutions: () => ({
    data: [],
    loading: false,
    error: null,
    refetch: vi.fn(),
    getTrace: vi.fn(),
    control: vi.fn(),
    signal: vi.fn(),
  }),
}));

afterEach(cleanup);

describe("WorkflowsPage accessibility", () => {
  it("has no axe violations in its empty operator state", async () => {
    const { container } = renderWithProviders(<WorkflowsPage />);
    expect(container).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
