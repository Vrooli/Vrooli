/**
 * Tests for the tool component registry (getToolComponent).
 *
 * Verifies that each known tool name maps to the correct specialized
 * component, and that unknown/empty/undefined names fall back to the
 * generic AgentToolCallCard.
 */

import { describe, it, expect } from "vitest";
import { getToolComponent } from ".";
import { AgentBashCard } from "./AgentBashCard";
import { AgentFileToolCard } from "./AgentFileToolCard";
import { AgentWebFetchCard } from "./AgentWebFetchCard";
import { AgentTaskCard } from "./AgentTaskCard";
import AgentToolCallCard from "../AgentToolCallCard";

describe("getToolComponent", () => {
  it("returns AgentBashCard for 'Bash'", () => {
    expect(getToolComponent("Bash")).toBe(AgentBashCard);
  });

  it("returns AgentFileToolCard for 'Read'", () => {
    expect(getToolComponent("Read")).toBe(AgentFileToolCard);
  });

  it("returns AgentFileToolCard for 'Write'", () => {
    expect(getToolComponent("Write")).toBe(AgentFileToolCard);
  });

  it("returns AgentFileToolCard for 'Edit'", () => {
    expect(getToolComponent("Edit")).toBe(AgentFileToolCard);
  });

  it("returns AgentFileToolCard for 'Glob'", () => {
    expect(getToolComponent("Glob")).toBe(AgentFileToolCard);
  });

  it("returns AgentFileToolCard for 'Grep'", () => {
    expect(getToolComponent("Grep")).toBe(AgentFileToolCard);
  });

  it("returns AgentWebFetchCard for 'WebFetch'", () => {
    expect(getToolComponent("WebFetch")).toBe(AgentWebFetchCard);
  });

  it("returns AgentWebFetchCard for 'WebSearch'", () => {
    expect(getToolComponent("WebSearch")).toBe(AgentWebFetchCard);
  });

  it("returns AgentTaskCard for 'Task'", () => {
    expect(getToolComponent("Task")).toBe(AgentTaskCard);
  });

  it("returns AgentToolCallCard (fallback) for unknown tool name", () => {
    expect(getToolComponent("SomethingUnknown")).toBe(AgentToolCallCard);
  });

  it("returns AgentToolCallCard (fallback) for undefined", () => {
    expect(getToolComponent(undefined)).toBe(AgentToolCallCard);
  });

  it("returns AgentToolCallCard (fallback) for empty string", () => {
    expect(getToolComponent("")).toBe(AgentToolCallCard);
  });
});
