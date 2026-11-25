import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import report from "./fixtures/agent-buyer-range-notes.identity.json";
import { PageIdentityTool, parsePageIdentityReport } from "./PageIdentityTool";

describe("PageIdentityTool", () => {
  it("turns the real Agent Buyer Range report into an owner and evidence view", async () => {
    render(<PageIdentityTool onClose={vi.fn()} />);
    const file = new File([JSON.stringify(report)], "agent-buyer-range-notes.identity.json", { type: "application/json" });
    Object.defineProperty(file, "text", { value: async () => JSON.stringify(report) });
    fireEvent.change(screen.getByLabelText(/Identity report/), { target: { files: [file] } });

    expect(await screen.findByText("agent buyer range · collection page")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "/notes" })).toBeInTheDocument();
    expect(screen.getByText("AttachmentUpload")).toBeInTheDocument();
    expect(screen.getAllByText("Owned by NotesCard").length).toBeGreaterThan(0);
    expect(screen.getByText(/Source\/build evidence only/)).toBeInTheDocument();
    expect(screen.getByText(/features\/notes\/AttachmentUpload\.tsx/)).toBeInTheDocument();
    expect(screen.getAllByText(/page-component-identity-cli/).length).toBeGreaterThan(0);
  });

  it("rejects a report outside the supported identity schema", () => {
    expect(() => parsePageIdentityReport({ schema: "other" })).toThrow(/page identity report/);
  });

  it("keeps report reading local and explains invalid input", async () => {
    render(<PageIdentityTool onClose={vi.fn()} />);
    const file = new File(["{}"], "wrong.json", { type: "application/json" });
    Object.defineProperty(file, "text", { value: async () => "{}" });
    fireEvent.change(screen.getByLabelText(/Identity report/), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent(/page identity report/));
    expect(screen.getByText(/not sent to a service/)).toBeInTheDocument();
  });
});
