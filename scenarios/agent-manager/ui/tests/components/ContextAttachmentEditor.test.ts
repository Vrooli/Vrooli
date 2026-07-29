import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ContextAttachmentEditor } from "../../src/components/ContextAttachmentEditor.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const attachment = (type: string, extra = {}) => ({ type, key: "source", label: "Evidence", tags: ["existing"], path: "", url: "", content: "", ...extra });

test("ContextAttachmentEditor adds each type and edits, tags, and removes attachments", async () => {
  const user = userEvent.setup();
  const onChange = vi.fn();
  const { rerender, container } = renderWithProviders(createElement(ContextAttachmentEditor, { attachments: [], onChange }));
  assert.ok(screen.getByText(/No context attachments/));

  for (const type of ["File", "Link", "Note"]) await user.click(screen.getByRole("button", { name: type }));
  assert.equal(onChange.mock.calls.length, 3);
  assert.equal(onChange.mock.calls[0]![0][0].type, "file");

  rerender(createElement(ContextAttachmentEditor, { attachments: [attachment("file")], onChange }));
  assert.ok(screen.getByLabelText("File Path *"));
  fireEvent.change(screen.getByLabelText("Key (optional)"), { target: { value: "source-updated" } });
  assert.equal(onChange.mock.calls.at(-1)?.[0][0].key, "source-updated");
  await user.type(screen.getByPlaceholderText("Add tag..."), " New ");
  fireEvent.keyDown(screen.getByPlaceholderText("Add tag..."), { key: "Enter" });
  assert.deepEqual(onChange.mock.calls.at(-1)?.[0][0].tags, ["existing", "new"]);
  await user.click(screen.getByText("existing"));
  assert.deepEqual(onChange.mock.calls.at(-1)?.[0][0].tags, []);

  rerender(createElement(ContextAttachmentEditor, { attachments: [attachment("link")], onChange }));
  assert.ok(screen.getByLabelText("URL *"));
  assert.ok(screen.getByLabelText("Description"));

  rerender(createElement(ContextAttachmentEditor, { attachments: [attachment("note")], onChange }));
  assert.ok(screen.getByLabelText("Content *"));
  await user.click(container.querySelector("button.text-destructive")!);
  assert.deepEqual(onChange.mock.calls.at(-1)?.[0], []);
});

test("ContextAttachmentEditor disables mutating controls and tag creation at the ten-tag limit", () => {
  const tags = Array.from({ length: 10 }, (_, index) => `tag-${index}`);
  renderWithProviders(createElement(ContextAttachmentEditor, { attachments: [attachment("unknown", { tags })], onChange: vi.fn(), disabled: true }));
  assert.equal(screen.getByRole("button", { name: "File" }).hasAttribute("disabled"), true);
  assert.equal(screen.getByPlaceholderText("Add tag...").hasAttribute("disabled"), true);
  assert.equal(screen.queryByLabelText("File Path *"), null);
});

test("ContextAttachmentEditor normalizes tags and leaves blank or duplicate tag attempts alone", async () => {
  const user = userEvent.setup();
  const onChange = vi.fn();
  renderWithProviders(createElement(ContextAttachmentEditor, { attachments: [attachment("unknown", { tags: ["known"] })], onChange }));
  const tagInput = screen.getByPlaceholderText("Add tag...");

  await user.type(tagInput, "   ");
  await user.click(screen.getAllByRole("button").at(-1)!);
  assert.equal(onChange.mock.calls.length, 0);
  await user.clear(tagInput);
  await user.type(tagInput, "KNOWN");
  await user.keyboard("{Enter}");
  assert.equal(onChange.mock.calls.length, 0);
  await user.clear(tagInput);
  await user.type(tagInput, "Fresh");
  await user.click(screen.getAllByRole("button").at(-1)!);
  assert.deepEqual(onChange.mock.calls.at(-1)?.[0][0].tags, ["known", "fresh"]);
});
