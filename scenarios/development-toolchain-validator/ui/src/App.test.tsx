/**
 * App composition smoke test.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { selectors } from "./consts/selectors";
import { makeApiMocks, renderWithProviders } from "./test-utils";

vi.mock("./api/golden", async (importOriginal) => {
  const { create } = await import("@bufbuild/protobuf");
  const { ListGoldensResponseSchema } = await import(
    "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb"
  );
  const actual = await importOriginal<typeof import("./api/golden")>();
  return {
    ...actual,
    goldenClient: {
      listGoldens: vi.fn().mockResolvedValue(create(ListGoldensResponseSchema, { goldens: [] })),
      getGolden: vi.fn(),
      registerGolden: vi.fn(),
      regenerateGolden: vi.fn(),
      deleteGolden: vi.fn(),
    },
  };
});

vi.mock("./api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/health")>();
  return { ...actual, ...makeApiMocks() };
});

import { AppRoutes } from "./App";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the AppShell and resolves the default route to the goldens index", async () => {
    renderWithProviders(<AppRoutes />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.nav.appShell)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.goldens.indexHeading)).toBeInTheDocument();
  });
});
