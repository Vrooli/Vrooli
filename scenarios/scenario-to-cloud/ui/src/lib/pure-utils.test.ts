import { describe, expect, it } from "vitest";
import {
  detectLanguage,
  getLanguageFromFilename,
  normalizeLanguage,
} from "./languageDetection";
import {
  getInitialSteps,
  updateStepStatus,
} from "../types/progress";
import {
  isReservedKey,
  isValidSecretKey,
} from "../types/secrets";
import {
  getLineForPath,
  getLinesForPaths,
  mapJsonPathsToLines,
} from "./jsonPathToLine";

describe("language detection utilities", () => {
  it.each([
    ["{\"name\": \"vrooli\"}", "json"],
    ["interface User { id: number }", "typescript"],
    ["const answer = 42", "javascript"],
    ["def greet(name):\n    return name", "python"],
    ["package main\n\nfunc main() {}", "go"],
    ["SELECT id FROM users", "sql"],
    ["#!/usr/bin/env bash\necho hello", "bash"],
    ["<div class=\"card\">Hello</div>", "html"],
    [".card { color: red; }", "css"],
    ["name:\n  value: vrooli", "yaml"],
    ["# Deployment\n\n**Important**: [guide](https://example.com)", "markdown"],
    ["fn main() { let mut value = 1; }", "rust"],
    ["[server]\nport = 8080", "toml"],
    ["FROM node:20\nWORKDIR /app", "dockerfile"],
    [".PHONY: test\ntest:\n\t pnpm test", "makefile"],
  ] as const)("detects %s as %s", (source, expected) => {
    expect(detectLanguage(source)).toBe(expected);
  });

  it("returns text for empty or unrecognized snippets", () => {
    expect(detectLanguage("")).toBe("text");
    expect(detectLanguage("plain prose without syntax markers")).toBe("text");
  });

  it.each([
    ["src/App.tsx", "tsx"],
    ["config.JSON", "json"],
    ["Dockerfile", "dockerfile"],
    [".env", "bash"],
    ["README", null],
    ["archive.unknown", null],
  ] as const)("maps filename %s to %s", (filename, expected) => {
    expect(getLanguageFromFilename(filename)).toBe(expected);
  });

  it.each([
    ["js", "javascript"],
    [" TS ", "typescript"],
    ["shell", "bash"],
    ["yml", "yaml"],
    ["dockerfile", "docker"],
    ["c++", "cpp"],
    ["custom-language", "custom-language"],
  ] as const)("normalizes %s to %s", (language, expected) => {
    expect(normalizeLanguage(language)).toBe(expected);
  });
});

describe("JSON path line mapping", () => {
  const json = [
    "{",
    '  "target": {',
    '    "vps": {',
    '      "host": "example.com",',
    '      "port": 443',
    "    },",
    '    "enabled": true',
    "  }",
    "}",
  ].join("\n");

  it("maps nested keys with one-indexed lines and key columns", () => {
    const mapping = mapJsonPathsToLines(json);

    expect(mapping.target).toEqual({ line: 2, column: 2 });
    expect(mapping["target.vps"]).toEqual({ line: 3, column: 4 });
    expect(mapping["target.vps.host"]).toEqual({ line: 4, column: 6 });
    expect(getLineForPath(json, "target.vps.port")).toBe(5);
    expect(getLineForPath(json, "target.missing")).toBeNull();
  });

  it("returns only found paths for a batch lookup", () => {
    expect(getLinesForPaths(json, ["target", "target.enabled", "missing"]))
      .toEqual({ target: 2, "target.enabled": 7 });
  });

  it("keeps escaped key text and supports a top-level key", () => {
    const escaped = '{\n  "top\\\"level": {\n    "value": 1\n  }\n}';
    const mapping = mapJsonPathsToLines(escaped);

    expect(mapping["top\\\"level"]).toEqual({ line: 2, column: 2 });
    expect(mapping['top\\"level.value']?.line).toBe(3);
  });
});

describe("deployment progress utilities", () => {
  it("creates every declared deployment step as pending", () => {
    const steps = getInitialSteps();

    expect(steps.length).toBeGreaterThan(10);
    expect(steps[0]).toEqual({
      id: "bundle_build",
      title: "Building bundle",
      status: "pending",
    });
    expect(steps.every((step) => step.status === "pending")).toBe(true);
  });

  it("updates a matching step while preserving unrelated steps", () => {
    const initial = getInitialSteps();
    const updated = updateStepStatus(initial, "upload", "completed");

    expect(updated.find((step) => step.id === "upload")?.status).toBe("completed");
    expect(updated.find((step) => step.id === "extract")?.status).toBe("pending");
    expect(initial.find((step) => step.id === "upload")?.status).toBe("pending");
  });

  it("initializes steps when an event arrives before state exists", () => {
    const steps = updateStepStatus(undefined, "setup", "running");

    expect(steps.find((step) => step.id === "setup")?.status).toBe("running");
    expect(steps.filter((step) => step.status === "pending")).not.toHaveLength(0);
  });
});

describe("custom secret key validation", () => {
  it.each(["API_KEY", "A", "VROOLI2_TOKEN", "A_B_C"]) (
    "accepts valid key %s",
    (key) => expect(isValidSecretKey(key)).toBe(true),
  );

  it.each(["", "api_key", "2FAST", "A-B", "A B", "_KEY"]) (
    "rejects invalid key %s",
    (key) => expect(isValidSecretKey(key)).toBe(false),
  );

  it("identifies reserved prefixes independently of format validation", () => {
    expect(isReservedKey("VROOLI_INTERNAL_TOKEN")).toBe(true);
    expect(isReservedKey("_SYSTEM_TOKEN")).toBe(true);
    expect(isReservedKey("PUBLIC_TOKEN")).toBe(false);
  });
});
