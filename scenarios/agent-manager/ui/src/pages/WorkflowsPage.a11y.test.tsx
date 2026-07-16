import { afterEach, describe, it, vi } from "vitest";
import { cleanup, render } from "@testing-library/react";
import { expectNoA11yViolations } from "../test-utils/a11y";
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
    const { container } = render(<WorkflowsPage />);
    await expectNoA11yViolations(container);
  });
});
