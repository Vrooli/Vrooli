import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings.generated";
import {
  DispositionDecision,
  FindingSeverity,
  FindingStatus,
  type Finding,
} from "../../api/review";
import { ReviewFindingDetail } from "./ReviewFindingDetail";

function finding(overrides: Partial<Finding> = {}): Finding {
  return {
    $typeName: "vrooli.react_component_library.v1.review.Finding",
    id: "finding-1",
    subjectKind: "asset",
    subjectId: "Button",
    gate: "accessibility",
    severity: FindingSeverity.BLOCKER,
    status: FindingStatus.OPEN,
    message: "The control has no accessible name.",
    remediation: "Add an accessible label.",
    evidenceJson: '{"selector":"button"}',
    docsRef: "docs/accessibility/buttons.md",
    owner: "team-ui",
    createdAt: "2026-11-20T10:00:00Z",
    dispositions: [],
    ...overrides,
  };
}

function renderDetail(selected = finding(), overrides: Partial<React.ComponentProps<typeof ReviewFindingDetail>> = {}) {
  const props: React.ComponentProps<typeof ReviewFindingDetail> = {
    selected,
    copied: false,
    copyEvidence: vi.fn().mockResolvedValue(undefined),
    decision: DispositionDecision.FIX,
    setDecision: vi.fn(),
    owner: "",
    setOwner: vi.fn(),
    assignmentPending: false,
    onAssign: vi.fn(),
    reason: "",
    setReason: vi.fn(),
    actor: "reviewer@example.test",
    setActor: vi.fn(),
    dispositionPending: false,
    onRecord: vi.fn(),
    message: null,
    ...overrides,
  };
  const view = renderWithProviders(<ReviewFindingDetail {...props} />);
  return { props, rerender: view.rerender };
}

describe("ReviewFindingDetail", () => {
  beforeEach(async () => setLocale("en"));
  afterEach(() => cleanup());

  it("shows finding evidence and lets a reviewer copy it and open its documentation", () => {
    const props = renderDetail();
    const region = screen.getByRole("region", { name: i18n.t(strings.workbench.review.detail) });

    expect(region).toHaveTextContent(props.props.selected.message);
    expect(region).toHaveTextContent(props.props.selected.remediation);
    expect(region).toHaveTextContent("Blocker");
    expect(screen.getByRole("link", { name: i18n.t(strings.workbench.review.docs) })).toHaveAttribute(
      "href",
      "/docs/accessibility/buttons.md",
    );

    fireEvent.click(
      screen.getByRole("button", { name: i18n.t(strings.workbench.review.copyEvidence) }),
    );
    expect(props.props.copyEvidence).toHaveBeenCalledWith(props.props.selected);
    fireEvent.click(screen.getByText(i18n.t(strings.workbench.review.evidence)));
    expect(region).toHaveTextContent(props.props.selected.evidenceJson);
  });

  it("enables assignment and decision recording only when required fields are ready", () => {
    const props = renderDetail(finding({ remediation: "", docsRef: "", owner: "" }));
    const assignLabel = i18n.t(strings.workbench.review.assign);
    const recordLabel = i18n.t(strings.workbench.review.record);
    const assign = screen.getByRole("button", { name: assignLabel });
    const record = screen.getByRole("button", { name: recordLabel });

    expect(screen.queryByRole("link", { name: i18n.t(strings.workbench.review.docs) })).not.toBeInTheDocument();
    expect(assign).toBeDisabled();
    expect(record).toBeDisabled();

    fireEvent.change(screen.getByLabelText(i18n.t(strings.workbench.review.owner)), {
      target: { value: "owner@example.test" },
    });
    expect(props.props.setOwner).toHaveBeenCalledWith("owner@example.test");
    props.rerender(<ReviewFindingDetail {...props.props} owner="owner@example.test" />);
    fireEvent.click(screen.getByRole("button", { name: assignLabel }));
    expect(props.props.onAssign).toHaveBeenCalledOnce();
    fireEvent.change(screen.getByLabelText(i18n.t(strings.workbench.review.reason)), {
      target: { value: "Verified fix" },
    });
    expect(props.props.setReason).toHaveBeenCalledWith("Verified fix");
    props.rerender(<ReviewFindingDetail {...props.props} owner="owner@example.test" reason="Verified fix" />);
    fireEvent.click(screen.getByRole("button", { name: recordLabel }));
    expect(props.props.onRecord).toHaveBeenCalledOnce();
  });

  it("renders decision history, pending states, and status feedback", async () => {
    const props = renderDetail(
      finding({
        severity: FindingSeverity.LOW,
        status: FindingStatus.ACCEPTED,
        dispositions: [
          {
            $typeName: "vrooli.react_component_library.v1.review.FindingDisposition",
            id: "d-1",
            decision: DispositionDecision.ACCEPT_RISK,
            actor: "reviewer@example.test",
            at: "2026-11-20T11:00:00Z",
            reason: "Risk accepted by owner",
          },
        ],
      }),
      { copied: true, assignmentPending: true, dispositionPending: true, message: "Saved" },
    );

    expect(screen.getByRole("button", { name: i18n.t(strings.workbench.review.copied) })).toBeInTheDocument();
    const region = screen.getByRole("region", { name: i18n.t(strings.workbench.review.detail) });
    expect(region).toHaveTextContent(i18n.t(strings.workbench.review.statusAccepted));
    expect(region).toHaveTextContent(i18n.t(strings.workbench.review.acceptRisk));
    expect(region).toHaveTextContent("Risk accepted by owner");
    expect(screen.getByRole("status")).toHaveTextContent("Saved");
    expect(screen.getByRole("button", { name: i18n.t(strings.workbench.review.assign) })).toBeDisabled();
    expect(screen.getByRole("button", { name: i18n.t(strings.workbench.review.record) })).toBeDisabled();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: i18n.t(strings.workbench.review.decisionLabel) }));
    await user.click(screen.getByRole("option", { name: i18n.t(strings.workbench.review.retire) }));
    expect(props.props.setDecision).toHaveBeenCalledWith(DispositionDecision.RETIRE);
  });

  it("uses readable fallback labels for open findings and every recorded decision", () => {
    const statuses = [
      [FindingStatus.UNSPECIFIED, strings.workbench.review.statusOpen],
      [FindingStatus.ACKNOWLEDGED, strings.workbench.review.statusAcknowledged],
      [FindingStatus.FIXED, strings.workbench.review.statusFixed],
      [FindingStatus.RETIRED, strings.workbench.review.statusRetired],
    ] as const;
    for (const [status, label] of statuses) {
      const { unmount } = renderWithProviders(
        <ReviewFindingDetail
          {...renderProps(finding({ status }))}
        />,
      );
      expect(screen.getByText(i18n.t(label))).toBeInTheDocument();
      unmount();
    }

    const decisions = [
      [DispositionDecision.FIX, strings.workbench.review.fix],
      [DispositionDecision.ACKNOWLEDGE, strings.workbench.review.acknowledge],
      [DispositionDecision.ACCEPT_RISK, strings.workbench.review.acceptRisk],
      [DispositionDecision.RETIRE, strings.workbench.review.retire],
    ] as const;
    for (const [decision, label] of decisions) {
      const { unmount } = renderWithProviders(
        <ReviewFindingDetail
          {...renderProps(
            finding({
              dispositions: [
                {
                  $typeName: "vrooli.react_component_library.v1.review.FindingDisposition",
                  id: `decision-${decision}`,
                  decision,
                  actor: "reviewer@example.test",
                  at: "2026-11-20T11:00:00Z",
                  reason: "Reviewed",
                },
              ],
            }),
          )}
        />,
      );
      expect(screen.getByText(i18n.t(label), { selector: "b" })).toBeInTheDocument();
      unmount();
    }
  });
});

function renderProps(selected: Finding): React.ComponentProps<typeof ReviewFindingDetail> {
  return {
    selected,
    copied: false,
    copyEvidence: vi.fn().mockResolvedValue(undefined),
    decision: DispositionDecision.FIX,
    setDecision: vi.fn(),
    owner: "",
    setOwner: vi.fn(),
    assignmentPending: false,
    onAssign: vi.fn(),
    reason: "",
    setReason: vi.fn(),
    actor: "reviewer@example.test",
    setActor: vi.fn(),
    dispositionPending: false,
    onRecord: vi.fn(),
    message: null,
  };
}
