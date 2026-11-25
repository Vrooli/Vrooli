import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const { fetchMock } = vi.hoisted(() => ({ fetchMock: vi.fn() }));

vi.mock("./client", async (importOriginal) => {
  const original = await importOriginal<typeof import("./client")>();
  const { createScenarioConnectTransport } = await vi.importActual<typeof import("@vrooli/api-base")>("@vrooli/api-base");
  return {
    ...original,
    transport: createScenarioConnectTransport({
      baseUrl: "http://search-hub.test/api/v1",
      fetch: fetchMock,
    }),
  };
});

import { strings } from "../consts/strings";
import { QueryPartEvidence_Source } from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";
import { renderWithProviders } from "../test-utils";
import { SearchResults } from "../features/search/SearchResults";
import { runQuery } from "./search";

describe("Search Hub Connect query consumer", () => {
  it("decodes an API-shaped task response through Connect and renders evidence-bound parts", async () => {
    const fixture = readFileSync(
      resolve(process.cwd(), "../api/handlers/routing/testdata/query-task-response.json"),
      "utf8",
    );
    fetchMock.mockResolvedValueOnce(new Response(fixture, {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));

    const response = await runQuery({ query: "HealthCard route and owner", types: ["doc"], all: false });
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(response.queryParts[0]?.source).toBe(QueryPartEvidence_Source.USER_QUERY);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://search-hub.test/api/v1/vrooli.search_hub.v1.routing.RoutingService/Query");
    const requestBytes = init.body as Uint8Array;
    expect(JSON.parse(new TextDecoder().decode(requestBytes))).toMatchObject({
      query: "HealthCard route and owner",
      types: ["doc"],
      explain: true,
    });

    renderWithProviders(<SearchResults data={response} query="HealthCard route and owner" />);

    expect(screen.getByText(strings.search.taskHeading)).toBeInTheDocument();
    expect(screen.getByText("Find the relevant interface")).toBeInTheDocument();
    expect(screen.getByText("HealthCard component")).toBeInTheDocument();
    expect(screen.getByText(/search\.unknownRetrievalSource/)).toBeInTheDocument();
    expect(screen.getByText("Find routing guidance")).toBeInTheDocument();
    expect(screen.getByText(/search\.supportUnavailable/)).toBeInTheDocument();
    expect(screen.getByText("Identify the owning capability")).toBeInTheDocument();
    expect(screen.getByText(/search\.partNotRouted/)).toBeInTheDocument();
    expect(screen.getByText("Preserve a future coverage state")).toBeInTheDocument();
    expect(screen.getByText(/search\.partStatusUnknownValue/)).toBeInTheDocument();
    expect(screen.getByText("Preserve a future request-part source")).toBeInTheDocument();
    expect(screen.getByText(/search\.partSourceUnknownValue/)).toBeInTheDocument();
  });
});
