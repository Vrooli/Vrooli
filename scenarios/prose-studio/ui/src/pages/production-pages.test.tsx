import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { VariationPage } from "./VariationPage";
import { DocumentPage } from "./DocumentPage";
import { StylesPage } from "./StylesPage";

describe("production operator pages", () => {
  afterEach(() => { cleanup(); vi.restoreAllMocks(); });

  it("generates and rerolls from the variation workspace", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(async () => new Response(JSON.stringify({ session: { id: "s" }, round: { measurement_basis: "semantic" }, candidates: [{ id: "c", text: "Measured prose", eligibility: { eligible: true } }] }), { status: 200 }));
    renderWithProviders(<VariationPage />);
    fireEvent.change(screen.getByLabelText("Subject"), { target: { value: "a shipped subject" } });
    fireEvent.click(screen.getByRole("button", { name: "Generate" }));
    await waitFor(() => expect(screen.getByText("Measured prose")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /reroll/i }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  });

  it("creates a document and displays its committed output", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () => new Response(JSON.stringify({ document: { id: "d", outline: [{ intent: "Opening", summary: "Hook", target_words: 100 }], assembled_text: "A completed document." } }), { status: 200 }));
    renderWithProviders(<DocumentPage />);
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "A title" } });
    fireEvent.click(screen.getByRole("button", { name: "Create document" }));
    await waitFor(() => expect(screen.getByText("A completed document.")).toBeInTheDocument());
    expect(screen.getByText(/Opening/)).toBeInTheDocument();
  });

  it("can assemble an existing document from the workspace", async () => {
    const responses = [
      { document: { id: "d", outline: [], assembled_text: "" } },
      { document: { id: "d", outline: [], assembled_text: "Now assembled." } },
    ];
    let index = 0;
    vi.spyOn(globalThis, "fetch").mockImplementation(async () => new Response(JSON.stringify(responses[index++] ?? responses[1]), { status: 200 }));
    renderWithProviders(<DocumentPage />);
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "A title" } });
    fireEvent.click(screen.getByRole("button", { name: "Create document" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Assemble" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Assemble" }));
    await waitFor(() => expect(screen.getByText("Now assembled.")).toBeInTheDocument());
  });

  it("renders transform schemas returned by the live registry", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () => new Response(JSON.stringify({ transforms: [{ kind: "reading_level", parameter_schema: { target_grade: "number" } }] }), { status: 200 }));
    renderWithProviders(<StylesPage />);
    await waitFor(() => expect(screen.getByText(/reading_level/)).toBeInTheDocument());
  });

  it("shows operator-facing errors and keeps unavailable actions explained", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("gateway unavailable"));
    renderWithProviders(<VariationPage />);
    fireEvent.change(screen.getByLabelText("Subject"), { target: { value: "subject" } });
    fireEvent.click(screen.getByRole("button", { name: "Generate" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("gateway unavailable"));
    cleanup();
    renderWithProviders(<StylesPage />);
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("gateway unavailable"));
  });

  it("keeps an empty generated set explicit", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () => new Response(JSON.stringify({ session: { id: "s" }, round: {}, candidates: [] }), { status: 200 }));
    renderWithProviders(<VariationPage />);
    fireEvent.change(screen.getByLabelText("Subject"), { target: { value: "subject" } });
    fireEvent.click(screen.getByRole("button", { name: "Generate" }));
    await waitFor(() => expect(screen.getByText("No candidate set yet")).toBeInTheDocument());
  });
});
