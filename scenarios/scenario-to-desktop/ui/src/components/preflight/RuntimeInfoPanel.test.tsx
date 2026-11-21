import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RuntimeInfoPanel } from "./RuntimeInfoPanel";
import { renderWithProviders } from "@vrooli/api-base/testing";

describe("RuntimeInfoPanel", () => {
  it("presents complete runtime identity without hiding paths or version provenance", () => {
    renderWithProviders(
      <RuntimeInfoPanel
        runtimeInfo={{
          instance_id: "instance-123456789",
          started_at: "2026-07-27T12:00:00Z",
          dry_run: true,
          manifest_hash: "abcdef123456789",
          app_name: "Canvas",
          app_version: "1.2.3",
          ipc_host: "127.0.0.1",
          ipc_port: 8123,
          runtime_version: "2.0.0",
          build_version: "build-7",
          bundle_root: "/bundle/canvas",
          app_data_dir: "/data/canvas",
        }}
      />,
    );
    expect(screen.getByText("instance-123")).toHaveAttribute(
      "title",
      "instance-123456789",
    );
    expect(screen.getByText("abcdef123456")).toHaveAttribute(
      "title",
      "abcdef123456789",
    );
    expect(screen.getByText("Canvas v1.2.3")).toBeInTheDocument();
    expect(screen.getByText("127.0.0.1:8123")).toBeInTheDocument();
    expect(screen.getByText("2.0.0 · build build-7")).toBeInTheDocument();
    expect(screen.getByText("/bundle/canvas")).toBeInTheDocument();
    expect(screen.getByText("/data/canvas")).toBeInTheDocument();
  });

  it("uses explicit unknown and no values for incomplete runtime data", () => {
    renderWithProviders(<RuntimeInfoPanel runtimeInfo={{}} />);
    expect(screen.getAllByText("Unknown")).toHaveLength(6);
    expect(screen.getByText("no")).toBeInTheDocument();
  });
});
