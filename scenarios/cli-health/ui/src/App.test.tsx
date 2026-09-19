/**
 * App tests — smoke only.
 *
 * App is a 15-line composition of <AppShell> + per-feature cards.
 * Per-feature behaviour is covered in features/<name>/<Name>Card.test.tsx;
 * shell + locale switching live in components/AppShell.test.tsx. This
 * file's job is to assert the composition: a real <App /> mounts the
 * shell selectors.
 *
 * No per-feature mocks are installed here on purpose: feature cards
 * own their own mock setup in their per-feature tests. If a scenario
 * deletes a feature, this file does not need to change — the smoke is
 * shell-only.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";

import App from "./App";
import { selectors } from "./consts/selectors";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the shell title (smoke: composition wires up)", () => {
    renderWithProviders(<App />);
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });
});
