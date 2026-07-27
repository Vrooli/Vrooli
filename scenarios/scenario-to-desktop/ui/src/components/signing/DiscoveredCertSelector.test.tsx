import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { DiscoveredCertSelector } from "./DiscoveredCertSelector";

describe("DiscoveredCertSelector", () => {
  it("stays absent when discovery returned no certificates", () => {
    const { container } = render(<DiscoveredCertSelector label="Certificates" discovered={[]} onSelect={vi.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("selects a discovered certificate and warns about imminent expiry", () => {
    const onSelect = vi.fn();
    const discovered = [
      { id: "current", name: "Current certificate", days_to_expiry: 90 },
      { id: "expiring", subject: "CN=Expiring", days_to_expiry: 14, is_expired: false },
      { id: "expired", name: "Expired", days_to_expiry: 0, is_expired: true },
    ] as never[];
    render(<DiscoveredCertSelector label="Certificates" discovered={discovered} onSelect={onSelect} expiryWarningText="Renew before packaging." />);
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "expiring" } });
    expect(onSelect).toHaveBeenCalledWith(discovered[1]);
    expect(screen.getByText("Renew before packaging.")).toBeInTheDocument();
  });
});
