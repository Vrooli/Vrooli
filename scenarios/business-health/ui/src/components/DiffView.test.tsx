import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";

import { i18n } from "../i18n";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { DiffView } from "./DiffView";

const renderDiff = (before: string, after: string) =>
  render(
    <I18nextProvider i18n={i18n}>
      <DiffView before={before} after={after} data-testid={selectors.wizard.preview} />
    </I18nextProvider>,
  );

describe("DiffView", () => {
  afterEach(() => cleanup());

  it("flags a new file when before is empty", () => {
    renderDiff("", "hello\nworld");
    expect(screen.getByText(strings.diff.newFile)).toBeInTheDocument();
  });

  it("shows the no-changes state for identical text", () => {
    renderDiff("same", "same");
    expect(screen.getByText(strings.diff.empty)).toBeInTheDocument();
  });

  it("renders under the provided test id", () => {
    renderDiff("a", "b");
    expect(screen.getByTestId(selectors.wizard.preview)).toBeInTheDocument();
  });
});
