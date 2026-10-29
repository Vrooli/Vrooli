import assert from "node:assert/strict";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";
import { DataState } from "./DataState";
import { ErrorNotice } from "./ErrorNotice";
import { useConfirmAction, type ConfirmRequest } from "./useConfirmAction";
import { answerConfirmDialog, getErrorNotice, waitForConfirmDialogClosed } from "../../test-utils/feedback";

test("ErrorNotice explains the cause, offers retry only when it can help, and keeps the raw detail", async () => {
  const user = userEvent.setup();
  const retry = vi.fn();
  const { rerender } = render(<ErrorNotice error="[unauthenticated] HTTP 401" subject="runs" onRetry={retry} />);
  const full = getErrorNotice("Sign-in required", "[unauthenticated] HTTP 401");
  assert.equal(full.dataset.errorKind, "auth");
  assert.ok(screen.getByText(/runs unavailable/));
  await user.click(screen.getByRole("button", { name: /try again/i }));
  assert.equal(retry.mock.calls.length, 1);

  rerender(<ErrorNotice compact error="Request failed: 503" subject="Runs" onRetry={retry} />);
  const compact = getErrorNotice("Runs: Agent Manager is unreachable", "Request failed: 503");
  await user.click(compact.querySelector("button") as HTMLButtonElement);
  assert.equal(retry.mock.calls.length, 2);

  rerender(<ErrorNotice compact error="[permission_denied] nope" onRetry={retry} />);
  assert.equal(getErrorNotice("Not permitted").querySelector("button"), null, "retry is not offered when it cannot help");
});

test("DataState shows empty only after a successful load, and keeps stale rows beside a refresh failure", () => {
  const props = { subject: "Runs", empty: <p>No runs yet</p>, children: <p>row</p> };
  const { rerender } = render(<DataState {...props} loading isEmpty />);
  assert.ok(screen.getByText(/Loading runs/));
  assert.equal(screen.queryByText("No runs yet"), null);

  rerender(<DataState {...props} loading={false} isEmpty error="Request failed: 503" />);
  getErrorNotice("unreachable");
  assert.equal(screen.queryByText("No runs yet"), null);

  rerender(<DataState {...props} loading={false} isEmpty compact error="Request failed: 503" />);
  getErrorNotice("Runs: Agent Manager is unreachable");

  rerender(<DataState {...props} loading compact isEmpty />);
  assert.ok(screen.getByRole("status").textContent?.includes("Loading runs"));

  rerender(<DataState {...props} loading={false} isEmpty={false} error="Request failed: 503" />);
  assert.ok(screen.getByText("row"));
  getErrorNotice("Runs: Agent Manager is unreachable");

  rerender(<DataState {...props} loading={false} isEmpty />);
  assert.ok(screen.getByText("No runs yet"));
});

function ConfirmHarness({ request }: { request: ConfirmRequest }) {
  const { confirm, dialog } = useConfirmAction("harness");
  return (
    <>
      <button type="button" onClick={() => confirm(request)}>Delete thing</button>
      {dialog}
    </>
  );
}

test("useConfirmAction runs the action only on confirmation and keeps a failure in the dialog", async () => {
  const user = userEvent.setup();
  const action = vi.fn().mockRejectedValueOnce(new Error("Request failed: 503")).mockResolvedValueOnce(undefined);
  render(<ConfirmHarness request={{ title: "Delete thing?", description: "Gone for good.", confirmLabel: "Delete thing", subject: "The thing", action }} />);

  await user.click(screen.getByText("Delete thing", { selector: "button" }));
  await answerConfirmDialog(user, "cancel");
  assert.equal(action.mock.calls.length, 0);
  await waitForConfirmDialogClosed();

  await user.click(screen.getByText("Delete thing", { selector: "button" }));
  assert.ok(screen.getByText("The thing"));
  const dialog = await answerConfirmDialog(user, "confirm");
  await waitFor(() => assert.match(dialog.textContent ?? "", /API is not responding/));
  assert.ok(dialog.isConnected, "a failed action keeps the dialog open");

  await answerConfirmDialog(user, "confirm");
  await waitForConfirmDialogClosed();
  assert.equal(action.mock.calls.length, 2);
});
