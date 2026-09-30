// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { EmbeddedScenarioFrame } from "./EmbeddedScenarioFrame";
import { strings } from "../../consts/strings";

const scenarioTitle = "Scenario";
const chatDraftLabel = "Chat draft";

it("keeps sibling chat usable when an embedded scenario is unavailable", () => { // EMB-05
  render(<><EmbeddedScenarioFrame src="javascript:scenario-crash" title={scenarioTitle} /><input aria-label={chatDraftLabel} /></>);
  expect(screen.getByRole("alert")).toHaveTextContent(strings.companion.embeddedUnavailable);
  expect(screen.getByLabelText(chatDraftLabel)).toBeInTheDocument();
});

it("renders a sandbox with no native privilege surface", () => {
  render(<EmbeddedScenarioFrame src="https://scenario.example.test" title={scenarioTitle} />);
  const frame = screen.getByTitle(scenarioTitle);
  expect(frame).toHaveAttribute("sandbox", "allow-forms allow-scripts allow-same-origin");
  expect(frame).toHaveAttribute("referrerpolicy", "no-referrer");
  expect(frame).not.toHaveAttribute("allow");
});


it("rejects credential-bearing or unsupported frame sources", () => {
  render(<EmbeddedScenarioFrame src="javascript:alert(1)" title="Unsafe scenario" />);
  expect(screen.getByRole("alert")).toHaveTextContent(strings.companion.embeddedUnavailable);
});

it("contains a failed loaded frame while preserving the sibling workspace", () => { // EMB-05
  render(<><EmbeddedScenarioFrame src="https://scenario.example.test" title={scenarioTitle} /><input aria-label={chatDraftLabel} /></>);
  fireEvent.error(screen.getByTitle(scenarioTitle));
  expect(screen.getByRole("alert")).toHaveTextContent(strings.companion.embeddedUnavailable);
  expect(screen.getByRole("textbox", { name: chatDraftLabel })).toBeInTheDocument();
});
