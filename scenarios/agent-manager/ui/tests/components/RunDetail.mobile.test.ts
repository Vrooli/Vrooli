import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, useState, type ReactNode } from "react";
import { test, vi } from "vitest";
import { RunDetail } from "../../src/components/RunDetail.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";

vi.mock("../../src/hooks/useViewportSize.js", () => ({
  useViewportSize: () => ({ isDesktop: false }),
}));
// Mobile action tests exercise the RunDetail shell; the report transport is
// covered independently. Keeping it synchronous prevents jsdom's relative
// fetch rejection/retry handles from outliving this worker.
vi.mock("../../src/hooks/useApi.js", () => ({
  useRunReport: () => ({ data: null, loading: false, error: null, refetch: vi.fn() }),
}));
vi.mock("../../src/components/RunTimeline.js", () => ({
  RunTimeline: () => createElement("div", null, "timeline"),
}));
vi.mock("../../src/components/runs/FallbackTimeline.js", () => ({
  FallbackTimeline: () => null,
}));

function MobileHeaderHost({ children }: { children: (props: { onMobileHeaderLeft: (node: ReactNode | null) => void; onMobileHeaderRight: (node: ReactNode | null) => void }) => ReactNode }) {
  const [left, setLeft] = useState<ReactNode>(null);
  const [right, setRight] = useState<ReactNode>(null);
  return createElement("div", null,
    createElement("div", { "data-testid": "mobile-header-left" }, left),
    createElement("div", { "data-testid": "mobile-header-right" }, right),
    children({ onMobileHeaderLeft: setLeft, onMobileHeaderRight: setRight }),
  );
}

test("RunDetail mobile menu executes each enabled lifecycle action and closes after selection", async () => {
  const run = makeRun({
    id: "mobile-run",
    actions: { canStop: true, canInvestigate: true, canApplyInvestigation: true, canRetry: true, canResumeFromFailure: true, canReview: true, canDelete: true },
  });
  const stop = vi.fn(async () => undefined);
  const investigate = vi.fn();
  const apply = vi.fn();
  const retry = vi.fn(async (value) => value);
  const resume = vi.fn();
  const remove = vi.fn();
  renderWithProviders(createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false,
    task: null, taskTitle: "Mobile task", profileName: "Reliability",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined),
    onRetry: retry, onResumeFromFailure: resume, onInvestigate: investigate,
    onApplyInvestigation: apply, onStop: stop, onDelete: remove,
    onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));

  fireEvent.click(screen.getByTitle("Stop run"));
  await waitFor(() => assert.equal(stop.mock.calls.length, 1));
  const choose = (name: string) => {
    fireEvent.click(screen.getByRole("button", { name: "Run actions" }));
    fireEvent.click(screen.getByRole("button", { name }));
    assert.equal(screen.queryByRole("button", { name: "Apply Fixes" }), null);
  };
  choose("Investigate"); choose("Apply Fixes"); choose("Re-run"); choose("Resume"); choose("Review"); choose("Delete");

  assert.deepEqual(investigate.mock.calls, [["mobile-run"]]);
  assert.deepEqual(apply.mock.calls, [["mobile-run"]]);
  assert.equal(retry.mock.calls[0]?.[0], run); assert.equal(resume.mock.calls[0]?.[0], run); assert.equal(remove.mock.calls[0]?.[0], run);
  await waitFor(() => assert.ok(screen.getByRole("heading", { name: "Review Changes" })));
});

test("RunDetail mobile menu closes when an operator clicks outside it", () => {
  const run = makeRun({ actions: { canInvestigate: true } });
  renderWithProviders(createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false,
    task: null, taskTitle: "Mobile task", profileName: "Reliability",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined),
    onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(),
    onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(),
    onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Run actions" }));
  assert.ok(screen.getByRole("button", { name: "Investigate" }));
  fireEvent.mouseDown(document.body);
  assert.equal(screen.queryByRole("button", { name: "Investigate" }), null);
});

test("RunDetail supplies mobile header controls that open and dismiss its run-details dialog", async () => {
  const run = makeRun({ id: "header-run", actions: { canStop: true } });
  const handlers = {
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined),
    onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(),
    onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(),
    onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined),
  };
  const renderDetail = ({ onMobileHeaderLeft, onMobileHeaderRight }: { onMobileHeaderLeft: (node: ReactNode | null) => void; onMobileHeaderRight: (node: ReactNode | null) => void }) => createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false,
    task: null, taskTitle: "Mobile task", profileName: "Reliability",
    ...handlers, deleteLoading: false, onMobileHeaderLeft, onMobileHeaderRight,
  });
  renderWithProviders(createElement(MobileHeaderHost, {
    children: renderDetail,
  }));

  await waitFor(() => assert.ok(screen.getByTestId("mobile-header-left").querySelector("button")));
  fireEvent.click(screen.getByTitle("Run details"));
  assert.ok(screen.getByRole("heading", { name: "Run Details" }));
  fireEvent.keyDown(document, { key: "Escape" });
  await waitFor(() => assert.equal(screen.queryByRole("heading", { name: "Run Details" }), null));
});

test("RunDetail mobile details lets an operator copy the full identifier and dismiss via backdrop", async () => {
  const clipboard = { writeText: vi.fn(async () => undefined) };
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: clipboard });
  const run = makeRun({ id: "mobile-full-run-id" });
  const handlers = {
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined),
    onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(),
    onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(),
    onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined),
  };
  const renderDetail = ({ onMobileHeaderLeft, onMobileHeaderRight }: { onMobileHeaderLeft: (node: ReactNode | null) => void; onMobileHeaderRight: (node: ReactNode | null) => void }) => createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false,
    task: null, taskTitle: "Mobile task", profileName: "Reliability",
    ...handlers, deleteLoading: false, onMobileHeaderLeft, onMobileHeaderRight,
  });
  renderWithProviders(createElement(MobileHeaderHost, { children: renderDetail }));

  await waitFor(() => assert.ok(screen.getByTitle("Run details")));
  fireEvent.click(screen.getByTitle("Run details"));
  await waitFor(() => assert.ok(screen.getByRole("heading", { name: "Run Details" })));
  fireEvent.click(screen.getAllByTitle("Copy run ID: mobile-full-run-id")[1]!);
  await waitFor(() => assert.deepEqual(clipboard.writeText.mock.calls, [["mobile-full-run-id"]]));
  fireEvent.click(document.querySelector(".fixed.inset-0.bg-black\\/40")!);
  await waitFor(() => assert.equal(screen.queryByRole("heading", { name: "Run Details" }), null));
});
