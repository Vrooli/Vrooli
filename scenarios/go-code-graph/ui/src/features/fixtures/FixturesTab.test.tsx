import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  ListFixturesResponseSchema,
  ValidateFixtureResponseSchema,
} from "@vrooli/proto-types/go-code-graph/v1/graph/graph_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/graph", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/graph")>();
  return {
    ...actual,
    goCodeGraphClient: {
      extract: vi.fn(),
      rewritePlan: vi.fn(),
      rewriteApply: vi.fn(),
      listFixtures: vi.fn(),
      validateFixture: vi.fn(),
    },
  };
});

import { FixturesTab } from "./FixturesTab";
import { goCodeGraphClient } from "../../api/graph";

const client = vi.mocked(goCodeGraphClient);

const listResponse = create(ListFixturesResponseSchema, {
  fixtures: [
    { name: "go-cycles", path: "bas/fixtures/go-cycles", hasExpected: true },
    { name: "go-mislocated", path: "bas/fixtures/go-mislocated", hasExpected: true },
  ],
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("FixturesTab", () => {
  it("lists the fixtures returned by the server", async () => {
    client.listFixtures.mockResolvedValue(listResponse);
    renderWithProviders(<FixturesTab />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.features.fixtures.item({ name: "go-cycles" }))).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.features.fixtures.item({ name: "go-mislocated" }))).toBeInTheDocument();
  });

  it("renders a pass result when validation succeeds", async () => {
    const user = userEvent.setup();
    client.listFixtures.mockResolvedValue(listResponse);
    client.validateFixture.mockResolvedValue(
      create(ValidateFixtureResponseSchema, {
        passed: true,
        graphHash: "abc123def456",
        expectedBytes: 100n,
        actualBytes: 100n,
      }),
    );
    renderWithProviders(<FixturesTab />);
    await waitFor(() => screen.getByTestId(selectors.features.fixtures.item({ name: "go-cycles" })));

    const row = screen.getByTestId(selectors.features.fixtures.item({ name: "go-cycles" }));
    await user.click(within(row).getByRole("button"));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.features.fixtures.result)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.shared.severityBadge.root({ level: "info" }))).toBeInTheDocument();
  });

  it("renders a diff when validation fails", async () => {
    const user = userEvent.setup();
    client.listFixtures.mockResolvedValue(listResponse);
    client.validateFixture.mockResolvedValue(
      create(ValidateFixtureResponseSchema, {
        passed: false,
        diff: "- old line\n+ new line\n",
        graphHash: "abc123def456",
        expectedBytes: 100n,
        actualBytes: 120n,
      }),
    );
    renderWithProviders(<FixturesTab />);
    await waitFor(() => screen.getByTestId(selectors.features.fixtures.item({ name: "go-mislocated" })));

    const row = screen.getByTestId(selectors.features.fixtures.item({ name: "go-mislocated" }));
    await user.click(within(row).getByRole("button"));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.features.fixtures.diff)).toBeInTheDocument();
    });
  });
});
