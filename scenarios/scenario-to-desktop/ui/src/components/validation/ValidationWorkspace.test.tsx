import { describe, expect, it, beforeEach, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { render, screen, waitFor } from "@/test-utils";
import { ValidationWorkspace } from "./ValidationWorkspace";
import { selectors } from "../../consts/selectors";

describe("ValidationWorkspace", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("shows available and unavailable target states with reasons", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          targets: [
            {
              kind: "local",
              os: "linux",
              architecture: "amd64",
              mode: "native",
              descriptor: {
                target_id: "local-linux-amd64",
                display_name: "Local host",
                available: true,
                capabilities: [1, 6],
                reason: "local target is ready",
              },
            },
            {
              kind: "bridge",
              os: "darwin",
              architecture: "arm64",
              mode: "remote",
              descriptor: {
                target_id: "bridge-mac",
                display_name: "Office Mac",
                available: false,
                capabilities: [],
                reason: "node is offline",
              },
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    render(<ValidationWorkspace />);

    await waitFor(() =>
      expect(screen.getByText("Local host")).toBeInTheDocument(),
    );
    expect(screen.getByText("available")).toBeInTheDocument();
    expect(screen.getAllByText("unavailable").length).toBeGreaterThan(0);
    expect(screen.getByText("node is offline")).toBeInTheDocument();
  });

  it("explains an empty target inventory", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ targets: [] }), { status: 200 }),
    );

    render(<ValidationWorkspace />);

    await waitFor(() =>
      expect(
        screen.getByText("No targets are currently registered."),
      ).toBeInTheDocument(),
    );
    expect(screen.getByText("0 discovered")).toBeInTheDocument();
  });

  it("shows a useful error when inventory discovery fails", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(
      new Error("connection refused"),
    );

    render(<ValidationWorkspace />);

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("connection refused"),
    );
  });

  it("uses the generic message for an unknown inventory failure", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue("offline");

    render(<ValidationWorkspace />);

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "target inventory unavailable",
      ),
    );
  });

  it("reports an unsuccessful inventory response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("unavailable", { status: 503 }),
    );

    render(<ValidationWorkspace />);

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "target inventory failed (503)",
      ),
    );
  });

  it("announces target discovery while the inventory is loading", async () => {
    let resolveInventory: (response: Response) => void = () => undefined;
    const pendingInventory = new Promise<Response>((resolve) => {
      resolveInventory = resolve;
    });
    vi.spyOn(globalThis, "fetch").mockReturnValue(pendingInventory);

    render(<ValidationWorkspace />);

    expect(
      screen.getByTestId(selectors.validation.inventoryLoading),
    ).toHaveTextContent("Discovering validation targets");
    resolveInventory(
      new Response(JSON.stringify({ targets: [] }), { status: 200 }),
    );
    await screen.findByText("No targets are currently registered.");
  });

  it("keeps incomplete target descriptors truthful", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          targets: [
            {
              kind: "remote",
              os: "linux",
              architecture: "arm64",
              mode: "bridge",
              descriptor: { target_id: "remote-linux" },
            },
            {
              kind: "unknown",
              os: "unknown",
              architecture: "unknown",
              mode: "unknown",
            },
          ],
        }),
        { status: 200 },
      ),
    );

    render(<ValidationWorkspace />);

    await waitFor(() =>
      expect(screen.getByText("remote-linux")).toBeInTheDocument(),
    );
    expect(screen.getByText("Unnamed target")).toBeInTheDocument();
    expect(screen.getAllByText("No limitation reported.")).toHaveLength(2);
  });

  it("creates, starts, waits once, and exposes typed cells, gate blockers, and evidence", async () => {
    const user = userEvent.setup();
    const calls: Array<{ url: string; method: string }> = [];
    const selection = {
      scenario_name: "demo",
      artifact_digest: "sha256:demo",
      journeys: [
        {
          journey_id: "journey-1",
          display_name: "Desktop smoke",
          required: true,
        },
      ],
      targets: [
        {
          kind: "local",
          descriptor: {
            target_id: "local-linux",
            display_name: "Local Linux",
            available: true,
          },
        },
      ],
      environment_profiles: [1],
    };
    const baseRun = { run_id: "run-1", selection, cells: [] };
    const completedRun = {
      ...baseRun,
      state: "completed",
      cells: [
        {
          state: "completed",
          cell: {
            cell_id: "cell-missing",
            journey_id: "journey-1",
            target_id: "local-linux",
            environment_profile: 1,
            disposition: 7,
            reason: "required cell was not run",
            required: true,
            applicable: true,
            evidence: [],
          },
        },
        {
          state: "completed",
          cell: {
            cell_id: "cell-failed",
            journey_id: "journey-1",
            target_id: "local-linux",
            environment_profile: 1,
            disposition: 2,
            reason: "desktop assertion failed",
            required: true,
            applicable: true,
            evidence: [
              {
                evidence_id: "evidence-1",
                kind: 2,
                uri: "file:///tmp/desktop.webm",
                sha256: "sha256:evidence",
                media_type: "video/webm",
                redacted: true,
              },
            ],
          },
        },
      ],
      gate: {
        passed: false,
        disposition: 2,
        required_cell_count: 2,
        passing_cell_count: 0,
        missing_cell_ids: ["cell-missing"],
        failed_cell_ids: ["cell-failed"],
        reason: "required cells are missing or failing",
      },
    };
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      const method = init?.method ?? "GET";
      calls.push({ url, method });
      if (url.endsWith("/targets"))
        return new Response(
          JSON.stringify({
            targets: [
              {
                kind: "local",
                os: "linux",
                architecture: "amd64",
                mode: "native",
                descriptor: {
                  target_id: "local-linux",
                  display_name: "Local Linux",
                  available: true,
                  capabilities: [],
                },
              },
            ],
          }),
          { status: 200 },
        );
      if (url.endsWith("/matrices") && method === "POST")
        return new Response(JSON.stringify(baseRun), { status: 201 });
      if (url.endsWith("/start"))
        return new Response(JSON.stringify({ ...baseRun, state: "running" }), {
          status: 202,
        });
      if (url.endsWith("/wait"))
        return new Response(JSON.stringify(completedRun), { status: 200 });
      return new Response(JSON.stringify(completedRun), { status: 200 });
    });

    render(<ValidationWorkspace />);
    await screen.findByText("Local Linux");
    await user.type(
      screen.getByTestId(selectors.validation.artifactDigest),
      "sha256:demo",
    );
    await user.click(screen.getByTestId(selectors.validation.createMatrix));
    await screen.findByText("Matrix review");
    expect(
      calls.some(
        (call) => call.method === "POST" && call.url.endsWith("/matrices"),
      ),
    ).toBe(true);

    await user.click(screen.getByTestId(selectors.validation.startRun));
    await screen.findByText("running");
    await user.click(screen.getByTestId(selectors.validation.waitRun));
    await waitFor(() => {
      expect(screen.getAllByText("completed").length).toBeGreaterThan(0);
    });
    expect(
      screen.getByText("required cells are missing or failing"),
    ).toBeInTheDocument();
    expect(screen.getByText("not run")).toBeInTheDocument();
    expect(screen.getAllByText("failed").length).toBeGreaterThan(0);

    await user.click(
      screen.getByRole("button", { name: "Inspect evidence for cell-failed" }),
    );
    expect(screen.getByText("file:///tmp/desktop.webm")).toBeInTheDocument();
    expect(screen.getByText(/Desktop runtime: present/)).toBeInTheDocument();
    expect(screen.getByText(/BAS workflow: not reported/)).toBeInTheDocument();
    expect(screen.getByText("redacted")).toBeInTheDocument();
    expect(calls.some((call) => call.url.endsWith("/wait"))).toBe(true);
  });

  it("discovers provider-owned cases without asking the UI to parse workflow files", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      if (url.endsWith("/targets")) {
        return new Response(
          JSON.stringify({
            targets: [
              {
                kind: "local",
                os: "linux",
                architecture: "amd64",
                mode: "native",
                descriptor: {
                  target_id: "local-linux",
                  display_name: "Local Linux",
                  available: true,
                },
              },
            ],
          }),
          { status: 200 },
        );
      }
      if (url.includes("/validation/catalog")) {
        return new Response(
          JSON.stringify({
            journeys: [
              {
                journey_id: "demo/login",
                display_name: "Login existing case",
                source_path: "bas/cases/login.json",
                execution_mode: "observer",
                required: false,
                category: "existing-bas-case",
                requirements: ["auth"],
                estimated_duration_seconds: 12,
                safety: { mutating: false },
              },
            ],
          }),
          { status: 200 },
        );
      }
      return new Response(
        JSON.stringify({ error: "unexpected validation request" }),
        { status: 500 },
      );
    });

    render(<ValidationWorkspace />);
    await screen.findByText("Local Linux");
    await user.click(screen.getByTestId(selectors.validation.discoverCatalog));
    await screen.findByText("Login existing case");
    expect(
      screen.getByRole("heading", { name: "existing-bas-case" }),
    ).toBeInTheDocument();
    expect(screen.getByText("observer · ~12s")).toBeInTheDocument();
    expect(
      screen.getByText("source: bas/cases/login.json"),
    ).toBeInTheDocument();
    expect(screen.getByText("requirements: auth")).toBeInTheDocument();
  });

  it("supports reattach, abort, and rerun-failed without hiding the server error", async () => {
    const user = userEvent.setup();
    let abortCount = 0;
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      const method = init?.method ?? "GET";
      if (url.endsWith("/targets"))
        return new Response(
          JSON.stringify({
            targets: [
              {
                kind: "local",
                os: "linux",
                architecture: "amd64",
                mode: "native",
                descriptor: {
                  target_id: "local-linux",
                  display_name: "Local Linux",
                  available: true,
                },
              },
            ],
          }),
          { status: 200 },
        );
      if (url.endsWith("/matrices/run-reattach") && method === "GET")
        return new Response(
          JSON.stringify({
            run_id: "run-reattach",
            state: "queued",
            cells: [],
            selection: {},
          }),
          { status: 200 },
        );
      if (url.endsWith("/abort")) {
        abortCount += 1;
        return new Response(
          JSON.stringify({
            run_id: "run-reattach",
            state: "cancelled",
            cells: [],
            selection: {},
          }),
          { status: 200 },
        );
      }
      if (url.endsWith("/rerun"))
        return new Response(
          JSON.stringify({
            run_id: "run-rerun",
            state: "queued",
            cells: [],
            selection: {},
            parent_run_id: "run-reattach",
          }),
          { status: 201 },
        );
      return new Response(
        JSON.stringify({ error: "unexpected validation request" }),
        { status: 500 },
      );
    });

    render(<ValidationWorkspace />);
    await screen.findByText("Local Linux");
    await user.type(screen.getByLabelText("Matrix run ID"), "run-reattach");
    await user.click(screen.getByTestId(selectors.validation.reattachRun));
    await screen.findByText("queued");
    await user.click(screen.getByTestId(selectors.validation.abortRun));
    await screen.findByText("cancelled");
    expect(abortCount).toBe(1);
    await user.click(screen.getByTestId(selectors.validation.rerunFailed));
    await screen.findByText("run-rerun");
  });

  it("keeps an empty matrix visible and explicitly blocked", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      if (url.endsWith("/targets")) {
        return new Response(
          JSON.stringify({
            targets: [
              {
                kind: "local",
                descriptor: {
                  target_id: "local-linux",
                  display_name: "Local Linux",
                  available: true,
                },
              },
            ],
          }),
          { status: 200 },
        );
      }
      return new Response(
        JSON.stringify({
          run_id: "run-empty",
          state: "completed",
          selection: {},
          cells: [],
          gate: { passed: false, reason: "required coverage is omitted" },
        }),
        { status: 200 },
      );
    });

    render(<ValidationWorkspace />);
    await screen.findByText("Local Linux");
    await user.type(screen.getByLabelText("Matrix run ID"), "run-empty");
    await user.click(screen.getByTestId(selectors.validation.reattachRun));

    await screen.findByText(
      "No cells were returned. Required coverage is omitted and the release gate cannot pass.",
    );
    expect(
      screen.getByTestId(selectors.validation.gateSummary),
    ).toHaveTextContent("required coverage is omitted");
  });

  it("renders a complete matrix as a passing gate", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      if (url.endsWith("/targets")) {
        return new Response(
          JSON.stringify({
            targets: [
              {
                kind: "local",
                descriptor: {
                  target_id: "local-linux",
                  display_name: "Local Linux",
                  available: true,
                },
              },
            ],
          }),
          { status: 200 },
        );
      }
      return new Response(
        JSON.stringify({
          run_id: "run-complete",
          state: "completed",
          selection: {
            scenario_name: "demo",
            journeys: [
              {
                journey_id: "journey-1",
                display_name: "Desktop smoke",
                required: true,
              },
            ],
            targets: [
              {
                kind: "local",
                descriptor: {
                  target_id: "local-linux",
                  display_name: "Local Linux",
                },
              },
            ],
          },
          cells: [
            {
              state: "completed",
              cell: {
                cell_id: "cell-pass",
                journey_id: "journey-1",
                target_id: "local-linux",
                environment_profile: 1,
                disposition: 1,
                reason: "all required evidence present",
                required: true,
                evidence: [
                  {
                    evidence_id: "desktop",
                    kind: 2,
                    uri: "file:///desktop.json",
                    sha256: "sha256:desktop",
                    redacted: true,
                  },
                ],
              },
            },
          ],
          gate: {
            passed: true,
            disposition: 1,
            required_cell_count: 1,
            passing_cell_count: 1,
            reason: "all required cells passed",
          },
        }),
        { status: 200 },
      );
    });

    render(<ValidationWorkspace />);
    await screen.findByText("Local Linux");
    await user.type(screen.getByLabelText("Matrix run ID"), "run-complete");
    await user.click(screen.getByTestId(selectors.validation.reattachRun));

    expect(
      screen.getByTestId(selectors.validation.gateSummary),
    ).toHaveTextContent("pass");
    expect(screen.getByText("all required cells passed")).toBeInTheDocument();
  });

  it("keeps unavailable cells non-passing with an operator reason", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      if (url.endsWith("/targets")) {
        return new Response(
          JSON.stringify({
            targets: [
              {
                kind: "bridge",
                descriptor: {
                  target_id: "bridge-mac",
                  display_name: "Office Mac",
                  available: false,
                },
              },
            ],
          }),
          { status: 200 },
        );
      }
      return new Response(
        JSON.stringify({
          run_id: "run-unavailable",
          state: "completed",
          selection: {
            journeys: [
              {
                journey_id: "journey-1",
                display_name: "Desktop smoke",
                required: true,
              },
            ],
            targets: [
              {
                kind: "bridge",
                descriptor: {
                  target_id: "bridge-mac",
                  display_name: "Office Mac",
                },
              },
            ],
          },
          cells: [
            {
              state: "completed",
              cell: {
                cell_id: "cell-unavailable",
                journey_id: "journey-1",
                target_id: "bridge-mac",
                environment_profile: 1,
                disposition: 4,
                reason: "bridge node is offline",
                required: true,
                evidence: [],
              },
            },
          ],
          gate: {
            passed: false,
            disposition: 4,
            required_cell_count: 1,
            passing_cell_count: 0,
            failed_cell_ids: ["cell-unavailable"],
            reason: "required target is unavailable",
          },
        }),
        { status: 200 },
      );
    });

    render(<ValidationWorkspace />);
    await screen.findByText("Office Mac");
    await user.type(screen.getByLabelText("Matrix run ID"), "run-unavailable");
    await user.click(screen.getByTestId(selectors.validation.reattachRun));

    expect(screen.getAllByText("unavailable").length).toBeGreaterThan(0);
    expect(screen.getByText("bridge node is offline")).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.validation.gateSummary),
    ).toHaveTextContent("required target is unavailable");
  });

  it("reruns selected cells and compares a prior matrix", async () => {
    const user = userEvent.setup();
    const calls: Array<{ url: string; method: string; body?: string }> = [];
    const run = {
      run_id: "run-actions",
      state: "completed",
      selection: {
        scenario_name: "demo",
        artifact_digest: "sha256:demo",
        journeys: [
          {
            journey_id: "journey-1",
            display_name: "Desktop smoke",
            required: true,
          },
        ],
        targets: [
          {
            kind: "local",
            descriptor: {
              target_id: "local-linux",
              display_name: "Local Linux",
              available: true,
            },
          },
        ],
        environment_profiles: [1],
      },
      cells: [
        {
          state: "completed",
          cell: {
            cell_id: "cell-1",
            journey_id: "journey-1",
            target_id: "local-linux",
            environment_profile: 1,
            disposition: 1,
            required: true,
            evidence: [],
          },
        },
      ],
      gate: {
        passed: true,
        disposition: 1,
        required_cell_count: 1,
        passing_cell_count: 1,
      },
    };
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url =
        typeof input === "string"
          ? input
          : input instanceof URL
            ? input.toString()
            : input.url;
      const method = init?.method ?? "GET";
      calls.push({
        url,
        method,
        body: typeof init?.body === "string" ? init.body : undefined,
      });
      if (url.endsWith("/targets"))
        return new Response(
          JSON.stringify({
            targets: [
              {
                kind: "local",
                os: "linux",
                architecture: "amd64",
                mode: "native",
                descriptor: {
                  target_id: "local-linux",
                  display_name: "Local Linux",
                  available: true,
                },
              },
            ],
          }),
          { status: 200 },
        );
      if (url.endsWith("/matrices/run-actions") && method === "GET")
        return new Response(JSON.stringify(run), { status: 200 });
      if (url.endsWith("/matrices/run-actions/rerun"))
        return new Response(
          JSON.stringify({
            ...run,
            state: "queued",
            parent_run_id: "run-actions",
          }),
          { status: 201 },
        );
      if (url.endsWith("/compare/prior-run"))
        return new Response(
          JSON.stringify({ changed: true, cells: [{ changed: true }] }),
          { status: 200 },
        );
      return new Response(
        JSON.stringify({ error: "unexpected validation request" }),
        { status: 500 },
      );
    });

    render(<ValidationWorkspace />);
    await screen.findByText("Local Linux");
    await user.type(screen.getByLabelText("Matrix run ID"), "run-actions");
    await user.click(screen.getByTestId(selectors.validation.reattachRun));
    await screen.findByText("Matrix review");
    await user.click(
      screen.getByRole("checkbox", { name: "Select cell cell-1 for rerun" }),
    );
    await user.click(screen.getByTestId(selectors.validation.rerunSelected));
    await waitFor(() => {
      expect(
        calls.some(
          (call) =>
            call.url.endsWith("/rerun") &&
            call.body?.includes('"cell_id":"cell-1"'),
        ),
      ).toBe(true);
    });
    await user.type(
      screen.getByTestId(selectors.validation.comparePriorRun),
      "prior-run",
    );
    await user.click(screen.getByTestId(selectors.validation.compareRun));
    await screen.findByText(/Comparison: changed/);
  });
});
