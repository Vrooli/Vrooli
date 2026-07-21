import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, within } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { RelatedTab } from "./RelatedTab";
import { dynamicSelectors, selectors } from "../../consts/selectors";

const { getRelated } = vi.hoisted(() => ({ getRelated: vi.fn() }));
vi.mock("../../services/related-service", () => ({ relatedService: { getRelated } }));

describe("RelatedTab", () => {
  beforeEach(() => {
    getRelated.mockReset();
    window.localStorage.clear();
  });

  it("renders fixed groups, explanations, deep links, and semantic degradation", async () => {
    getRelated.mockResolvedValue({ groups: [
      { name: "linked", entities: [{ entityKind: "backlog", key: "idea/linked", title: "Linked", status: "backlog", archived: false, reasons: ["depends on this item"] }], degraded: false },
      { name: "same_scope", entities: [], degraded: false },
      { name: "similar", entities: [], degraded: true },
    ] });
    renderWithProviders(<RelatedTab target={{ kind: "backlog", backlogKind: "idea", name: "source" }} enabled />, { initialEntries: ["/backlog/idea/source?tab=related"] });
    expect(await screen.findByTestId(selectors.related.groupLinked)).toBeInTheDocument();
    expect(screen.getByText("depends on this item")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Linked" })).toHaveAttribute("href", "/backlog/idea/linked");
    expect(within(screen.getByTestId(dynamicSelectors.related.rowByEntity({ entity: "backlog", id: "idea/linked" }))).getByText("backlog")).toHaveClass("bg-cyan-500/15");
    expect(screen.getByTestId(selectors.related.groupSameScope)).toHaveTextContent("No related work.");
    expect(screen.getByRole("button", { name: /similar \(0\).*similarity unavailable/i })).toBeInTheDocument();
    expect(getRelated).toHaveBeenCalledWith(
      { kind: "backlog", backlogKind: "idea", name: "source" },
      { excludeHistorical: false, entityKinds: [] },
    );
    fireEvent.click(screen.getByRole("button", { name: /linked \(1\)/i }));
    expect(screen.queryByTestId(selectors.related.groupLinked)).not.toBeInTheDocument();
  });

  it("uses the shared loading state and does not retry a missing server route", async () => {
    getRelated.mockRejectedValueOnce(new Error("route unavailable"));
    renderWithProviders(<RelatedTab target={{ kind: "backlog", backlogKind: "idea", name: "source" }} enabled />, { initialEntries: ["/backlog/idea/source?tab=related"] });
    expect(await screen.findByText("Related work is not available yet")).toBeInTheDocument();
    expect(getRelated).toHaveBeenCalledTimes(1);
  });
});
