import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { strings } from "../../consts/strings";
import { markSensitive } from "../../lib/sensitiveMedia";
import { renderWithProviders } from "../../test-utils";
import { SETTINGS_STORAGE_KEY, type SensitiveImages } from "../settings/useSettings";
import { SensitiveImage } from "./SensitiveImage";

// renderWithProviders mounts the real SettingsProvider, which reads the stored
// preference, so the operator's choice is set the way the page sets it.
const renderImage = (ui: React.ReactElement, sensitiveImages: SensitiveImages = "blur") => {
  window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify({ sensitiveImages }));
  return renderWithProviders(ui);
};

// Alt text is passed through unchanged, so the queries use the same constants.
const ALT = { plain: "plain", result: "result", shown: "shown" } as const;

beforeEach(() => window.localStorage.clear());
afterEach(cleanup);

describe("SensitiveImage", () => {
  it("renders a bare image when nothing flags it", () => {
    const { container } = renderImage(<SensitiveImage src="blob:plain" alt={ALT.plain} className="x" />);
    expect(container.querySelector("[data-sensitive]")).toBeNull();
    expect(screen.getByAltText(ALT.plain)).toHaveAttribute("src", "blob:plain");
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("blurs a flagged image behind a tap-to-reveal and hides it again", async () => {
    const user = userEvent.setup();
    const { container } = renderImage(<SensitiveImage src="blob:a" sensitive alt={ALT.result} />);
    const wrapper = container.querySelector("[data-sensitive]");
    expect(wrapper).toHaveAttribute("data-sensitive", "blurred");
    expect(container.querySelector(".blur-2xl")).not.toBeNull();
    // The blurred image carries no alt text: its content is not announced.
    expect(container.querySelector("img")).toHaveAttribute("alt", "");

    await user.click(screen.getByRole("button", { name: strings.sensitive.reveal }));
    expect(wrapper).toHaveAttribute("data-sensitive", "revealed");
    expect(container.querySelector(".blur-2xl")).toBeNull();
    expect(screen.getByAltText(ALT.result)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: strings.sensitive.hide }));
    expect(wrapper).toHaveAttribute("data-sensitive", "blurred");
  });

  it("looks the flag up by object url when the caller does not pass one", () => {
    markSensitive({ url: "blob:marked" });
    const { container } = renderImage(<SensitiveImage src="blob:marked" alt="m" />);
    expect(container.querySelector("[data-sensitive]")).toHaveAttribute("data-sensitive", "blurred");
  });

  it("blurs by default when no settings provider is mounted", () => {
    const { container } = render(<SensitiveImage src="blob:b" sensitive alt="b" />);
    expect(container.querySelector("[data-sensitive]")).toHaveAttribute("data-sensitive", "blurred");
  });

  it("shows a flagged image plainly when the operator chose Show", () => {
    const { container } = renderImage(<SensitiveImage src="blob:c" sensitive alt={ALT.shown} />, "show");
    expect(container.querySelector("[data-sensitive]")).toBeNull();
    expect(screen.getByAltText(ALT.shown)).toBeInTheDocument();
  });

  it("re-blurs when the source changes so a new result is never shown revealed", async () => {
    const user = userEvent.setup();
    const { container, rerender } = renderImage(<SensitiveImage src="blob:one" sensitive alt="one" />);
    await user.click(screen.getByRole("button", { name: strings.sensitive.reveal }));
    expect(container.querySelector("[data-sensitive]")).toHaveAttribute("data-sensitive", "revealed");
    rerender(<SensitiveImage src="blob:two" sensitive alt="two" />);
    expect(container.querySelector("[data-sensitive]")).toHaveAttribute("data-sensitive", "blurred");
  });

  it("stays blurred without a control when it sits inside another control", () => {
    renderImage(
      <button type="button">
        <SensitiveImage src="blob:d" sensitive interactive={false} alt="d" />
      </button>,
    );
    expect(screen.getAllByRole("button")).toHaveLength(1);
    expect(document.querySelector("[data-sensitive]")).toHaveAttribute("data-sensitive", "blurred");
  });
});
