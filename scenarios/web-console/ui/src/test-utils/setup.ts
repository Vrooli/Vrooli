import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Automatic cleanup after each test — prevents leaked DOM state
// between tests that use @testing-library/react render().
afterEach(() => {
  cleanup();
});
