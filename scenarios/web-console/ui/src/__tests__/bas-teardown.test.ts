import { runInNewContext } from "node:vm";
import { afterEach, describe, expect, it, vi } from "vitest";
import action from "../../../bas/actions/teardown-test-sessions.json";

const expression = action.flow_definition.nodes[0]?.action.evaluate.expression;
if (!expression) throw new Error("BAS teardown action has no executable expression");
const execute = () => runInNewContext(expression, { document, sessionStorage, localStorage, fetch: globalThis.fetch, Date, setTimeout }) as Promise<void>;
const listing = (sessions: object[]) => ({ ok: true, json: () => Promise.resolve({ sessions }) });

function deletionID(init: RequestInit): string {
  if (typeof init.body !== "string") throw new Error("Delete request body must be JSON text");
  const body: unknown = JSON.parse(init.body);
  if (!body || typeof body !== "object" || !("id" in body) || typeof body.id !== "string") throw new Error("Delete request has no session identity");
  return body.id;
}

afterEach(() => {
  vi.unstubAllGlobals();
  sessionStorage.clear();
  localStorage.clear();
  document.body.removeAttribute("data-bas-teardown-clean");
  document.body.removeAttribute("data-bas-teardown-leaked");
});

describe("BAS terminal teardown ownership", () => {
  it("deletes only lease-owned sessions even when an operator starts a session after baseline", async () => {
    sessionStorage.setItem("__bas_baseline_sessions", JSON.stringify(["existing-operator"]));
    const existing = { id: "existing-operator", owner: "human" };
    const concurrent = { id: "concurrent-operator", owner: "human" };
    const mine = { id: "test-owned", owner: "vrooli-test-lease" };
    const deleted: string[] = [];
    let lists = 0;
    vi.stubGlobal("fetch", vi.fn((url: string, init: RequestInit) => {
      if (url.endsWith("/Delete")) {
        deleted.push(deletionID(init));
        return Promise.resolve({ ok: true });
      }
      return Promise.resolve(listing(lists++ === 0 ? [existing, mine, concurrent] : deleted.includes(concurrent.id) ? [existing] : [existing, concurrent]));
    }));
    await execute();
    expect(deleted).toEqual(["test-owned"]);
    expect(document.body.getAttribute("data-bas-teardown-leaked")).toBe("0");
  });

  it("refuses cleanup without a recorded baseline", async () => {
    const request = vi.fn();
    vi.stubGlobal("fetch", request);
    await expect(execute()).rejects.toThrow(/baseline/i);
    expect(request).not.toHaveBeenCalled();
  });

  it("does not report a clean teardown when the final ownership inventory is unavailable", async () => {
    sessionStorage.setItem("__bas_baseline_sessions", "[]");
    let lists = 0;
    vi.stubGlobal("fetch", vi.fn((url: string) => {
      if (url.endsWith("/Delete")) return Promise.resolve({ ok: true });
      return Promise.resolve(lists++ === 0 ? listing([{ id: "test-owned", owner: "vrooli-test-lease" }]) : { ok: false, status: 503 });
    }));
    await expect(execute()).rejects.toThrow(/list|inventory/i);
    expect(document.body.getAttribute("data-bas-teardown-clean")).not.toBe("true");
    expect(document.body.getAttribute("data-bas-teardown-leaked")).not.toBe("0");
  });

  it("refuses to delete a captured session whose lease ownership is not established", async () => {
    sessionStorage.setItem("__bas_baseline_sessions", "[]");
    localStorage.setItem("bas.terminalSmoke.sessionId", "operator-session");
    const deleted: string[] = [];
    vi.stubGlobal("fetch", vi.fn((url: string, init: RequestInit) => {
      if (url.endsWith("/Delete")) { deleted.push(deletionID(init)); return Promise.resolve({ ok: true }); }
      return Promise.resolve(listing(deleted.length ? [] : [{ id: "operator-session", owner: "human" }]));
    }));
    await expect(execute()).rejects.toThrow(/ownership|lease/i);
    expect(deleted).toEqual([]);
  });
});
