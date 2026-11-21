import { create } from "@bufbuild/protobuf";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  AssessmentFindingSchema,
  MaturityAssessmentSchema,
} from "@vrooli/proto-types/common/v1/maturity_pb";
import {
  ProtoFileSchema,
  ProtoRpcSchema,
  ProtoServiceSchema,
  ProtoSurfaceSchema,
  TransportKind,
  TransportWorld,
} from "@vrooli/proto-types/proto-health/v1/shared/surface_pb";
import { DescribeScenarioProtosResponseSchema } from "@vrooli/proto-types/proto-health/v1/validation/validation_pb";
import {
  ValidateScenarioResponseSchema,
  ValidationStatus,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ProtoHealthPanel } from "./ProtoHealthPanel";
import { describeScenarioProtos, validateScenario } from "../../api/protoHealth";

vi.mock("../../api/protoHealth", () => ({
  describeScenarioProtos: vi.fn(),
  validateScenario: vi.fn(),
}));

const validateScenarioMock = vi.mocked(validateScenario);
const describeScenarioProtosMock = vi.mocked(describeScenarioProtos);

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ProtoHealthPanel", () => {
  it("renders a passing validation report and surface facts", async () => {
    validateScenarioMock.mockResolvedValue(
      create(ValidateScenarioResponseSchema, {
        scenario: "proto-health",
        status: ValidationStatus.PASSED,
        assessment: create(MaturityAssessmentSchema, {
          findingsBySeverity: {
            SEVERITY_ERROR: 0,
            SEVERITY_WARNING: 1,
            SEVERITY_INFO: 2,
          },
          findings: [],
        }),
      }),
    );
    describeScenarioProtosMock.mockResolvedValue(
      create(DescribeScenarioProtosResponseSchema, {
        surface: create(ProtoSurfaceSchema, {
          scenario: "proto-health",
          transportWorld: TransportWorld.CONNECT,
          files: [
            create(ProtoFileSchema, {
              path: "proto-health/v1/validation/validation.proto",
              package: "vrooli.proto_health.v1.validation",
              version: "v1",
              domain: "validation",
            }),
          ],
          services: [
            create(ProtoServiceSchema, {
              name: "ProtoHealthService",
              fullName: "vrooli.proto_health.v1.validation.ProtoHealthService",
              domain: "validation",
              rpcs: [
                create(ProtoRpcSchema, {
                  name: "DescribeScenarioProtos",
                  transport: TransportKind.CONNECT,
                }),
              ],
            }),
          ],
        }),
      }),
    );

    renderWithProviders(<ProtoHealthPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.protoHealth.status)).toHaveTextContent(
        "protoHealth.status.passed",
      );
    });
    expect(screen.getAllByText("proto-health")).toHaveLength(2);
    expect(screen.getByText("protoHealth.empty")).toBeInTheDocument();
    expect(screen.getAllByText("1").length).toBeGreaterThanOrEqual(3);
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("ProtoHealthService")).toBeInTheDocument();
    expect(screen.getByText("protoHealth.transport.connect")).toBeInTheDocument();
  });

  it("submits a custom scenario and renders blocking findings", async () => {
    validateScenarioMock.mockResolvedValue(
      create(ValidateScenarioResponseSchema, {
        scenario: "cli-health",
        status: ValidationStatus.FAILED,
        assessment: create(MaturityAssessmentSchema, {
          findingsBySeverity: { SEVERITY_ERROR: 1 },
          findings: [
            create(AssessmentFindingSchema, {
              code: "proto.package_mismatch",
              severity: "SEVERITY_ERROR",
              message: "package does not match scenario",
              location: "cli-health/v1/api.proto",
            }),
          ],
        }),
      }),
    );
    describeScenarioProtosMock.mockResolvedValue(
      create(DescribeScenarioProtosResponseSchema, {
        surface: create(ProtoSurfaceSchema, {
          scenario: "cli-health",
          transportWorld: TransportWorld.NONE,
        }),
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ProtoHealthPanel />);
    await user.clear(screen.getByTestId(selectors.protoHealth.scenarioInput));
    await user.type(screen.getByTestId(selectors.protoHealth.scenarioInput), "cli-health");
    await user.click(screen.getByTestId(selectors.protoHealth.runButton));

    await waitFor(() => {
      expect(validateScenarioMock).toHaveBeenLastCalledWith("cli-health");
    });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.protoHealth.status)).toHaveTextContent(
        "protoHealth.status.blocked",
      );
    });
    expect(screen.getByTestId(selectors.protoHealth.finding)).toHaveTextContent(
      "proto.package_mismatch",
    );
    expect(screen.getByText("protoHealth.noServices")).toBeInTheDocument();
  });

  it("quick target buttons update the selected scenario immediately", async () => {
    validateScenarioMock.mockResolvedValue(
      create(ValidateScenarioResponseSchema, {
        scenario: "measures-health",
        status: ValidationStatus.PASSED,
        assessment: create(MaturityAssessmentSchema),
      }),
    );
    describeScenarioProtosMock.mockResolvedValue(
      create(DescribeScenarioProtosResponseSchema, {
        surface: create(ProtoSurfaceSchema, {
          scenario: "measures-health",
          transportWorld: TransportWorld.MIXED,
        }),
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<ProtoHealthPanel />);
    await user.click(
      screen.getByTestId(selectors.protoHealth.quickTarget({ scenario: "measures-health" })),
    );

    await waitFor(() => {
      expect(validateScenarioMock).toHaveBeenLastCalledWith("measures-health");
    });
    expect(screen.getByTestId(selectors.protoHealth.scenarioInput)).toHaveValue("measures-health");
    expect(screen.getByText("protoHealth.transport.mixed")).toBeInTheDocument();
  });

  it("renders API errors without showing stale report content", async () => {
    validateScenarioMock.mockRejectedValue(new Error("network down"));
    describeScenarioProtosMock.mockResolvedValue(
      create(DescribeScenarioProtosResponseSchema, {
        surface: create(ProtoSurfaceSchema),
      }),
    );

    renderWithProviders(<ProtoHealthPanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.protoHealth.error)).toHaveTextContent("network down");
    });
    expect(screen.queryByTestId(selectors.protoHealth.status)).not.toBeInTheDocument();
  });
});
