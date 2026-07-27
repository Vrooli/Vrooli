import { fireEvent, render, screen } from "@/test-utils";
import { Shield } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { SigningFormWrapper } from "./SigningFormWrapper";

describe("SigningFormWrapper", () => {
  it("keeps disabled signing forms explanatory and toggles configuration accessibly", () => {
    const onToggle = vi.fn();
    render(<SigningFormWrapper platform="Windows" platformId="windows" icon={Shield} isConfigured={false} onToggle={onToggle} disabledMessage="Enable Windows signing to add a certificate." headerActions={<button type="button">Discover</button>} testId="windows-signing"><p>Certificate fields</p></SigningFormWrapper>);
    expect(screen.getByText("Enable Windows signing to add a certificate.")).toBeInTheDocument();
    expect(screen.queryByText("Certificate fields")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Discover" })).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Configure"));
    expect(onToggle).toHaveBeenCalledWith(true);
  });

  it("renders configured form controls", () => {
    render(<SigningFormWrapper platform="Linux" platformId="linux" icon={Shield} isConfigured onToggle={vi.fn()} disabledMessage="unused"><p>Certificate fields</p></SigningFormWrapper>);
    expect(screen.getByText("Certificate fields")).toBeInTheDocument();
    expect(screen.getByLabelText("Configure")).toBeChecked();
  });
});
