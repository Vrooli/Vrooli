import { afterEach, describe, expect, it, vi } from "vitest";
import { onProfilerRender } from "./profiler";

describe("profiling telemetry", () => {
  afterEach(()=>vi.unstubAllGlobals());

  it("records the commit duration against the performance clock",()=>{
    const measure=vi.fn();
    vi.stubGlobal("performance",{now:()=>100,measure});
    onProfilerRender("Storage dashboard","update",8,10,92,100);
    expect(measure).toHaveBeenCalledWith("⚛ Storage dashboard (update)",{start:92,duration:8});
  });

  it("keeps a rejected telemetry measurement from breaking the dashboard",()=>{
    vi.stubGlobal("performance",{now:()=>100,measure:()=>{throw new Error("unsupported measurement")}});
    expect(()=>onProfilerRender("Storage dashboard","mount",8,10,92,100)).not.toThrow();
  });
});
