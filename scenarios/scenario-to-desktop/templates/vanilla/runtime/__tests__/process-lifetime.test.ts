import fs from "node:fs";
import vm from "node:vm";
import ts from "typescript";
import { describe, expect, it, vi } from "vitest";
import { bundledRuntimeSpawnOptions, signalBundledRuntimeGroup } from "../process-lifetime";

describe("bundled runtime process lifetime", () => {
    it("gives POSIX bundled runtime a dedicated process group", () => {
        expect(bundledRuntimeSpawnOptions("linux")).toEqual({ detached: true });
        expect(bundledRuntimeSpawnOptions("darwin")).toEqual({ detached: true });
        expect(bundledRuntimeSpawnOptions("win32")).toEqual({ detached: false });
    });

    it("signals the owned process group rather than a host launch scope", () => {
        const signalProcess = vi.fn();
        const child = { pid: 4321, kill: vi.fn() };
        signalBundledRuntimeGroup(child, "SIGTERM", "linux", signalProcess as typeof process.kill);
        expect(signalProcess).toHaveBeenCalledWith(-4321, "SIGTERM");
        expect(child.kill).not.toHaveBeenCalled();
    });

    it("falls back to the direct child if the owned process group is already gone", () => {
        const signalProcess = vi.fn(() => {
            throw new Error("process group no longer exists");
        });
        const child = { pid: 4321, kill: vi.fn() };

        signalBundledRuntimeGroup(child, "SIGKILL", "linux", signalProcess as typeof process.kill);

        expect(signalProcess).toHaveBeenCalledWith(-4321, "SIGKILL");
        expect(child.kill).toHaveBeenCalledWith("SIGKILL");
    });
});

// Execute the actual registered main-process callback without booting Electron.
// This checks event/cleanup ordering, including a second quit while cleanup waits.
it("keeps normal application quit pending until runtime cleanup finishes", async () => {
    const script = mainCallback("app.on", "before-quit");
    let finish!: () => void;
    const cleanup = new Promise<void>(resolve => { finish = resolve; });
    const quit = vi.fn();
    const context: Record<string, unknown> & { runtimeProcess: object | null } = {
        isSmokeTestDemo: false, demoRecordingEnded: false, presentationController: null,
        runtimeProcess: {}, runtimeQuitPending: false, serverProcess: null,
        app: { quit }, packagedLaunch: { complete: vi.fn() },
        shutdownRuntime: vi.fn(() => { context.runtimeProcess = null; return cleanup; }),
        shutdownLocalVrooli: vi.fn(), recordAppSessionOutcome: vi.fn(), recordTelemetry: vi.fn(),
        ipcRegistration: null, authManager: null,
    };
    const beforeQuit = vm.runInNewContext(script, context) as (event: { preventDefault(): void }) => void;
    const first = { preventDefault: vi.fn() };
    beforeQuit(first);
    expect(first.preventDefault).toHaveBeenCalledOnce();
    expect(quit).not.toHaveBeenCalled();
    const second = { preventDefault: vi.fn() };
    beforeQuit(second);
    expect(second.preventDefault).toHaveBeenCalledOnce();
    expect(context.shutdownRuntime).toHaveBeenCalledOnce();
    finish();
    await cleanup;
    await Promise.resolve();
    expect(quit).toHaveBeenCalledOnce();
    const completed = { preventDefault: vi.fn() };
    beforeQuit(completed);
    expect(completed.preventDefault).not.toHaveBeenCalled();
});

function mainCallback(owner: string, event: string): string {
    const source = ts.createSourceFile("main.ts", fs.readFileSync(new URL("../../main.ts", import.meta.url), "utf8"), ts.ScriptTarget.Latest, true);
    let callback: ts.Expression | undefined;
    const visit = (node: ts.Node): void => {
        if (ts.isCallExpression(node) && node.expression.getText(source) === owner && ts.isStringLiteral(node.arguments[0]) && node.arguments[0].text === event) callback = node.arguments[1];
        ts.forEachChild(node, visit);
    };
    visit(source);
    if (!callback) throw new Error(`Missing ${owner} ${event} callback`);
    return ts.transpileModule(`(${callback!.getText(source)})`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText;
}

it("finishes runtime shutdown before exiting on SIGTERM", async () => {
    let finish!: () => void;
    const cleanup = new Promise<void>(resolve => { finish = resolve; });
    const exit = vi.fn();
    const shutdownRuntime = vi.fn(() => cleanup);
    const handler = vm.runInNewContext(mainCallback("process.on", "SIGTERM"), {
        console: { log: vi.fn(), error: vi.fn() },
        process: { exit }, recordTelemetry: vi.fn(async () => {}), shutdownRuntime,
        smokeTestUploadURL: "", telemetryRecorder: null, telemetryUploader: null,
    }) as () => Promise<void>;
    const pending = handler();
    await new Promise<void>(resolve => setImmediate(resolve));
    expect(shutdownRuntime).toHaveBeenCalledOnce();
    expect(exit).not.toHaveBeenCalled();
    finish();
    await pending;
    expect(exit).toHaveBeenCalledWith(1);
});

function readinessBlock(): string {
    const source = ts.createSourceFile("main.ts", fs.readFileSync(new URL("../../main.ts", import.meta.url), "utf8"), ts.ScriptTarget.Latest, true);
    const launch = source.statements.find(node => ts.isFunctionDeclaration(node) && node.name?.text === "startBundledRuntime") as ts.FunctionDeclaration;
    const statements = launch.body!.statements;
    const start = statements.findIndex(node => ts.isVariableStatement(node) && node.declarationList.declarations.some(value => value.name.getText(source) === "runtimeReady"));
    const end = statements.findIndex(node => ts.isIfStatement(node) && node.expression.getText(source) === "!runtimeReady");
    if (start < 0 || end <= start) throw new Error("Missing runtime readiness boundary");
    return ts.transpileModule(`(async () => { ${statements.slice(start, end).map(node => node.getText(source)).join("\n")} return runtimeReady; })()`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText;
}

function readinessContext(criticalOnly: boolean, timeout: number, waitForReady: (remaining: number, criticalOnly: boolean) => Promise<{ ready: boolean }>, clock = { now: 0 }) {
    return { clock, context: {
        APP_CONFIG: { OPTIMIZATION_PROFILE: { sharedRuntimeReadiness: criticalOnly } },
        serverCheckTimeoutMs: timeout, Date: { now: () => clock.now },
        delay: async (ms: number) => { clock.now += ms; },
        runtimeControlClient: { waitForReady, request: async () => ({ ready: true }) },
    }};
}

it.each([false, true])("uses the canonical readiness wait with critical_only=%s", async criticalOnly => {
    const wait = vi.fn(async () => ({ ready: true }));
    const { context } = readinessContext(criticalOnly, 1000, wait);
    expect(await vm.runInNewContext(readinessBlock(), context)).toBe(true);
    expect(wait).toHaveBeenCalledWith(1000, criticalOnly);
});

it("preserves a startup window longer than the server's bounded wait", async () => {
    const clock = { now: 0 };
    const wait = vi.fn(async (remaining: number): Promise<{ ready: boolean }> => {
        if (wait.mock.calls.length === 1) { clock.now += Math.min(120000, remaining); return { ready: false }; }
        return { ready: true };
    });
    const { context } = readinessContext(true, 180000, wait, clock);
    expect(await vm.runInNewContext(readinessBlock(), context)).toBe(true);
    expect(wait).toHaveBeenCalledTimes(2);
    expect(wait.mock.calls[1][0]).toBeGreaterThan(0);
    expect(wait.mock.calls[1][0]).toBeLessThanOrEqual(60000);
    expect(clock.now).toBeLessThan(180000);
});

it("retries a transient readiness request within the original deadline", async () => {
    const wait = vi.fn(async (): Promise<{ ready: boolean }> => { if (wait.mock.calls.length === 1) throw new Error("transient connection loss"); return { ready: true }; });
    const { context, clock } = readinessContext(true, 1000, wait);
    expect(await vm.runInNewContext(readinessBlock(), context)).toBe(true);
    expect(wait).toHaveBeenCalledTimes(2);
    expect(clock.now).toBeLessThan(1000);
});

it("stops readiness waits at the configured startup deadline", async () => {
    const clock = { now: 0 };
    const wait = vi.fn(async (remaining: number) => { clock.now += remaining; return { ready: false }; });
    const { context } = readinessContext(false, 1000, wait, clock);
    expect(await vm.runInNewContext(readinessBlock(), context)).toBe(false);
    expect(wait).toHaveBeenCalledOnce();
    expect(clock.now).toBe(1000);
});


it.each(["2026-10-22T18:00:10.000Z", null])("records only the current application session's runtime waterfall (%s)", async startedAt => {
 const source = ts.createSourceFile("main.ts", fs.readFileSync(new URL("../../main.ts", import.meta.url), "utf8"), ts.ScriptTarget.Latest, true);
 const declaration = source.statements.find(node => ts.isFunctionDeclaration(node) && node.name?.text === "recordPackagedRuntimeWaterfallMarks")!;
 const script = ts.transpileModule(`(${declaration.getText(source)})`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText;
 const emit = vi.fn(async (_event: string, _service: string, _role: string, _details: { mark_name: string }) => {});
 const oldProbe = { ts: "2026-10-22T18:00:01.000Z", event: "service_health_probe", service_id: "api", details: { success: true, attempt: 2 } };
 const current = { ts: "2026-10-22T18:00:11.000Z", event: "service_start", service_id: "api" };
 const currentProbe = { ...current, event: "service_health_probe", details: { success: true, attempt: 1 } };
 const contents = [oldProbe, current, currentProbe, { ...currentProbe, ts: "invalid" }].map(value => JSON.stringify(value)).concat("malformed JSON").join("\n");
 const record = vm.runInNewContext(script, {
  isSmokeTestDemo: true, appSessionStartedAt: startedAt,
  runtimeControlClient: { request: vi.fn(async () => ({ path: "owned-runtime.jsonl" })) },
  fs: { promises: { readFile: vi.fn(async () => contents) } },
  packagedLaunch: { emit }, recordTelemetry: vi.fn(async () => {}),
 }) as () => Promise<void>;
 await record();
 expect(emit.mock.calls.map(call => call[3].mark_name)).toEqual(startedAt ? ["service_spawned:api", "service_health_success:api:attempt-1"] : []);
});
