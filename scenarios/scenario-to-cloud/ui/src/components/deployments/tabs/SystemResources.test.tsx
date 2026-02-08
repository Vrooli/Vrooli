import "@testing-library/jest-dom";
import { render, screen } from "@testing-library/react";

import { SystemResources } from "./SystemResources";
import type { SystemState } from "../../../lib/api";

function baseSystem(): SystemState {
  return {
    cpu: {
      cores: 1,
      model: "DO-Regular",
      usage_percent: 5,
      load_average: [0, 0, 0],
    },
    memory: {
      total_mb: 961,
      used_mb: 400,
      free_mb: 561,
      usage_percent: 41.6,
    },
    disk: {
      total_gb: 23,
      used_gb: 15,
      free_gb: 8,
      usage_percent: 65,
    },
    swap: {
      total_mb: 0,
      used_mb: 0,
      usage_percent: 0,
    },
    ssh: {
      connected: true,
      latency_ms: 55,
      key_in_auth: false,
      key_path: "",
    },
    uptime_seconds: 1000,
  };
}

describe("SystemResources SSH key auth state", () => {
  test("shows unknown status and guidance when key state is unknown", () => {
    const system = baseSystem();
    system.ssh.key_in_auth_state = "unknown";

    render(<SystemResources system={system} />);

    expect(screen.getByText("Unknown")).toBeInTheDocument();
    expect(
      screen.getByText(/SSH key authorization is unknown because no deployment key path is configured/i)
    ).toBeInTheDocument();
  });

  test("shows unauthorized status and warning when key state is unauthorized", () => {
    const system = baseSystem();
    system.ssh.key_in_auth_state = "unauthorized";

    render(<SystemResources system={system} />);

    expect(screen.getByText("Unauthorized")).toBeInTheDocument();
    expect(
      screen.getByText(/The configured SSH key is not in the VPS authorized_keys/i)
    ).toBeInTheDocument();
  });
});
