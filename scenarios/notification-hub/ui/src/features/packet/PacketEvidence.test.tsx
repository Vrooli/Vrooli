import { beforeEach, describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import { FollowthroughSchema, RelatedAskSchema } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { en } from "../../i18n/catalog";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { makeAsk } from "./mocks/askFactory";
import { FollowthroughBanner, ImpactFacts, PacketEvidence, ReadinessPanel, RelatedDecisions } from "./PacketEvidence";

const show = (node: React.ReactElement) => renderWithProviders(node);

const RESULT_URL = "https://example.test/run/42";
const FINDING_BLOCKER = "The brief is not attached.";
const OWNER_RESEARCHER = "Owner: researcher";
const FIELD_ATTACHMENTS = "Field: attachments";
const OWNER_LINE = "Owner: Steward";
const SUMMARY = "Both floors were restructured.";
const BOUND_30 = "Bound 30 min";
const SIMILAR_WORDING = "Similar wording 0.72";
const FOURTH_KIN = "Fourth kin";
const UNBLOCKS = "Unblocks: three Portal grants";
const COST = "Cost of waiting: the report stays stale";
const REPORT = JSON.stringify({
  grade: "needs_work",
  score: 0.6,
  findings: [
    { code: "packet.missing", severity: "blocker", field: "attachments", owner: "researcher", message: "The brief is not attached." },
    { code: "cover.restates", severity: "warning", field: "question", owner: "researcher", message: "The page restates the cover." },
  ],
  basis: [],
});

describe("packet readiness", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("shows the grade and its findings, with severity, message and owner", () => {
    show(<ReadinessPanel ask={makeAsk({ qualityGrade: "needs_work", qualityReport: REPORT })} />);
    expect(screen.getByRole("heading", { name: en.packet.readiness.title })).toBeInTheDocument();
    expect(screen.getByText(en.asks.row.readiness.needsWork)).toBeInTheDocument();
    expect(screen.getByText(FINDING_BLOCKER)).toBeInTheDocument();
    expect(screen.getByText(en.packet.readiness.severity.blocker)).toBeInTheDocument();
    expect(screen.getByText(en.packet.readiness.severity.warning)).toBeInTheDocument();
    expect(screen.getByText(FIELD_ATTACHMENTS)).toBeInTheDocument();
    expect(screen.getAllByText(OWNER_RESEARCHER)).toHaveLength(2);
  });

  it("keeps the grade when the report does not parse", () => {
    show(<ReadinessPanel ask={makeAsk({ qualityGrade: "ready", qualityReport: "{not json" })} />);
    expect(screen.getByTestId("packet-readiness")).toBeInTheDocument();
    expect(screen.getByText(en.asks.row.readiness.ready)).toBeInTheDocument();
    expect(screen.getByText(en.packet.readiness.unavailable)).toBeInTheDocument();
  });

  it("says a parsed report has no findings instead of hiding the grade", () => {
    show(<ReadinessPanel ask={makeAsk({ qualityReport: JSON.stringify({ grade: "ready", findings: [] }) })} />);
    expect(screen.getByText(en.asks.row.readiness.ready)).toBeInTheDocument();
    expect(screen.getByText(en.packet.readiness.none)).toBeInTheDocument();
  });
});

describe("related decisions", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("shows up to three suggestions with the evidence that connects them", () => {
    const related = [
      create(RelatedAskSchema, { askId: "a", title: "First kin", basis: "family", score: 0.9, sender: "Steward" }),
      create(RelatedAskSchema, { askId: "b", question: "Second kin", basis: "subject", score: 0.8 }),
      create(RelatedAskSchema, { askId: "c", title: "Third kin", basis: "lexical", score: 0.72 }),
      create(RelatedAskSchema, { askId: "d", title: "Fourth kin", basis: "semantic", score: 0.74 }),
    ];
    show(<RelatedDecisions related={related} />);
    const links = screen.getAllByRole("link");
    expect(links).toHaveLength(3);
    expect(links[0]).toHaveAttribute("href", "/asks/a");
    expect(screen.getByText(en.packet.related.basis.family)).toBeInTheDocument();
    expect(screen.getByText(en.packet.related.basis.subject)).toBeInTheDocument();
    expect(screen.getByText(SIMILAR_WORDING)).toBeInTheDocument();
    expect(screen.queryByText(FOURTH_KIN)).not.toBeInTheDocument();
    expect(screen.getByText(en.packet.related.note)).toBeInTheDocument();
  });

  it("draws nothing when the Hub suggests none", () => {
    show(<RelatedDecisions related={[]} />);
    expect(screen.queryByTestId("packet-related")).not.toBeInTheDocument();
  });
});

describe("follow-through", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("shows what the owner reported and links a safe result", () => {
    const followthrough = create(FollowthroughSchema, {
      state: "done",
      owner: "Steward",
      boundSeconds: 1800,
      finishedAt: "2026-01-01T02:00:00.000Z",
      resultUrl: "https://example.test/run/42",
      summary: "Both floors were restructured.",
    });
    show(<FollowthroughBanner ask={makeAsk({ state: "answered", followthrough })} />);
    expect(screen.getByText(en.packet.followthrough.state.done)).toBeInTheDocument();
    expect(screen.getByText(OWNER_LINE)).toBeInTheDocument();
    expect(screen.getByText(BOUND_30)).toBeInTheDocument();
    expect(screen.getByText(SUMMARY)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: en.packet.followthrough.result })).toHaveAttribute("href", RESULT_URL);
  });

  it("says an answered ask has no owner report, and never invents progress", () => {
    show(<FollowthroughBanner ask={makeAsk({ state: "answered" })} />);
    expect(screen.getByText(en.packet.followthrough.absent)).toBeInTheDocument();
  });

  it("stays out of the way while the decision is still open", () => {
    show(<FollowthroughBanner ask={makeAsk({ state: "pending" })} />);
    expect(screen.queryByTestId("packet-followthrough")).not.toBeInTheDocument();
  });

  it("does not link a result URL that is not http(s)", () => {
    const followthrough = create(FollowthroughSchema, { state: "running", resultUrl: "javascript:alert(1)" });
    show(<FollowthroughBanner ask={makeAsk({ state: "answered", followthrough })} />);
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});

describe("impact facts", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("names what the answer unblocks and what waiting costs, one line each", () => {
    show(<ImpactFacts ask={makeAsk({ unblocks: "three Portal grants", costOfWaiting: "the report stays stale" })} />);
    expect(screen.getByText(UNBLOCKS)).toBeInTheDocument();
    expect(screen.getByText(COST)).toBeInTheDocument();
  });

  it("is absent when the producer sent neither fact", () => {
    show(<ImpactFacts ask={makeAsk()} />);
    expect(screen.queryByTestId("packet-impact")).not.toBeInTheDocument();
  });

  it("is included in the packet's evidence block", () => {
    show(<PacketEvidence ask={makeAsk({ qualityReport: REPORT, unblocks: "a release" })} />);
    expect(screen.getByTestId("packet-readiness")).toBeInTheDocument();
    expect(screen.getByTestId("packet-impact")).toBeInTheDocument();
  });
});
