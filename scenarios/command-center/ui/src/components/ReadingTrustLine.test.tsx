import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { expectNoA11yViolations } from "../test-utils/a11y";
import { makeReading } from "../test-utils/readings";
import { ReadingTrustLine } from "./ReadingTrustLine";

const observedAt = new Date(Date.now() - 3 * 60_000).toISOString();

describe("ReadingTrustLine accessibility and transitions", () => {
  it("keeps state, reason, age, and retained last-good age together in its accessible name", async () => {
    const reading = makeReading({
      label: "Revenue", coverage: "NOW", trust: "UNAVAILABLE", value: 1200,
      observedAt, trustReason: "producer observation is stale",
    });
    const { container } = renderWithProviders(<ReadingTrustLine reading={reading} />);
    const line = screen.getByRole("group", { name: /Revenue: UNAVAILABLE.*producer observation is stale · last good 3m ago/ });
    expect(line).toHaveTextContent("UNAVAILABLE");
    expect(line.querySelector("[data-qualifier]")).toHaveTextContent("source not answering · last good 3m ago");
    expect(line.querySelector("[data-qualifier]")).toHaveAttribute("title", expect.stringContaining("producer observation is stale"));
    await expectNoA11yViolations(container);
  });

  it("announces a non-degraded live qualifier without adding a repeating live region", () => {
    const reading = makeReading({
      label: "Revenue", coverage: "NOW", trust: "VALID", value: 1200,
      observedAt: new Date().toISOString(),
    });
    const { container } = renderWithProviders(<ReadingTrustLine reading={reading} hideLive />);
    const line = screen.getByRole("group", { name: /Revenue: CURRENT/ });
    expect(line.getAttribute("aria-live")).toBeNull();
    expect(line.querySelector("[data-qualifier]")).toHaveClass("cc-visually-hidden");
    expect(container.querySelector("[data-trust-state]")).toBeNull();
  });

  it("updates one consumer through stale, expired, and recovered readings", () => {
    const current = makeReading({ label: "Launches", coverage: "NOW", trust: "VALID", value: 14, observedAt: new Date().toISOString() });
    const stale = { ...current, trust: "CACHED" as const, observedAt, trustReason: "refresh failed" };
    const expired = { ...stale, trust: "UNAVAILABLE" as const };
    const { rerender } = renderWithProviders(<ReadingTrustLine reading={current} />);

    expect(screen.getByRole("group", { name: /Launches: CURRENT/ })).toBeInTheDocument();
    rerender(<ReadingTrustLine reading={stale} />);
    expect(screen.getByRole("group", { name: /Launches: CACHED.*last good 3m ago · refresh failed/ })).toBeInTheDocument();
    rerender(<ReadingTrustLine reading={expired} />);
    expect(screen.getByRole("group", { name: /Launches: UNAVAILABLE.*refresh failed · last good 3m ago/ })).toBeInTheDocument();
    rerender(<ReadingTrustLine reading={current} />);
    expect(screen.getByRole("group", { name: /Launches: CURRENT/ })).toBeInTheDocument();
  });

  it("keeps authored value-specific copy beside degraded trust context", () => {
    const reading = makeReading({ label: "Paid signups", coverage: "NOW", trust: "CACHED", value: 14, observedAt, trustReason: "refresh failed" });
    const { container } = renderWithProviders(<ReadingTrustLine reading={reading} qualifier={{ text: "paid step", tone: "live" }} />);
    expect(screen.getByRole("group", { name: /Paid signups: CACHED.*paid step.*refresh failed/ })).toBeInTheDocument();
    expect(container.querySelector("[data-qualifier]")).toHaveTextContent("paid step");
    expect(container.querySelector("[data-qualifier]")).toHaveTextContent("refresh failed");
  });

  it.each(["CACHED", "UNAVAILABLE", "UNTRUSTED"] as const)("labels the %s state with its reading-specific reason", async (trust) => {
    const reading = makeReading({
      label: "Launches", coverage: "NOW", trust, value: trust === "UNAVAILABLE" ? 14 : 14,
      observedAt, trustReason: "selector returned an invalid shape",
    });
    const { container } = renderWithProviders(<ReadingTrustLine reading={reading} />);
    expect(screen.getByRole("group", { name: new RegExp(`Launches: ${trust}.*selector returned an invalid shape`) })).toBeInTheDocument();
    expect(container.querySelector("[data-trust-state]")).toHaveTextContent(trust);
    await expectNoA11yViolations(container);
  });

  it("keeps the full untrusted reason accessible while showing compact copy", async () => {
    const reading = makeReading({ label: "Readiness", coverage: "NOW", trust: "UNTRUSTED", value: 0, observedAt: null, trustReason: "producer did not supply observation time" });
    const { container } = renderWithProviders(<ReadingTrustLine reading={reading} />);
    const line = screen.getByRole("group", { name: /Readiness: UNTRUSTED.*producer did not supply observation time/ });
    expect(line.querySelector("[data-qualifier]")).toHaveTextContent("no source time");
    expect(line.querySelector("[data-qualifier]")).toHaveAttribute("title", expect.stringContaining("producer did not supply observation time"));
    await expectNoA11yViolations(container);
  });
});
