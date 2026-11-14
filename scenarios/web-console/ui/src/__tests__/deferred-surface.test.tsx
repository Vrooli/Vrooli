import { lazy, useState } from "react";
import { act, fireEvent, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../test-utils";
import DeferredSurface from "../components/DeferredSurface";

it("downloads a surface on demand, shows progress, then preserves its state through closing", async () => {
  let resolve!: (module: { default: typeof Surface }) => void;
  function Surface({ open }: { open: boolean }) {
    const [draft, setDraft] = useState("");
    return <input aria-label="Deferred draft" hidden={!open} value={draft} onChange={event => { setDraft(event.target.value); }} />;
  }
  const load = vi.fn(() => new Promise<{ default: typeof Surface }>(done => { resolve = done; }));
  const LazySurface = lazy(load);
  const view = render(<DeferredSurface active={false}><LazySurface open={false} /></DeferredSurface>);
  expect(load).not.toHaveBeenCalled();
  view.rerender(<DeferredSurface active><LazySurface open /></DeferredSurface>);
  expect(load).toHaveBeenCalledOnce();
  expect(screen.getByRole("status")).toBeInTheDocument();
  await act(async () => { resolve({ default: Surface }); await Promise.resolve(); });
  fireEvent.change(screen.getByLabelText("Deferred draft"), { target: { value: "Unsent draft" } });
  view.rerender(<DeferredSurface active={false}><LazySurface open={false} /></DeferredSurface>);
  expect(screen.getByLabelText("Deferred draft")).toHaveValue("Unsent draft");
  expect(screen.getByLabelText("Deferred draft")).not.toBeVisible();
  view.rerender(<DeferredSurface active><LazySurface open /></DeferredSurface>);
  expect(screen.getByLabelText("Deferred draft")).toHaveValue("Unsent draft");
  expect(load).toHaveBeenCalledOnce();
});
