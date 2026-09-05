import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BaselineCollectionSchema } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import * as api from "../../lib/api-baseline-collections";
import { BaselineCollectionsPanel } from "./BaselineCollectionsPanel";

vi.mock("../../lib/api-baseline-collections", () => ({
  getBaselineCollection: vi.fn(),
  diffBaselineCollection: vi.fn(),
}));

beforeEach(() => vi.clearAllMocks());

describe("BaselineCollectionsPanel", () => {
  it("renders durable coverage and keeps path evidence informational", async () => {
    vi.mocked(api.getBaselineCollection).mockResolvedValue(create(BaselineCollectionSchema, {
      name: "plan-before",
      branch: "agi",
      coverage: { required: 2, ready: 2, complete: true },
      members: [{ scenario: "git-control-tower", required: true, status: "ready" }],
      pathSnapshots: [{ name: "paths-before", branch: "agi" }],
    }));
    render(<BaselineCollectionsPanel repoId="1" />);

    fireEvent.change(screen.getByLabelText("Collection name"), { target: { value: "plan-before" } });
    fireEvent.click(screen.getByRole("button", { name: "Load" }));

    expect(await screen.findByText(/coverage: 2\/2 ready/)).toBeInTheDocument();
    expect(screen.getByText(/git-control-tower: ready/)).toBeInTheDocument();
    expect(screen.getByText(/Informational source evidence: paths-before/)).toBeInTheDocument();
  });
});
