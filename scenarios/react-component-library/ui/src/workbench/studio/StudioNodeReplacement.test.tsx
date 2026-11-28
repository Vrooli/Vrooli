import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { AssetKind } from "@vrooli/proto-types/react-component-library/v1/components/components_pb";
import type { CatalogAsset } from "../data";
import { node } from "../templates/spec";
import { StudioNodeReplacement } from "./StudioNodeReplacement";
import {
  asCatalogReplacementNode,
  asReplacementNode,
  rankCatalogCandidates,
  rankReplacementCandidates,
} from "./nodeReplacement";

const catalog = [
  {
    id: "asset-plant-card",
    libraryId: "plants.care-card",
    displayName: "Plant Care Card",
    description: "Shows watering progress and plant health status.",
    tags: ["plant", "water", "progress"],
    assetKind: AssetKind.COMPONENT,
  },
  {
    id: "asset-audio-player",
    libraryId: "audio.player",
    displayName: "Audio Player",
    description: "Play or pause a sound.",
    tags: ["media"],
    assetKind: AssetKind.COMPONENT,
  },
] as CatalogAsset[];

describe("StudioNodeReplacement", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  it("ranks a semantic action layer toward the RCL button", () => {
    const selected = node("confirm", "text", { label: "Confirm purchase", text: "Pay now" });
    expect(rankReplacementCandidates(selected)[0]?.type).toBe("button");
  });

  it("ranks child-bearing selections toward containers and preserves explicit replacements", () => {
    const selected = node("section", "surface", { label: "Profile" }, [
      node("name", "text", { text: "Ari" }),
    ]);
    expect(
      rankReplacementCandidates(selected).some((candidate) => candidate.type === "button"),
    ).toBe(false);
    const replacedWithButton = asReplacementNode(selected, "button");
    expect(replacedWithButton.type).toBe("button");
    expect(replacedWithButton.children).toHaveLength(1);
    expect(asReplacementNode(selected, "card").children).toHaveLength(1);
  });

  it("offers Button for an explicit search even when the selected region is generic", () => {
    const selected = node("section", "surface", { label: "Profile" }, [
      node("name", "text", { text: "Ari" }),
    ]);
    expect(
      rankReplacementCandidates(selected, "button").map((candidate) => candidate.type),
    ).toContain("button");
  });

  it("does not offer or apply replacements to the required frame root", () => {
    const root = node("page", "page", {}, [node("heading", "text", { text: "Overview" })]);
    expect(rankReplacementCandidates(root)).toEqual([]);
    expect(asReplacementNode(root, "card")).toBe(root);
  });

  it("searches candidate names and exposes explicit replacement actions", () => {
    const onReplace = vi.fn();
    renderWithProviders(
      <StudioNodeReplacement
        node={node("copy", "text", { text: "Welcome" })}
        onReplace={onReplace}
      />,
    );

    fireEvent.change(screen.getByRole("textbox", { name: "Search RCL building blocks" }), {
      target: { value: "progress" },
    });
    expect(screen.getByRole("button", { name: "Replace with Meter" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Replace with Button" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Replace with Meter" }));
    expect(onReplace).toHaveBeenCalledWith("meter");
  });

  it("preserves selected identity and maps text into native input content", () => {
    const replaced = asReplacementNode(
      node("search", "text", { text: "Find a page", align: "left" }),
      "input",
    );
    expect(replaced).toMatchObject({
      id: "search",
      type: "input",
      props: { text: "Find a page", placeholder: "Find a page", align: "left" },
    });
  });

  it("ranks catalog assets by the selected layer's content and tags", () => {
    const selected = node("plant-status", "stack", { label: "Overview" }, [
      node("copy", "text", { text: "Plant watering progress" }),
    ]);
    expect(rankCatalogCandidates(selected, catalog)[0]?.id).toBe("asset-plant-card");
    expect(
      rankCatalogCandidates(selected, catalog, "audio").map((candidate) => candidate.id),
    ).toEqual(["asset-audio-player"]);
  });

  it("does not use imported CSS or SVG data as semantic match text", () => {
    const selected = node(
      "decorative-panel",
      "surface",
      {
        label: "Decorative panel",
        css: '{"backgroundImage":"url(data:image/svg+xml;base64,audio-player)","--theme":"Plant care"}',
      },
      [node("landscape", "image", { src: "data:image/svg+xml;base64,plant-watering-progress" })],
    );
    const ranked = rankCatalogCandidates(selected, catalog);
    expect(ranked.find((candidate) => candidate.id === "asset-plant-card")?.score).toBe(0);
    expect(ranked.find((candidate) => candidate.id === "asset-audio-player")?.score).toBe(0);
  });

  it("replaces a selected region with a live component slot and keeps its fallback tree", () => {
    const selected = node(
      "plant-status",
      "stack",
      { label: "Plant status", gap: 12, css: '{"background":"#123456","padding":"20px"}' },
      [node("copy", "text", { text: "Next watering" })],
    );
    expect(asCatalogReplacementNode(selected, "asset-plant-card", "plant-default")).toMatchObject({
      id: "plant-status",
      type: "slot",
      props: {
        gap: 12,
        css: '{"background":"#123456","padding":"20px"}',
        hint: "Plant status",
        componentId: "asset-plant-card",
        storyId: "plant-default",
      },
      children: [{ id: "copy", type: "text" }],
    });
  });

  it("offers real catalog matches beside native RCL building blocks", () => {
    const onReplaceCatalogAsset = vi.fn();
    renderWithProviders(
      <StudioNodeReplacement
        node={node("plant-status", "text", { text: "Plant watering progress" })}
        onReplace={vi.fn()}
        catalogCandidates={catalog}
        onReplaceCatalogAsset={onReplaceCatalogAsset}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Replace with Plant Care Card component" }));
    expect(onReplaceCatalogAsset).toHaveBeenCalledWith("asset-plant-card");
  });
});
