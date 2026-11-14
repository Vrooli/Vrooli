import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../test-utils";
import { JobReadinessPage } from "./JobReadinessPage";

describe("JobReadinessPage", () => {
  afterEach(() => { cleanup(); vi.restoreAllMocks(); });

  it("loads a phone-readable opening inbox and saves a role with its source", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const path = String(input);
      if (path.includes("/openings") && init?.method === "POST") {
        return json({ opening: { id: "new-opening", title: "Staff Designer", company: "Northstar", url: "https://jobs.example/role", category: "Needs review", status: "saved" } });
      }
      if (path.includes("/openings")) return json({ openings: [{ id: "opening-1", title: "Product Designer", company: "Acme", url: "https://jobs.example/acme", category: "Design", status: "saved" }] });
      if (path.includes("/facts")) return json({ facts: [] });
      if (path.includes("/resumes")) return json({ resumes: [] });
      if (path.includes("/cover-letters")) return json({ cover_letters: [] });
      return json({});
    });
    const user = userEvent.setup();
    renderWithProviders(<JobReadinessPage />);

    expect(await screen.findByRole("heading", { name: "Move your job search forward" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Product Designer" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /view source/i })).toHaveAttribute("href", "https://jobs.example/acme");
    await user.type(screen.getByLabelText("Role title"), "Staff Designer");
    await user.type(screen.getByLabelText("Company"), "Northstar");
    await user.type(screen.getByLabelText("Source link"), "https://jobs.example/role");
    await user.click(screen.getByRole("button", { name: /save opening/i }));

    expect(await screen.findByRole("status")).toHaveTextContent("Opening saved");
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("/api/v1/jobs/openings"), expect.objectContaining({ method: "POST" })));
  });

  it("shows the evidence bank and keeps missing evidence visible for resume preparation", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const path = String(input);
      if (path.includes("/openings")) return json({ openings: [{ id: "opening-1", title: "Engineer", company: "Acme", status: "saved" }] });
      if (path.includes("/facts")) return json({ facts: [{ id: "fact-1", label: "Shipped migration", content: "Delivered a staged data migration" }] });
      if (path.includes("/resumes")) return json({ resumes: [] });
      if (path.includes("/cover-letters")) return json({ cover_letters: [] });
      return json({});
    });
    renderWithProviders(<JobReadinessPage />);
    await screen.findByRole("heading", { name: "Move your job search forward" });
    await userEvent.setup().click(screen.getByRole("button", { name: /resume versions/i }));
    expect(screen.getByRole("heading", { name: "Build a resume version" })).toBeInTheDocument();
    expect(screen.getByText("Shipped migration")).toBeInTheDocument();
    expect(screen.getByText(/No facts selected/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save version/i })).toBeDisabled();
  });

  it("categorizes an opening, reports isolated search failures, and saves reviewed role drafts", async () => {
    const store = {
      openings: [{ id: "opening-1", title: "Product Designer", company: "Acme", status: "saved", category: "Needs review", description: "Design product experiences" }],
      facts: [{ id: "fact-1", label: "Shipped redesign", content: "Built product redesign and improved activation" }],
      resumes: [] as Array<Record<string, unknown>>,
      letters: [] as Array<Record<string, unknown>>,
    };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const path = String(input);
      const body = init?.body ? JSON.parse(String(init.body)) as Record<string, unknown> : {};
      if (path.endsWith("/discover")) return json({ partial: true, results: [
        { source: "Remote OK", query: body.query, status: "ok", openings: [{ title: "Found Engineer", company: "Acme Labs", url: "https://jobs.example/found" }] },
        { source: "Indeed", query: body.query, url: "https://indeed.example/search", status: "link_out" },
        { source: "Optional API", query: body.query, status: "unavailable", error: "Provider timed out" },
      ] });
      if (path.includes("/openings/") && init?.method === "PATCH") {
        store.openings = store.openings.map((opening) => ({ ...opening, ...body }));
        return json({ opening: store.openings[0] });
      }
      if (path.endsWith("/openings") && init?.method === "POST") {
        const opening = { ...body, id: "found-opening" } as typeof store.openings[number];
        store.openings = [opening, ...store.openings];
        return json({ opening });
      }
      if (path.endsWith("/openings")) return json({ openings: store.openings });
      if (path.endsWith("/facts")) return json({ facts: store.facts });
      if (path.includes("/resumes/") && path.endsWith("/review")) {
        const resume = { ...store.resumes[0], review_findings: ["Unsupported claim: no close match in linked facts."] };
        store.resumes[0] = resume;
        return json({ resume, review: resume.review_findings });
      }
      if (path.endsWith("/resumes") && init?.method === "POST") {
        const resume = { ...body, id: "resume-1", review_findings: [] };
        store.resumes.push(resume);
        return json({ resume });
      }
      if (path.endsWith("/resumes")) return json({ resumes: store.resumes });
      if (path.endsWith("/cover-letters") && init?.method === "POST") {
        const letter = { ...body, id: "letter-1" };
        store.letters.push(letter);
        return json({ cover_letter: letter });
      }
      if (path.endsWith("/cover-letters")) return json({ cover_letters: store.letters });
      return json({});
    });

    const user = userEvent.setup();
    renderWithProviders(<JobReadinessPage />);
    await screen.findByRole("heading", { name: "Move your job search forward" });

    await user.selectOptions(screen.getByLabelText("Categorize Product Designer at Acme"), "Design");
    expect(await screen.findByRole("status")).toHaveTextContent("category updated");
    await user.click(screen.getByRole("button", { name: "Prepare for Product Designer at Acme" }));
    expect(await screen.findByRole("heading", { name: "Build a resume version" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /openings/i }));
    await user.type(screen.getByLabelText("Role, skills, or location"), "product design remote");
    await user.click(screen.getByRole("button", { name: /run searches/i }));
    expect(await screen.findByText("Provider timed out")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /open search/i })).toHaveAttribute("href", "https://indeed.example/search");
    expect(screen.getByText("Found Engineer")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save role" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Opening saved to your inbox");
    expect(screen.getByText(/via Remote OK/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /resume versions/i }));
    await user.type(screen.getByLabelText("Version name"), "Product design version");
    await user.click(screen.getByLabelText(/shipped redesign/i));
    await user.type(screen.getByLabelText("Resume draft"), "Built product redesign and improved activation.");
    await user.click(screen.getByRole("button", { name: /save version/i }));
    expect(await screen.findByText("Product design version")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /hostile review/i }));
    expect(await screen.findByText(/Unsupported claim: no close match/)).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Target opening"), "__current_resume__");
    await user.type(screen.getByLabelText("Version name"), "Current resume");
    await user.type(screen.getByLabelText("Resume draft"), "My current resume content.");
    await user.click(screen.getByRole("button", { name: /save version/i }));
    expect(await screen.findByText("Current resume baseline")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cover letters/i }));
    await user.selectOptions(screen.getByLabelText("Target opening"), "opening-1");
    await user.type(screen.getByLabelText("Draft name"), "Acme introduction");
    await user.click(screen.getByRole("button", { name: /insert guided outline/i }));
    await user.type(screen.getByLabelText("Letter draft"), " My supporting example is linked above.");
    await user.click(screen.getByLabelText(/shipped redesign/i));
    expect((screen.getByLabelText("Letter draft") as HTMLTextAreaElement).value).toContain("Dear Acme team");
    await user.click(screen.getByRole("button", { name: /save letter/i }));
    expect(await screen.findByText("Acme introduction")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("/api/v1/jobs/discover"), expect.objectContaining({ method: "POST" }));
  });

  it("recovers from an API outage and saves evidence facts without losing the workflow", async () => {
    let unavailable = true;
    let rejectOpeningWrite = true;
    let facts: Array<{ id: string; label: string; content: string }> = [];
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      if (unavailable) throw new Error("API unavailable");
      const path = String(input);
      if (path.endsWith("/openings") && init?.method === "POST") {
        if (rejectOpeningWrite) {
          rejectOpeningWrite = false;
          throw new Error("Temporary write failure");
        }
        return json({ opening: { id: "retry-opening", title: "Engineer", company: "Acme", status: "saved" } });
      }
      if (path.endsWith("/openings")) return json({ openings: [] });
      if (path.endsWith("/facts") && init?.method === "POST") {
        const fact = { ...JSON.parse(String(init.body)) as { label: string; content: string }, id: "fact-added" };
        facts = [fact];
        return json({ fact });
      }
      if (path.endsWith("/facts")) return json({ facts });
      if (path.endsWith("/resumes")) return json({ resumes: [] });
      if (path.endsWith("/cover-letters")) return json({ cover_letters: [] });
      return json({});
    });
    const user = userEvent.setup();
    renderWithProviders(<JobReadinessPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("API unavailable");
    unavailable = false;
    await user.click(screen.getByRole("button", { name: "Retry" }));
    await screen.findByRole("heading", { name: "Opening inbox" });
    await user.click(screen.getByRole("button", { name: /evidence bank/i }));
    expect(screen.getByText("Start with a few strong facts")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Short label"), "Improved activation");
    await user.type(screen.getByLabelText("Evidence and exact details"), "Shipped an onboarding redesign that increased activation.");
    await user.click(screen.getByRole("button", { name: /save fact/i }));
    expect(await screen.findByText("Improved activation")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("/api/v1/jobs/facts"), expect.objectContaining({ method: "POST" }));
    await user.click(screen.getByRole("button", { name: /openings/i }));
    await user.type(screen.getByLabelText("Role title"), "Engineer");
    await user.type(screen.getByLabelText("Company"), "Acme");
    await user.click(screen.getByRole("button", { name: /save opening/i }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Temporary write failure");
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Opening saved");
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("/api/v1/jobs/openings"), expect.objectContaining({ method: "POST" }));
  });
});

function json(data: unknown) {
  return Promise.resolve(new Response(JSON.stringify(data), { status: 200, headers: { "content-type": "application/json" } }));
}
