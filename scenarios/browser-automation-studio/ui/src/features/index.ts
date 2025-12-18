/**
 * Features module - domain-organized feature components
 *
 * The Vrooli Ascension is organized into these main features:
 * - projects: Project management (Dashboard, ProjectDetail, ProjectModal)
 * - workflows: Visual workflow builder and node types
 * - execution: Execution viewing, history, and replay
 * - ai: AI-powered workflow generation and editing
 * - settings: Application settings and preferences
 * - onboarding: Tutorial and first-time user experience
 * - docs: In-app documentation hub (Getting Started, Node Reference, Schema Reference)
 */
export * from "./projects";
export * from "./workflows";
export * from "./execution";
export * from "./ai";
export * from "./settings";
export * from "@shared/onboarding";
export * from "./docs";
