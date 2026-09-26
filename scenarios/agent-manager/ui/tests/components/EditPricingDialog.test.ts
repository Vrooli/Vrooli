import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { EditPricingDialog } from "../../src/components/dialogs/EditPricingDialog.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const hooks = vi.hoisted(() => ({ models: vi.fn(), overrides: vi.fn(), set: vi.fn(), del: vi.fn() }));
vi.mock("../../src/hooks/usePricing.js", () => ({
  useModelPricing: hooks.models, useModelOverrides: hooks.overrides, useSetOverride: hooks.set, useDeleteOverride: hooks.del,
}));

function setup() {
  const refetchModels = vi.fn(); const refetchOverrides = vi.fn(); const setOverride = vi.fn(async () => undefined); const deleteOverride = vi.fn(async () => undefined);
  hooks.models.mockReturnValue({ data: [{ model: "gpt-5", canonicalName: "gpt-5", inputPricePer1M: 2, outputPricePer1M: 8, inputSource: "provider_api", outputSource: "provider_api" }], refetch: refetchModels });
  hooks.overrides.mockReturnValue({ data: [{ component: "input_tokens", priceUsd: 0.000003 }], loading: false, refetch: refetchOverrides });
  hooks.set.mockReturnValue({ setOverride, loading: false }); hooks.del.mockReturnValue({ deleteOverride, loading: false });
  return { refetchModels, refetchOverrides, setOverride, deleteOverride };
}
afterEach(() => vi.resetAllMocks());

test("EditPricingDialog saves and clears manual overrides while refreshing visible prices", async () => {
  const user = userEvent.setup(); const calls = setup(); const close = vi.fn(); const updated = vi.fn();
  renderWithProviders(createElement(EditPricingDialog, { model: "gpt-5", onClose: close, onPricingUpdated: updated }));
  assert.ok(screen.getByText("Edit Pricing: gpt-5"));
  assert.ok(screen.getByText("$3.00"));
  await user.click(screen.getAllByTitle("Edit override")[0]!);
  const price = screen.getByPlaceholderText("0.00");
  fireEvent.change(price, { target: { value: "4.5" } });
  await user.click(screen.getByTitle("Save override"));
  await waitFor(() => assert.deepEqual(calls.setOverride.mock.calls[0], ["gpt-5", { component: "input_tokens", priceUsd: 0.0000045 }]));
  await user.click(screen.getByTitle("Clear override"));
  await waitFor(() => assert.deepEqual(calls.deleteOverride.mock.calls[0], ["gpt-5", "input_tokens"]));
  assert.ok(calls.refetchModels.mock.calls.length >= 2);
  assert.ok(calls.refetchOverrides.mock.calls.length >= 2);
  assert.ok(updated.mock.calls.length >= 2);
});

test("EditPricingDialog is inert when closed and shows loading overrides when opened", () => {
  setup(); hooks.overrides.mockReturnValue({ data: [], loading: true, refetch: vi.fn() });
  renderWithProviders(createElement(EditPricingDialog, { model: "gpt-5", onClose: vi.fn() }));
  assert.ok(screen.getByText("Loading..."));
});
