import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { AcceptanceCriteriaEditor } from "./acceptance-criteria-editor";
import { selectors } from "../../../consts/selectors";

function Harness({
  initial,
  saved,
  onSave,
  isPending,
}: {
  initial: string;
  saved: string[];
  onSave?: () => void;
  isPending?: boolean;
}) {
  const [value, setValue] = useState(initial);
  return (
    <AcceptanceCriteriaEditor
      value={value}
      saved={saved}
      isPending={isPending ?? false}
      onChange={setValue}
      onSave={onSave ?? (() => {})}
    />
  );
}

describe("AcceptanceCriteriaEditor", () => {
  it("renders the parsed preview as an ordered list", () => {
    render(<Harness initial={"First\nSecond"} saved={["First", "Second"]} />);
    const preview = screen.getByTestId(selectors.initiativeDetails.criteriaPreview);
    expect(preview).toHaveTextContent("First");
    expect(preview).toHaveTextContent("Second");
  });

  it("shows muted placeholder when no criteria are present", () => {
    render(<Harness initial="" saved={[]} />);
    const preview = screen.getByTestId(selectors.initiativeDetails.criteriaPreview);
    expect(preview).toHaveTextContent(/Enter one criterion per line/);
  });

  it("counts parsed criteria correctly", () => {
    render(<Harness initial={"A\nB\nC"} saved={["A", "B", "C"]} />);
    expect(screen.getByTestId(selectors.initiativeDetails.criteriaCount)).toHaveTextContent("3 criterions");
  });

  it("hides the save button when parsed value matches saved", () => {
    render(<Harness initial={"X\nY"} saved={["X", "Y"]} />);
    expect(screen.queryByTestId(selectors.initiativeDetails.criteriaSave)).toBeNull();
  });

  it("shows the save button when parsed value diverges from saved", async () => {
    render(<Harness initial="X" saved={["X"]} />);
    const textarea = screen.getByTestId(selectors.initiativeDetails.criteriaInput);
    await userEvent.type(textarea, "\nY");
    expect(screen.getByTestId(selectors.initiativeDetails.criteriaSave)).toBeInTheDocument();
  });

  it("appends a common-criteria chip line when missing", async () => {
    render(<Harness initial="" saved={[]} />);
    const chips = screen.getAllByTestId(selectors.initiativeDetails.criteriaCommonChip);
    const first = chips[0]!;
    await userEvent.click(first);
    const preview = screen.getByTestId(selectors.initiativeDetails.criteriaPreview);
    expect(preview.textContent).toContain(first.textContent?.replace("+ ", "").trim());
  });

  it("dedupes common-criteria chip clicks (case-insensitive) so the same line never duplicates", async () => {
    render(<Harness initial="all tests pass" saved={[]} />);
    const chip = screen
      .getAllByTestId(selectors.initiativeDetails.criteriaCommonChip)
      .find((el) => el.textContent?.includes("All tests pass"))!;
    await userEvent.click(chip);
    const textarea: HTMLTextAreaElement = screen.getByTestId(selectors.initiativeDetails.criteriaInput);
    expect(textarea.value).toBe("all tests pass");
  });

  it("calls onSave with the in-flight value when Save is clicked", async () => {
    const onSave = vi.fn();
    render(<Harness initial="A" saved={[]} onSave={onSave} />);
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.criteriaSave));
    expect(onSave).toHaveBeenCalled();
  });
});
