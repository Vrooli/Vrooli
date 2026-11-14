import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { selectors } from "../consts/selectors";
import { DashboardPage } from "./DashboardPage";

describe("Extraction Workbench", () => {
  let posted: unknown;
  let scoreSaved: Record<string, unknown> | null;

  beforeEach(async () => {
    await setLocale("en");
    posted = undefined; scoreSaved = null;
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const requestUrl = String(input);
      if (requestUrl.endsWith("/levels")) return { ok: true, json: async () => ({ levels: [{ id: "table-01", family: "tables", family_name: "Tables", difficulty: 1, title: "Product directory", task: "Extract product records.", tags: ["stable IDs"], target_url: "/practice/table-01" }] }) };
      if (requestUrl.endsWith("/score") && init?.method === "POST") {
        posted = JSON.parse(String(init.body));
        scoreSaved = { tool: "scrapy-toolkit 0.3", level_id: "table-01", family: "tables", correct: 1, submitted: 1, expected: 1, precision: 1, recall: 1, f1: 1, passed: true };
        return { ok: true, json: async () => scoreSaved };
      }
      return { ok: true, json: async () => ({ scores: scoreSaved ? [scoreSaved] : [] }) };
    }));
  });

  afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

  it("scores a named tool submission and shows the persistent best result", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);

    expect(await screen.findByRole("heading", { name: "Product directory" })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.range.scoreboard)).toBeInTheDocument();
    await user.type(screen.getByTestId(selectors.range.toolInput), "scrapy-toolkit 0.3");
    fireEvent.change(screen.getByTestId(selectors.range.recordsInput), { target: { value: '[{"id":"A1","name":"Sample product"}]' } });
    await user.click(screen.getByTestId(selectors.range.submitButton));

    expect(await screen.findByText(/1\/1 correct · F1 100.0%/)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId(selectors.range.scoreList)).toHaveTextContent("scrapy-toolkit 0.3"));
    expect(posted).toEqual({ tool: "scrapy-toolkit 0.3", level_id: "table-01", records: [{ id: "A1", name: "Sample product" }] });
    expect(screen.getByTestId(selectors.range.scoreList)).toHaveTextContent("100.0%");
    expect(screen.getByTestId("experience-surface-workbench-data")).toHaveAttribute("data-experience-state", "ready");
  });

  it("announces an initial API failure and retries into a ready workbench", async () => {
    let failedLevels = false;
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path.endsWith("/levels") && !failedLevels) { failedLevels = true; throw new Error("API offline"); }
      return path.endsWith("/levels")
        ? { ok: true, json: async () => ({ levels: [{ id: "table-01", family: "tables", family_name: "Tables", difficulty: 1, title: "Product directory", task: "Extract it.", tags: [], target_url: "/practice/table-01" }] }) }
        : { ok: true, json: async () => ({ scores: [] }) };
    }));
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("API offline");
    expect(screen.getByTestId("experience-surface-workbench-data")).toHaveAttribute("data-experience-state", "error");
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByRole("heading", { name: "Product directory" })).toBeInTheDocument();
    expect(screen.getByTestId("experience-surface-workbench-data")).toHaveAttribute("data-experience-state", "ready");
  });

  it("keeps the loaded workbench visible when refresh fails and retries", async () => {
    let levelsCalls = 0;
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path.endsWith("/levels")) {
        levelsCalls += 1;
        if (levelsCalls > 1 && levelsCalls === 2) throw new Error("temporary refresh failure");
        return { ok: true, json: async () => ({ levels: [{ id: "table-01", family: "tables", family_name: "Tables", difficulty: 1, title: "Product directory", task: "Extract it.", tags: [], target_url: "/practice/table-01" }] }) };
      }
      return { ok: true, json: async () => ({ scores: [] }) };
    }));
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByRole("heading", { name: "Product directory" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Refresh scoreboard" }));
    expect(await screen.findByText(/Refresh failed\. Showing the last loaded data\./)).toBeInTheDocument();
    expect(screen.getByTestId("experience-surface-workbench-data")).toHaveAttribute("data-experience-state", "partial");
    await user.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(screen.getByTestId("experience-surface-workbench-data")).toHaveAttribute("data-experience-state", "ready"));
  });

  it("handles empty catalogs and validates tool names and JSON records", async () => {
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => String(input).endsWith("/levels")
      ? { ok: true, json: async () => ({ levels: [] }) }
      : { ok: true, json: async () => ({ scores: [] }) }));
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText("No challenges are available. Try refreshing the range.")).toBeInTheDocument();
    expect(screen.getByTestId("experience-surface-workbench-data")).toHaveAttribute("data-experience-state", "empty");
  });

  it("shows actionable validation and submission errors while preserving entered results", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByRole("heading", { name: "Product directory" })).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.range.submitButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("Add a tool name");
    await user.type(screen.getByTestId(selectors.range.toolInput), "my crawler");
    fireEvent.change(screen.getByTestId(selectors.range.recordsInput), { target: { value: "not-json" } });
    await user.click(screen.getByTestId(selectors.range.submitButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("valid JSON");
    fireEvent.change(screen.getByTestId(selectors.range.recordsInput), { target: { value: "[null]" } });
    await user.click(screen.getByTestId(selectors.range.submitButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("JSON array of record objects");

    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/score") && init?.method === "POST") return { ok: false, text: async () => "grader unavailable" };
      if (String(input).endsWith("/levels")) return { ok: true, json: async () => ({ levels: [{ id: "table-01", family: "tables", family_name: "Tables", difficulty: 1, title: "Product directory", task: "Extract records.", tags: [], target_url: "/practice/table-01" }] }) };
      return { ok: true, json: async () => ({ scores: [] }) };
    }));
    fireEvent.change(screen.getByTestId(selectors.range.recordsInput), { target: { value: "[]" } });
    await user.click(screen.getByTestId(selectors.range.submitButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("grader unavailable");
    expect(screen.getByTestId(selectors.range.recordsInput)).toHaveValue("[]");
  });

  it("supports narrow-screen challenge, detail, family, and scoreboard navigation", async () => {
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: query.includes("1279px"), media: query, onchange: null,
      addEventListener: () => undefined, removeEventListener: () => undefined,
      addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false,
    }));
    const levels = [
      { id: "table-01", family: "tables", family_name: "Tables", difficulty: 1, title: "Product directory", task: "Extract it.", tags: [], target_url: "/practice/table-01" },
      { id: "table-02", family: "tables", family_name: "Tables", difficulty: 2, title: "Nested directory", task: "Extract it.", tags: [], target_url: "/practice/table-02" },
      { id: "catalog-01", family: "catalogs", family_name: "Catalogs", difficulty: 1, title: "Catalog", task: "Extract it.", tags: [], target_url: "/practice/catalog-01" },
    ];
    const saved = { tool: "crawler 1.0", level_id: "table-01", family: "tables", correct: 1, submitted: 1, expected: 1, precision: 1, recall: 1, f1: 1, passed: true };
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => String(input).endsWith("/levels")
      ? { ok: true, json: async () => ({ levels }) }
      : { ok: true, json: async () => ({ scores: [saved] }) }));
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByRole("tab", { name: /Challenges/ })).toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: /Nested directory/ }));
    expect(await screen.findByRole("heading", { name: "Nested directory" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /All challenges/ }));
    await user.selectOptions(screen.getByLabelText("Filter challenges by family"), "catalogs");
    expect(screen.getByTestId(selectors.range.level({ key: "catalog-01" }))).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Filter challenges by family"), "all");
    await user.click(screen.getByRole("tab", { name: /Scoreboard/ }));
    expect(screen.getByTestId(selectors.range.scoreList)).toHaveTextContent("crawler 1.0");
    expect(screen.getByText("Pass")).toBeInTheDocument();
  });
});
