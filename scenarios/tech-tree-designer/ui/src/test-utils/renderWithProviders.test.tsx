import { useQueryClient } from "@tanstack/react-query";
import { cleanup } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { renderWithProviders } from "./index";

afterEach(cleanup);

it("exposes the isolated client returned by the canonical render helper", () => {
  let observed: ReturnType<typeof useQueryClient> | undefined;
  function Probe() {
    observed = useQueryClient();
    return null;
  }
  const first = renderWithProviders(<Probe />);
  expect(observed).toBe(first.queryClient);
  first.queryClient.setQueryData(["isolation-probe"], "old render");
  first.unmount();
  const second = renderWithProviders(<Probe />);
  expect(observed).toBe(second.queryClient);
  expect(second.queryClient).not.toBe(first.queryClient);
  expect(second.queryClient.getQueryData(["isolation-probe"])).toBeUndefined();
  expect(second.queryClient.getDefaultOptions().queries?.retry).toBe(false);
});
