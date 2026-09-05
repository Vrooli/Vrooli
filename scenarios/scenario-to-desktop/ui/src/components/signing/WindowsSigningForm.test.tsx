import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { WindowsSigningForm } from "./WindowsSigningForm";

describe("WindowsSigningForm", () => {
  it("enables secure defaults and can disable Windows signing", () => {
    const onChange = vi.fn();
    const { rerender } = render(<WindowsSigningForm onChange={onChange} />);
    expect(screen.getByText(/Enable Windows signing/)).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Configure"));
    expect(onChange).toHaveBeenCalledWith({
      certificate_source: "file",
      timestamp_server: "http://timestamp.digicert.com",
      sign_algorithm: "sha256",
    });
    rerender(
      <WindowsSigningForm
        config={{ certificate_source: "file" }}
        onChange={onChange}
      />,
    );
    fireEvent.click(screen.getByLabelText("Configure"));
    expect(onChange).toHaveBeenLastCalledWith(undefined);
  });

  it("edits file certificate details, timestamping, algorithm, and dual signing", () => {
    const onChange = vi.fn();
    const config = {
      certificate_source: "file" as const,
      certificate_file: "old.pfx",
    };
    render(<WindowsSigningForm config={config} onChange={onChange} />);
    fireEvent.change(screen.getByLabelText("Certificate File Path"), {
      target: { value: "/secure/release.pfx" },
    });
    fireEvent.change(screen.getByLabelText("Password Environment Variable"), {
      target: { value: "WINDOWS_SIGNING_SECRET_ENV" },
    });
    fireEvent.change(screen.getByLabelText("Timestamp Server"), {
      target: { value: "http://timestamp.sectigo.com" },
    });
    fireEvent.change(screen.getByLabelText("Signing Algorithm"), {
      target: { value: "sha512" },
    });
    fireEvent.click(screen.getByLabelText(/Dual Sign/));
    expect(onChange).toHaveBeenNthCalledWith(1, {
      certificate_source: "file",
      certificate_file: "/secure/release.pfx",
    });
    expect(onChange).toHaveBeenNthCalledWith(2, {
      certificate_source: "file",
      certificate_file: "old.pfx",
      certificate_password_env: "WINDOWS_SIGNING_SECRET_ENV",
    });
    expect(onChange).toHaveBeenNthCalledWith(3, {
      certificate_source: "file",
      certificate_file: "old.pfx",
      timestamp_server: "http://timestamp.sectigo.com",
    });
    expect(onChange).toHaveBeenNthCalledWith(4, {
      certificate_source: "file",
      certificate_file: "old.pfx",
      sign_algorithm: "sha512",
    });
    expect(onChange).toHaveBeenNthCalledWith(5, {
      certificate_source: "file",
      certificate_file: "old.pfx",
      dual_sign: true,
    });
  });

  it("supports certificate-store signing and discovered certificate application", () => {
    const onChange = vi.fn();
    const onApplyDiscovered = vi.fn();
    const discovered = [
      { id: "cert-1", name: "Release certificate" },
    ] as never[];
    render(
      <WindowsSigningForm
        config={{ certificate_source: "store", certificate_thumbprint: "OLD" }}
        onChange={onChange}
        discovered={discovered}
        onApplyDiscovered={onApplyDiscovered}
      />,
    );
    expect(
      screen.queryByLabelText("Certificate File Path"),
    ).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Certificate Thumbprint"), {
      target: { value: "NEW-THUMBPRINT" },
    });
    fireEvent.change(screen.getByLabelText("Certificate Source"), {
      target: { value: "azure_keyvault" },
    });
    fireEvent.change(
      screen.getByRole("combobox", { name: "Discovered certificates" }),
      { target: { value: "cert-1" } },
    );
    expect(onChange).toHaveBeenNthCalledWith(1, {
      certificate_source: "store",
      certificate_thumbprint: "NEW-THUMBPRINT",
    });
    expect(onChange).toHaveBeenNthCalledWith(2, {
      certificate_source: "azure_keyvault",
      certificate_thumbprint: "OLD",
    });
    expect(onApplyDiscovered).toHaveBeenCalledWith(discovered[0]);
  });
});
