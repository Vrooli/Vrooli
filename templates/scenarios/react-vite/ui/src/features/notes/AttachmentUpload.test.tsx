import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
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
});
