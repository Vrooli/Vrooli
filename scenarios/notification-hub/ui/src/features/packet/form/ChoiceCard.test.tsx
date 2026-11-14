import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../../test-utils";
import { ChoiceCard } from "./ChoiceCard";

// Requester-authored option text can be in a different script than the page.
// Each text run must resolve its own direction, so punctuation stays with the
// words it belongs to.
describe("ChoiceCard bidi isolation", () => {
  const label = "Approve (recommended).";
  const detail = "Ship it, then watch.";
  const arabicLabel = "الموافقة (موصى بها).";

  it("lets the label and the detail resolve their own direction", () => {
    renderWithProviders(
      <div dir="rtl">
        <ChoiceCard type="radio" name="q" value="a" label={label} detail={detail} onChange={() => undefined} />
      </div>,
    );

    expect(screen.getByText(label)).toHaveAttribute("dir", "auto");
    expect(screen.getByText(detail)).toHaveAttribute("dir", "auto");
  });

  it("works the same for a right-to-left option on a left-to-right page", () => {
    renderWithProviders(<ChoiceCard type="checkbox" name="q" value="b" label={arabicLabel} onChange={() => undefined} />);

    expect(screen.getByText(arabicLabel)).toHaveAttribute("dir", "auto");
  });
});
