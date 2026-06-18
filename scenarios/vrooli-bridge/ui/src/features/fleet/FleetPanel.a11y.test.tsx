/**
 * FleetPanel accessibility regression tests.
 *
 * The fleet feature owns its query-backed loading/success/empty/error UI, so
 * a11y coverage lives here. Presence must be conveyed without relying on color
 * alone — the populated assertion exercises the labelled presence dots.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { NodeStatus } from "../../api/nodes";
import { makeNode } from "./mocks/factories";

const { listNodes, revokeNode } = vi.hoisted(() => ({
  listNodes: vi.fn(),
  revokeNode: vi.fn(),
}));

vi.mock("../../api/nodes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/nodes")>();
  return {
    ...actual,
    nodesClient: { listNodes, revokeNode },
  };
});

import { FleetPanel } from "./FleetPanel";

describe("FleetPanel accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a populated fleet without axe violations", async () => {
    listNodes.mockResolvedValue({
      nodes: [
        makeNode({ id: "n1", name: "ubuntu-ci", status: NodeStatus.ONLINE, online: true }),
        makeNode({ id: "n2", name: "mac-mini", status: NodeStatus.OFFLINE, online: false }),
      ],
    });
    const { container } = renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.list)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });

  it("renders the empty state without axe violations", async () => {
    listNodes.mockResolvedValue({ nodes: [] });
    const { container } = renderWithProviders(<FleetPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
