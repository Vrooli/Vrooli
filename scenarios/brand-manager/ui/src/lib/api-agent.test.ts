import { describe, it, expect } from "vitest";
import type {
  AgentApplyRequest,
  AgentApplyResult,
  AgentValidateRequest,
  AgentValidateResult,
  LighthouseRequest,
  LighthouseResult,
} from "./api";

// Type contract tests for agent-assisted and lighthouse API types.
// [REQ:BM-REQ-AGENT-SPAWN] [REQ:BM-REQ-AGENT-INSTRUCT] [REQ:BM-REQ-AGENT-VALIDATE] [REQ:BM-REQ-LIGHTHOUSE]

describe("AgentApplyRequest type shape", () => {
  it("accepts minimal request with only scenario_name", () => {
    const req: AgentApplyRequest = { scenario_name: "test-scenario" };
    expect(req.scenario_name).toBe("test-scenario");
    expect(req.elements).toBeUndefined();
    expect(req.prompt).toBeUndefined();
  });

  it("accepts full request with elements and prompt", () => {
    const req: AgentApplyRequest = {
      scenario_name: "my-app",
      elements: ["colors", "typography"],
      prompt: "Focus on accessibility",
    };
    expect(req.elements).toHaveLength(2);
    expect(req.prompt).toBe("Focus on accessibility");
  });
});

describe("AgentApplyResult type shape", () => {
  it("carries all required fields", () => {
    const result: AgentApplyResult = {
      scenario: "test-scenario",
      brand_id: "b1",
      brand_version: 3,
      status: "pending",
      elements: ["colors", "typography", "identity", "favicon", "logo"],
      instructions: "Apply branding...",
    };
    expect(result.scenario).toBe("test-scenario");
    expect(result.brand_id).toBe("b1");
    expect(result.brand_version).toBe(3);
    expect(result.status).toBe("pending");
    expect(result.elements).toHaveLength(5);
    expect(result.instructions).toContain("Apply");
  });

  it("supports optional dry_run and agent_id", () => {
    const result: AgentApplyResult = {
      scenario: "s",
      brand_id: "b",
      brand_version: 1,
      status: "running",
      elements: [],
      instructions: "",
      dry_run: true,
      agent_id: "agent-123",
    };
    expect(result.dry_run).toBe(true);
    expect(result.agent_id).toBe("agent-123");
  });
});

describe("AgentValidateRequest type shape", () => {
  it("accepts minimal request", () => {
    const req: AgentValidateRequest = { scenario_name: "test" };
    expect(req.scenario_name).toBe("test");
    expect(req.elements).toBeUndefined();
  });

  it("accepts request with specific elements", () => {
    const req: AgentValidateRequest = {
      scenario_name: "test",
      elements: ["colors"],
    };
    expect(req.elements).toEqual(["colors"]);
  });
});

describe("AgentValidateResult type shape", () => {
  it("reports valid when no missing markers", () => {
    const result: AgentValidateResult = {
      scenario: "test",
      valid: true,
      expected: ["primary", "secondary"],
      found: ["primary", "secondary"],
      missing: [],
    };
    expect(result.valid).toBe(true);
    expect(result.missing).toHaveLength(0);
  });

  it("reports invalid with missing markers", () => {
    const result: AgentValidateResult = {
      scenario: "test",
      valid: false,
      expected: ["primary", "secondary", "accent"],
      found: ["primary"],
      missing: ["secondary", "accent"],
    };
    expect(result.valid).toBe(false);
    expect(result.missing).toContain("secondary");
    expect(result.missing).toContain("accent");
    expect(result.found).toContain("primary");
  });
});

describe("LighthouseRequest type shape", () => {
  it("accepts minimal request", () => {
    const req: LighthouseRequest = { scenario_name: "test" };
    expect(req.scenario_name).toBe("test");
    expect(req.url).toBeUndefined();
  });

  it("accepts request with custom URL", () => {
    const req: LighthouseRequest = {
      scenario_name: "test",
      url: "http://localhost:3000",
    };
    expect(req.url).toBe("http://localhost:3000");
  });
});

describe("LighthouseResult type shape", () => {
  it("carries all fields for a completed audit", () => {
    const result: LighthouseResult = {
      scenario: "test",
      brand_id: "b1",
      url: "http://localhost:3000",
      score: 95,
      passed: true,
      threshold: 90,
      status: "completed",
    };
    expect(result.score).toBe(95);
    expect(result.passed).toBe(true);
    expect(result.threshold).toBe(90);
    expect(result.status).toBe("completed");
  });

  it("carries error_message for failed audits", () => {
    const result: LighthouseResult = {
      scenario: "test",
      brand_id: "b1",
      url: "http://localhost:3000",
      score: 0,
      passed: false,
      threshold: 90,
      status: "error",
      error_message: "Connection refused",
    };
    expect(result.status).toBe("error");
    expect(result.error_message).toBe("Connection refused");
  });

  it("represents pending state", () => {
    const result: LighthouseResult = {
      scenario: "test",
      brand_id: "b1",
      url: "http://localhost",
      score: 0,
      passed: false,
      threshold: 90,
      status: "pending",
    };
    expect(result.status).toBe("pending");
  });
});
