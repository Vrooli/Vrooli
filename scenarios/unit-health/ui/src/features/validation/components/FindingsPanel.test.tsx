import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent } from "@testing-library/react";
import { afterEach, beforeEach, expect, it } from "vitest";
import { ValidationFindingSchema } from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";
import { renderWithProviders } from "../../../test-utils";
import { FindingsPanel } from "./FindingsPanel";
import { setLocale } from "../../../i18n";

beforeEach(async () => { await setLocale("en"); });
afterEach(cleanup);

it("retains suppressed totals, original severity, and accountable exception details", () => {
  const findings = Array.from({ length: 51 }, (_, i) => create(ValidationFindingSchema, {
    id: `finding-${i}`, code: `RULE-${i}`, severity: "error", message: `original finding ${i}`,
    suppressionReasons: [{ reason: "migration", owner: "test-team", evidence: "record-1", expiresAt: "2027-01-01", revisit: "release" }],
  }));
  const view = renderWithProviders(<FindingsPanel findings={findings} suppressed />);
  expect(view.getByText("Suppressed findings")).toBeVisible();
  expect(view.container).toHaveTextContent("They do not resolve the findings or prove passing tests.");
  expect(view.container.querySelectorAll("article")).toHaveLength(50);
  expect(view.container).toHaveTextContent("51");
  fireEvent.click(view.getAllByText("Exception reasons and ownership")[0]!);
  expect(view.getAllByText(/"owner": "test-team"/)[0]).toBeVisible();
  fireEvent.click(view.getByRole("button"));
  expect(view.container.querySelectorAll("article")).toHaveLength(51);
  expect(view.getByText("original finding 50")).toBeVisible();
  expect(view.getAllByText("error")).toHaveLength(51);
});

it("does not invent historical exception details", () => {
  const view = renderWithProviders(<FindingsPanel findings={[create(ValidationFindingSchema, { id: "old", code: "OLD" })]} suppressed />);
  fireEvent.click(view.getByText("Exception reasons and ownership"));
  expect(view.getByText(/Exception details are unknown/)).toBeVisible();
});
