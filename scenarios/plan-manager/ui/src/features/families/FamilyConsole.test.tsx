import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  EdgeKind,
  EdgeProvenance,
  FamilyEdgeSchema,
  GraphRevisionSchema,
  PlanFamilySchema,
  ReviewDecision,
} from "@vrooli/proto-types/plan-manager/v1/families/families_pb";

import * as api from "../../api/families";
import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { FamilyConsole } from "./FamilyConsole";

vi.mock("../../api/families");

const edge = create(FamilyEdgeSchema, { fromPlanId: "plan-b", toPlanId: "plan-a", kind: EdgeKind.DEPENDENCY, provenance: EdgeProvenance.INFERRED, reason: "plan-b consumes the contract produced by plan-a", claimIds: ["claim-a"] });
const family = create(PlanFamilySchema, { familyId: "family-1", slug: "coordination", outcome: "Coordinate safely", revision: 4n, graph: create(GraphRevisionSchema, { revision: 3n, edges: [edge], cyclic: false }) });

describe("FamilyConsole", () => {
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("shows edge evidence and cannot launch an unreviewed graph", async () => {
    vi.mocked(api.listFamilies).mockResolvedValue([family]);
    vi.mocked(api.getFamily).mockResolvedValue(family);
    vi.mocked(api.getFrontier).mockResolvedValue({ $typeName: "vrooli.plan_manager.v1.families.GetFrontierResponse", graphRevision: 3n, batches: [], launchable: false, diagnostics: ["current graph revision is not approved"] });
    renderWithProviders(<FamilyConsole />);
    fireEvent.change(await screen.findByTestId(selectors.families.select), { target: { value: "family-1" } });
    expect(await screen.findByTestId(selectors.families.edge({ index: 0 }))).toHaveTextContent("plan-b consumes the contract produced by plan-a");
    expect(screen.getByTestId(selectors.families.launchState)).toHaveTextContent("pages.families.notLaunchable");
    expect(screen.queryByRole("button", { name: /launch/i })).not.toBeInTheDocument();
  });

  it("submits a typed approval against the displayed revisions", async () => {
    vi.mocked(api.listFamilies).mockResolvedValue([family]);
    vi.mocked(api.getFamily).mockResolvedValue(family);
    vi.mocked(api.getFrontier).mockResolvedValue({ $typeName: "vrooli.plan_manager.v1.families.GetFrontierResponse", graphRevision: 3n, batches: [], launchable: false, diagnostics: [] });
    vi.mocked(api.reviewGraph).mockResolvedValue(family);
    renderWithProviders(<FamilyConsole />);
    fireEvent.change(await screen.findByTestId(selectors.families.select), { target: { value: "family-1" } });
    await screen.findByTestId(selectors.families.review);
    fireEvent.change(screen.getByLabelText("pages.families.reviewer"), { target: { value: "agent-1" } });
    fireEvent.change(screen.getByLabelText("pages.families.rationale"), { target: { value: "claims checked" } });
    fireEvent.click(screen.getByRole("button", { name: "pages.families.approve" }));
    await waitFor(() => expect(api.reviewGraph).toHaveBeenCalledWith(expect.objectContaining({ decision: ReviewDecision.APPROVED, reviewer: "agent-1", rationale: "claims checked" })));
  });
});
