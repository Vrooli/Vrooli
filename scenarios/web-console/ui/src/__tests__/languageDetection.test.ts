import { describe, it, expect } from "vitest";
import { detectLanguage, normalizeLanguage } from "../components/markdown/utils/languageDetection";

describe("detectLanguage", () => {
  it("returns 'text' for empty input", () => {
    expect(detectLanguage("")).toBe("text");
    expect(detectLanguage("   ")).toBe("text");
  });

  it("detects JSON", () => {
    expect(detectLanguage('{ "name": "test", "version": "1.0" }')).toBe("json");
  });

  it("detects TypeScript", () => {
    expect(detectLanguage("interface User {\n  name: string;\n  age: number;\n}")).toBe("typescript");
  });

  it("detects Python", () => {
    expect(detectLanguage("def hello(name):\n  print(f'Hello {name}')\n\nif __name__ == '__main__':\n  hello('world')")).toBe("python");
  });

  it("detects Go", () => {
    expect(detectLanguage("package main\n\nfunc main() {\n  fmt.Println(\"hello\")\n}")).toBe("go");
  });

  it("detects SQL", () => {
    expect(detectLanguage("SELECT id, name FROM users WHERE age > 18")).toBe("sql");
  });

  it("detects bash", () => {
    expect(detectLanguage("#!/bin/bash\necho \"Hello $USER\"")).toBe("bash");
  });

  it("returns 'text' for ambiguous content", () => {
    expect(detectLanguage("hello world")).toBe("text");
  });
});

describe("normalizeLanguage", () => {
  it("maps js to javascript", () => {
    expect(normalizeLanguage("js")).toBe("javascript");
  });

  it("maps ts to typescript", () => {
    expect(normalizeLanguage("ts")).toBe("typescript");
  });

  it("maps py to python", () => {
    expect(normalizeLanguage("py")).toBe("python");
  });

  it("maps sh to bash", () => {
    expect(normalizeLanguage("sh")).toBe("bash");
  });

  it("maps shell to bash", () => {
    expect(normalizeLanguage("shell")).toBe("bash");
  });

  it("maps yml to yaml", () => {
    expect(normalizeLanguage("yml")).toBe("yaml");
  });

  it("passes through unknown languages", () => {
    expect(normalizeLanguage("rust")).toBe("rust");
    expect(normalizeLanguage("kotlin")).toBe("kotlin");
  });

  it("is case-insensitive", () => {
    expect(normalizeLanguage("JS")).toBe("javascript");
    expect(normalizeLanguage("TypeScript")).toBe("typescript");
  });
});
