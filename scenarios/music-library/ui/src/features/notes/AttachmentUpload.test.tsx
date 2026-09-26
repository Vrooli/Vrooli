import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeAttachment } from "./mocks/factories";
import { makeNotesMocks } from "./mocks/notes";

vi.mock("../../api/notes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/notes")>();
  return { ...actual, ...makeNotesMocks() };
});

import { uploadAttachment } from "../../api/notes";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { AttachmentUpload } from "./AttachmentUpload";

describe("AttachmentUpload", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("uploads the selected file for the note", async () => {
    const user = userEvent.setup();
    const file = new File(["hello"], "hello.txt", { type: "text/plain" });

    renderWithProviders(<AttachmentUpload noteId="note-1" />);
    await user.upload(screen.getByTestId(selectors.notes.attachmentFile), file);
    await user.click(screen.getByTestId(selectors.notes.attachmentButton));

    await waitFor(() => {
      expect(uploadAttachment).toHaveBeenCalledWith("note-1", file);
    });
    expect(screen.getByTestId(selectors.notes.attachmentStatus).textContent).toContain("hello.txt");
  });

  it("keeps upload disabled until a file is selected", () => {
    renderWithProviders(<AttachmentUpload noteId="note-1" />);

    expect(screen.getByTestId(selectors.notes.attachmentButton)).toBeDisabled();
  });

  it("ignores a stale success after a new file is selected", async () => {
    const user = userEvent.setup();
    const firstUpload = deferred<Awaited<ReturnType<typeof uploadAttachment>>>();
    vi.mocked(uploadAttachment).mockImplementationOnce(() => firstUpload.promise);
    const fileA = new File(["a"], "a.txt", { type: "text/plain" });
    const fileB = new File(["b"], "b.txt", { type: "text/plain" });

    renderWithProviders(<AttachmentUpload noteId="note-1" />);
    await user.upload(screen.getByTestId(selectors.notes.attachmentFile), fileA);
    await user.click(screen.getByTestId(selectors.notes.attachmentButton));
    await user.upload(screen.getByTestId(selectors.notes.attachmentFile), fileB);
    await act(async () => {
      firstUpload.resolve(makeAttachment());
      await firstUpload.promise;
    });

    expect(screen.queryByTestId(selectors.notes.attachmentStatus)).toBeNull();
    expect(screen.getByTestId(selectors.notes.attachmentButton)).toBeEnabled();
  });

  it("ignores a stale failure after the selection is reset", async () => {
    const user = userEvent.setup();
    const failedUpload = deferred<Awaited<ReturnType<typeof uploadAttachment>>>();
    vi.mocked(uploadAttachment).mockImplementationOnce(() => failedUpload.promise);
    const file = new File(["a"], "a.txt", { type: "text/plain" });

    renderWithProviders(<AttachmentUpload noteId="note-1" />);
    const input = screen.getByTestId(selectors.notes.attachmentFile);
    await user.upload(input, file);
    await user.click(screen.getByTestId(selectors.notes.attachmentButton));
    fireEvent.change(input, { target: { files: [] } });
    await act(async () => {
      failedUpload.reject(new Error("network failed"));
      await failedUpload.promise.catch(() => undefined);
    });

    expect(screen.queryByTestId(selectors.notes.attachmentStatus)).toBeNull();
    expect(screen.getByTestId(selectors.notes.attachmentButton)).toBeDisabled();
  });

  it("shows current failures and can retry with a new attempt", async () => {
    const user = userEvent.setup();
    vi.mocked(uploadAttachment)
      .mockRejectedValueOnce(new Error("network failed"))
      .mockResolvedValueOnce(makeAttachment());
    const file = new File(["a"], "retry.txt", { type: "text/plain" });

    renderWithProviders(<AttachmentUpload noteId="note-1" />);
    await user.upload(screen.getByTestId(selectors.notes.attachmentFile), file);
    await user.click(screen.getByTestId(selectors.notes.attachmentButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.attachmentStatus).textContent).toContain("network failed");
    });

    await user.click(screen.getByTestId(selectors.notes.attachmentButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.notes.attachmentStatus).textContent).toContain("retry.txt");
    });
  });
});

function deferred<Value>() {
  let resolve!: (value: Value) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<Value>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, resolve, reject };
}
