import { describe, it, expect, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { ProviderTierBadge } from "./ProviderTierBadge";

afterEach(() => {
  cleanup();
});

describe("ProviderTierBadge", () => {
  it("reflects LOCAL tier copy", () => {
    render(<ProviderTierBadge tier={ProviderTier.LOCAL} />);
    expect(screen.getByText(/^Local$/)).toBeInTheDocument();
  });

  it("reflects BYOK tier copy", () => {
    render(<ProviderTierBadge tier={ProviderTier.BYOK} />);
    expect(screen.getByText(/^BYOK$/)).toBeInTheDocument();
  });

  it("reflects VROOLI tier copy", () => {
    render(<ProviderTierBadge tier={ProviderTier.VROOLI} />);
    expect(screen.getByText(/^Vrooli$/)).toBeInTheDocument();
  });

  it("falls back to dash on unknown tier", () => {
    render(<ProviderTierBadge tier={ProviderTier.UNSPECIFIED} />);
    expect(screen.getByText(/^—$/)).toBeInTheDocument();
  });
});
