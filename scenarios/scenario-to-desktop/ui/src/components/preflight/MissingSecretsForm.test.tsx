/**
 * Tests for MissingSecretsForm component.
 */

import { describe, it, expect, vi } from "vitest";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { render, screen, fireEvent } from "@/test-utils";
import { MissingSecretsForm } from "./MissingSecretsForm";
import {
  PreflightSecretSchema,
  SecretClass,
  type PreflightSecret,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";

const makeSecret = (init: MessageInitShape<typeof PreflightSecretSchema>) =>
  create(PreflightSecretSchema, init);

describe("MissingSecretsForm", () => {
  const mockOnSecretChange = vi.fn();
  const mockOnApplySecrets = vi.fn();

  const defaultProps = {
    missingSecrets: [] as PreflightSecret[],
    secretInputs: {} as Record<string, string>,
    preflightPending: false,
    onSecretChange: mockOnSecretChange,
    onApplySecrets: mockOnApplySecrets,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing when no missing secrets", () => {
    const { container } = render(<MissingSecretsForm {...defaultProps} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders form when there are missing secrets", () => {
    const missingSecrets = [
      makeSecret({
        id: "API_KEY",
        secretClass: SecretClass.API_KEY,
        prompt: {
          label: "API Key",
          hint: "Enter your API key",
        },
      }),
    ];
    render(
      <MissingSecretsForm {...defaultProps} missingSecrets={missingSecrets} />,
    );
    expect(screen.getByText("Missing required secrets")).toBeInTheDocument();
    expect(screen.getByText("API Key")).toBeInTheDocument();
  });

  it("displays secret id as label when no prompt label provided", () => {
    const missingSecrets = [makeSecret({ id: "SECRET_VALUE" })];
    render(
      <MissingSecretsForm {...defaultProps} missingSecrets={missingSecrets} />,
    );
    expect(screen.getByText("SECRET_VALUE")).toBeInTheDocument();
  });

  it("displays class information when provided", () => {
    const missingSecrets = [
      makeSecret({
        id: "TOKEN",
        secretClass: SecretClass.TOKEN,
      }),
    ];
    render(
      <MissingSecretsForm {...defaultProps} missingSecrets={missingSecrets} />,
    );
    expect(screen.getByText("token")).toBeInTheDocument();
  });

  it("calls onSecretChange when input value changes", () => {
    const missingSecrets = [makeSecret({ id: "MY_SECRET" })];
    render(
      <MissingSecretsForm {...defaultProps} missingSecrets={missingSecrets} />,
    );

    const input = screen.getByPlaceholderText("Enter value");
    fireEvent.change(input, { target: { value: "secret-value" } });

    expect(mockOnSecretChange).toHaveBeenCalledWith(
      "MY_SECRET",
      "secret-value",
    );
  });

  it("displays existing secret values in inputs", () => {
    const missingSecrets = [makeSecret({ id: "EXISTING_SECRET" })];
    const secretInputs = { EXISTING_SECRET: "pre-filled-value" };

    render(
      <MissingSecretsForm
        {...defaultProps}
        missingSecrets={missingSecrets}
        secretInputs={secretInputs}
      />,
    );

    const input = screen.getByDisplayValue("pre-filled-value");
    expect(input).toBeInTheDocument();
  });

  it("calls onApplySecrets with secretInputs when button clicked", () => {
    const missingSecrets = [makeSecret({ id: "SECRET1" })];
    const secretInputs = { SECRET1: "value1" };

    render(
      <MissingSecretsForm
        {...defaultProps}
        missingSecrets={missingSecrets}
        secretInputs={secretInputs}
      />,
    );

    const button = screen.getByRole("button", { name: /apply secrets/i });
    fireEvent.click(button);

    expect(mockOnApplySecrets).toHaveBeenCalledWith(secretInputs);
  });

  it("disables button when preflightPending is true", () => {
    const missingSecrets = [makeSecret({ id: "SECRET" })];

    render(
      <MissingSecretsForm
        {...defaultProps}
        missingSecrets={missingSecrets}
        preflightPending={true}
      />,
    );

    const button = screen.getByRole("button", { name: /apply secrets/i });
    expect(button).toBeDisabled();
  });

  it("renders multiple secrets correctly", () => {
    const missingSecrets = [
      makeSecret({
        id: "API_KEY",
        prompt: { label: "API Key", hint: "Your API key" },
      }),
      makeSecret({
        id: "DATABASE_URL",
        prompt: { label: "Database URL", hint: "PostgreSQL connection string" },
      }),
      makeSecret({
        id: "JWT_SECRET",
        prompt: { label: "JWT Secret" },
      }),
    ];

    render(
      <MissingSecretsForm {...defaultProps} missingSecrets={missingSecrets} />,
    );

    expect(screen.getByText("API Key")).toBeInTheDocument();
    expect(screen.getByText("Database URL")).toBeInTheDocument();
    expect(screen.getByText("JWT Secret")).toBeInTheDocument();
  });

  it("uses hint as placeholder when provided", () => {
    const missingSecrets = [
      makeSecret({
        id: "SECRET",
        prompt: { label: "Secret", hint: "Custom placeholder" },
      }),
    ];

    render(
      <MissingSecretsForm {...defaultProps} missingSecrets={missingSecrets} />,
    );

    const input = screen.getByPlaceholderText("Custom placeholder");
    expect(input).toBeInTheDocument();
  });

  it("uses default placeholder when no hint provided", () => {
    const missingSecrets = [makeSecret({ id: "SECRET" })];

    render(
      <MissingSecretsForm {...defaultProps} missingSecrets={missingSecrets} />,
    );

    const input = screen.getByPlaceholderText("Enter value");
    expect(input).toBeInTheDocument();
  });
});
