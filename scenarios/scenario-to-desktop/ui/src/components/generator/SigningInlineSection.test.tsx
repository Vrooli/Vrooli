import { fireEvent, render, screen } from "@/test-utils";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SigningInlineSection } from "./SigningInlineSection";

function renderSection(
  overrides: Partial<ComponentProps<typeof SigningInlineSection>> = {},
) {
  const props = {
    scenarioName: "canvas-lab",
    signingEnabled: false,
    signingConfig: null,
    readiness: undefined,
    loading: false,
    onToggleSigning: vi.fn(),
    onOpenSigning: vi.fn(),
    onRefresh: vi.fn(),
    ...overrides,
  };
  render(<SigningInlineSection {...props} />);
  return props;
}

afterEach(() => {
  window.localStorage.removeItem("std_signing_expiry_warning");
});

describe("SigningInlineSection", () => {
  it("warns about unsigned builds and exposes intentional signing actions", () => {
    const props = renderSection();
    expect(
      screen.getByText(/Unsigned installers may trigger OS warnings/),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Enable signing now" }));
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    fireEvent.click(screen.getByRole("button", { name: /Open Signing tab/ }));
    expect(props.onToggleSigning).toHaveBeenCalledWith(true);
    expect(props.onRefresh).toHaveBeenCalledOnce();
    expect(props.onOpenSigning).toHaveBeenCalledOnce();
  });

  it("reports missing configuration and actionable readiness issues", () => {
    const missing = renderSection({ signingEnabled: true });
    expect(screen.getByText(/No signing config saved yet/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "Sign this build" }));
    expect(missing.onToggleSigning).toHaveBeenCalledWith(false);

    const issue = renderSection({
      signingEnabled: true,
      signingConfig: {} as never,
      readiness: {
        ready: false,
        issues: ["Apple certificate expired"],
      } as never,
    });
    expect(screen.getByText("Apple certificate expired")).toBeInTheDocument();
    expect(issue.onRefresh).not.toHaveBeenCalled();
  });

  it("shows a ready state, persisted expiry warning, and a loading status", () => {
    window.localStorage.setItem(
      "std_signing_expiry_warning",
      "Certificate expires in 7 days",
    );
    const { unmount } = render(
      <SigningInlineSection
        scenarioName="canvas-lab"
        signingEnabled={true}
        signingConfig={{} as never}
        readiness={{ ready: true, issues: [] } as never}
        loading={false}
        onToggleSigning={vi.fn()}
        onOpenSigning={vi.fn()}
        onRefresh={vi.fn()}
      />,
    );
    expect(
      screen.getByText(/Signing ready for at least one platform/),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Certificate expires in 7 days"),
    ).toBeInTheDocument();

    unmount();
    renderSection({ signingEnabled: true, loading: true });
    expect(screen.getByText("Checking signing status…")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Refresh" })).toBeDisabled();
  });

  it("does not allow signing actions without a selected scenario", () => {
    renderSection({ scenarioName: "" });
    expect(
      screen.getByRole("checkbox", { name: "Skip signing for this build" }),
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Refresh" })).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Enable signing now" }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /Open Signing tab/ }),
    ).toBeDisabled();
  });
});
