import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { LinuxSigningForm } from "./LinuxSigningForm";

describe("LinuxSigningForm", () => {
  it("enables a clean GPG signing configuration and can disable it", () => {
    const onChange = vi.fn();
    const { rerender } = render(<LinuxSigningForm onChange={onChange} />);
    expect(
      screen.getByText(/Enable Linux signing to configure GPG key settings/),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Configure"));
    expect(onChange).toHaveBeenCalledWith({ gpg_key_id: "" });

    rerender(
      <LinuxSigningForm config={{ gpg_key_id: "key" }} onChange={onChange} />,
    );
    fireEvent.click(screen.getByLabelText("Configure"));
    expect(onChange).toHaveBeenLastCalledWith(undefined);
  });

  it("edits GPG fields, applies discovered keys, and presents generation feedback", () => {
    const onChange = vi.fn();
    const onApplyDiscovered = vi.fn();
    const onGenerate = vi.fn();
    const discovered = [
      {
        id: "gpg-1",
        name: "Release key",
        days_to_expiry: 20,
        is_expired: false,
      },
    ] as never[];
    render(
      <LinuxSigningForm
        config={{ gpg_key_id: "existing" }}
        onChange={onChange}
        discovered={discovered}
        onApplyDiscovered={onApplyDiscovered}
        onGenerate={onGenerate}
        generationMessage="GPG key created"
      />,
    );
    fireEvent.change(screen.getByLabelText("GPG Key ID"), {
      target: { value: "ABC123" },
    });
    fireEvent.change(screen.getByLabelText(/Passphrase Environment Variable/), {
      target: { value: "GPG_SECRET" },
    });
    fireEvent.change(screen.getByLabelText(/GPG Home Directory/), {
      target: { value: "/secure/gnupg" },
    });
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: "gpg-1" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Generate GPG key" }));
    expect(onChange).toHaveBeenNthCalledWith(1, { gpg_key_id: "ABC123" });
    expect(onChange).toHaveBeenNthCalledWith(2, {
      gpg_key_id: "existing",
      gpg_passphrase_env: "GPG_SECRET",
    });
    expect(onChange).toHaveBeenNthCalledWith(3, {
      gpg_key_id: "existing",
      gpg_homedir: "/secure/gnupg",
    });
    expect(onApplyDiscovered).toHaveBeenCalledWith(discovered[0]);
    expect(onGenerate).toHaveBeenCalledOnce();
    expect(screen.getByText("GPG key created")).toBeInTheDocument();
  });

  it("disables key generation while it is in progress", () => {
    render(
      <LinuxSigningForm
        config={{ gpg_key_id: "key" }}
        onChange={vi.fn()}
        onGenerate={vi.fn()}
        generating
      />,
    );
    expect(screen.getByRole("button", { name: "Generating…" })).toBeDisabled();
  });
});
