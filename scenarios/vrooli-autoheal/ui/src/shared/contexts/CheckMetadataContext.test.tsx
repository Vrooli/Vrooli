import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { CheckMetadataProvider, useCheckMetadata } from "./CheckMetadataContext";

vi.mock("../../lib/api", () => ({
  fetchChecks: vi.fn().mockResolvedValue([
    { id: "infra-dns", title: "DNS", description: "DNS check", importance: "required", category: "infrastructure", intervalSeconds: 30 },
  ]),
}));

function Consumer() {
  const metadata = useCheckMetadata();
  return <div>{metadata.getTitle("infra-dns")}|{metadata.getTitle("unknown")}|{metadata.getMetadata("infra-dns")?.description}</div>;
}

describe("CheckMetadataContext", () => {
  it("indexes check metadata and provides fallbacks", async () => {
    renderWithProviders(
      <CheckMetadataProvider>
        <Consumer />
      </CheckMetadataProvider>,
    );
    expect(await screen.findByText("DNS|unknown|DNS check")).toBeInTheDocument();
  });
});
