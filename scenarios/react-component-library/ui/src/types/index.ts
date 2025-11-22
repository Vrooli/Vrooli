// Core type definitions for react-component-library

export interface Component {
  id: string;
  libraryId: string;
  displayName: string;
  description: string;
  version: string;
  filePath: string;
  sourcePath: string;
  tags: string[];
  category?: string;
  techStack: string[];
  createdAt: string;
  updatedAt: string;
}

export interface ComponentVersion {
  id: string;
  componentId: string;
  version: string;
  content: string;
  changelog?: string;
  createdAt: string;
}

export interface AdoptionRecord {
  id: string;
  componentId: string;
  componentLibraryId: string;
  scenarioName: string;
  adoptedPath: string;
  version: string;
  status: "current" | "behind" | "modified" | "unknown";
  createdAt: string;
  updatedAt: string;
}

export interface ViewportPreset {
  id: string;
  name: string;
  width: number;
  height: number;
  icon: string;
}

export interface EmulatorFrame {
  id: string;
  preset: ViewportPreset;
  props?: Record<string, unknown>;
}

export interface AIMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: string;
  codeSnippet?: string;
  attachedElements?: string[];
}

export interface ElementSelection {
  frameId: string;
  selector: string;
  xpath: string;
  tagName: string;
  attributes: Record<string, string>;
}

export const VIEWPORT_PRESETS: ViewportPreset[] = [
  { id: "desktop", name: "Desktop", width: 1920, height: 1080, icon: "Monitor" },
  { id: "laptop", name: "Laptop", width: 1366, height: 768, icon: "Laptop" },
  { id: "tablet", name: "Tablet", width: 768, height: 1024, icon: "Tablet" },
  { id: "mobile", name: "Mobile", width: 375, height: 667, icon: "Smartphone" },
];
