import { screen } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { describe, expect, it, vi } from "vitest";

import type { AuthProfile, ProviderStatus } from "../../api/authentication";

vi.mock("../../i18n", () => ({
  useTranslation: () => ({ t: (key: string, values?: Record<string, string>) => `${key}${values ? `:${Object.values(values).join(":")}` : ""}` }),
}));

import { AuthenticationProfilesCard } from "./AuthenticationProfilesCard";

const profile: AuthProfile = {
  id: "profile-1",
  device_id: "android-test",
  method: "pin",
  credential_identity: "device-control/android-test/profile-1",
  credential_field: "unlock",
  verification: "fresh_lock_state_unlocked",
  policy: { max_attempts: 1, attempt_limit: 15_000_000_000, settle: 750_000_000 },
  status: "active",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("AuthenticationProfilesCard", () => {
  it("renders metadata and provider state without accepting a credential value", () => {
    const providers: Record<string, ProviderStatus> = {
      [profile.id]: { provider: "test", provider_state: "available", configured: true },
    };
    renderWithProviders(<AuthenticationProfilesCard profiles={[profile]} providers={providers} />);

    expect(screen.getByTestId("authentication-profiles-card")).toHaveTextContent("profile-1");
    expect(screen.getByTestId("authentication-profiles-card")).toHaveTextContent("available");
    expect(screen.getByTestId("authentication-profiles-card")).not.toHaveTextContent("credential_value");
  });

  it("renders the empty state when no profiles are configured", () => {
    renderWithProviders(<AuthenticationProfilesCard profiles={[]} providers={{}} />);
    expect(screen.getByTestId("authentication-profiles-card")).toHaveTextContent("auth.none");
  });

  it("reports missing provider state and never exposes optional credential data", () => {
    renderWithProviders(
      <AuthenticationProfilesCard
        profiles={[{ ...profile, last_outcome: "failed" }]}
        providers={{}}
      />,
    );
    const card = screen.getByTestId("authentication-profiles-card");
    expect(card).toHaveTextContent("not checked");
    expect(card).toHaveTextContent("auth.unconfigured");
    expect(card).not.toHaveTextContent("credential_value");
  });
});
