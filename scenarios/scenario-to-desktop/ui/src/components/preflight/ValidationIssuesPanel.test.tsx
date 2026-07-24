/**
 * Tests for ValidationIssuesPanel and ValidationWarningsPanel components.
 */

import { describe, it, expect } from "vitest";
import { render, screen } from "@/test-utils";
import { ValidationIssuesPanel, ValidationWarningsPanel } from "./ValidationIssuesPanel";
import type { BundleValidationResult } from "../../lib/api";

describe("ValidationIssuesPanel", () => {
  it("renders nothing when validation is valid", () => {
    const validation: BundleValidationResult = {
      valid: true,
    };
    const { container } = render(<ValidationIssuesPanel validation={validation} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders errors section when errors exist", () => {
    const validation: BundleValidationResult = {
      valid: false,
      errors: [
        { code: "MANIFEST_PARSE_ERROR", message: "Failed to parse manifest" },
      ],
    };
    render(<ValidationIssuesPanel validation={validation} />);
    expect(screen.getByText("Validation issues")).toBeInTheDocument();
    expect(screen.getByText("Failed to parse manifest")).toBeInTheDocument();
  });

  it("renders service and path info for errors", () => {
    const validation: BundleValidationResult = {
      valid: false,
      errors: [
        {
          code: "SERVICE_ERROR",
          message: "Service failed",
          service: "api-server",
          path: "/path/to/file",
        },
      ],
    };
    render(<ValidationIssuesPanel validation={validation} />);
    expect(screen.getByText(/Service: api-server/)).toBeInTheDocument();
    expect(screen.getByText(/Path: \/path\/to\/file/)).toBeInTheDocument();
  });

  it("renders missing binaries section when binaries are missing", () => {
    const validation: BundleValidationResult = {
      valid: false,
      missing_binaries: [
        { service_id: "api", path: "/bin/api", platform: "linux" },
        { service_id: "worker", path: "/bin/worker", platform: "windows" },
      ],
    };
    render(<ValidationIssuesPanel validation={validation} />);
    expect(screen.getByText("Missing binaries")).toBeInTheDocument();
    expect(screen.getByText(/api: \/bin\/api \(linux\)/)).toBeInTheDocument();
    expect(screen.getByText(/worker: \/bin\/worker \(windows\)/)).toBeInTheDocument();
    expect(screen.getByText(/Rebuild the service binaries/)).toBeInTheDocument();
  });

  it("renders missing assets section when assets are missing", () => {
    const validation: BundleValidationResult = {
      valid: false,
      missing_assets: [
        { service_id: "ui", path: "/dist/index.html" },
        { service_id: "ui", path: "/dist/main.js" },
      ],
    };
    render(<ValidationIssuesPanel validation={validation} />);
    expect(screen.getByText("Missing assets")).toBeInTheDocument();
    expect(screen.getByText("ui: /dist/index.html")).toBeInTheDocument();
    expect(screen.getByText("ui: /dist/main.js")).toBeInTheDocument();
    expect(screen.getByText(/Rebuild UI\/assets/)).toBeInTheDocument();
  });

  it("renders invalid checksums section when checksums are invalid", () => {
    const validation: BundleValidationResult = {
      valid: false,
      invalid_checksums: [
        { service_id: "api", path: "/bin/api" },
      ],
    };
    render(<ValidationIssuesPanel validation={validation} />);
    expect(screen.getByText("Invalid checksums")).toBeInTheDocument();
    expect(screen.getByText("api: /bin/api")).toBeInTheDocument();
    expect(screen.getByText(/Re-export the bundle/)).toBeInTheDocument();
  });

  it("renders all sections together when multiple issues exist", () => {
    const validation: BundleValidationResult = {
      valid: false,
      errors: [{ code: "ERROR", message: "Some error" }],
      missing_binaries: [{ service_id: "api", path: "/bin", platform: "linux" }],
      missing_assets: [{ service_id: "ui", path: "/dist" }],
      invalid_checksums: [{ service_id: "worker", path: "/bin/worker" }],
    };
    render(<ValidationIssuesPanel validation={validation} />);
    expect(screen.getByText("Some error")).toBeInTheDocument();
    expect(screen.getByText("Missing binaries")).toBeInTheDocument();
    expect(screen.getByText("Missing assets")).toBeInTheDocument();
    expect(screen.getByText("Invalid checksums")).toBeInTheDocument();
  });
});

describe("ValidationWarningsPanel", () => {
  it("renders nothing when warnings array is empty", () => {
    const { container } = render(<ValidationWarningsPanel warnings={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing when warnings is undefined", () => {
    // @ts-expect-error - testing undefined case
    const { container } = render(<ValidationWarningsPanel warnings={undefined} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders warnings when they exist", () => {
    const warnings = [
      { code: "DEPRECATED_FIELD", message: "Field X is deprecated" },
      { code: "PERFORMANCE", message: "Consider optimizing Y" },
    ];
    render(<ValidationWarningsPanel warnings={warnings} />);
    expect(screen.getByText("Warnings")).toBeInTheDocument();
    expect(screen.getByText("Field X is deprecated")).toBeInTheDocument();
    expect(screen.getByText("Consider optimizing Y")).toBeInTheDocument();
  });
});
