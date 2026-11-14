import { create } from "@bufbuild/protobuf";
import { fireEvent, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { MutationPreviewSchema } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import { CommitAuthorizationDialog, type PendingCommitAuthorization } from "./CommitAuthorizationDialog";
import { renderWithProviders } from "../test-utils/renderWithProviders";

const pending: PendingCommitAuthorization = {
  message: "Keep the mobile approval flow",
  options: { conventional: false, amend: false },
  preview: create(MutationPreviewSchema, {
    repositoryId: "repo-1",
    repositoryPath: "/workspace/repo",
    operation: "repo.commit",
    branch: "main",
    expectedRevision: "revision-1",
    subjectDigest: "digest-1",
    stagedFiles: ["src/App.tsx"],
    fileCount: 1,
  }),
};

test("renders commit authorization as the shared narrow-viewport sheet", () => {
  const onClose = vi.fn();
  const onConfirm = vi.fn();
  renderWithProviders(
    <CommitAuthorizationDialog
      pending={pending}
      isAuthorizing={false}
      skipPrecommit={false}
      dontAskAgain={false}
      showPrecommitOption
      onSkipPrecommitChange={vi.fn()}
      onDontAskAgainChange={vi.fn()}
      onConfirm={onConfirm}
      onClose={onClose}
    />,
  );

  const dialog = screen.getByTestId("commit-authorization-dialog");
  expect(dialog).toHaveAttribute("role", "dialog");
  expect(dialog.parentElement).toHaveAttribute("data-presentation", "sheet");
  expect(screen.getByText("Keep the mobile approval flow")).toBeVisible();

  fireEvent.click(screen.getByTestId("confirm-commit-authorization"));
  expect(onConfirm).toHaveBeenCalledOnce();
  fireEvent.click(screen.getByTestId("commit-authorization-dialog.grabber"));
  expect(onClose).toHaveBeenCalledOnce();
});
