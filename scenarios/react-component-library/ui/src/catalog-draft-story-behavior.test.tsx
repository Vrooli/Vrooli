import type { ComponentType } from "react";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "./test-utils";
import * as cardStoryModule from "../../library/components/CardShell/versions/1.0.8/story";
import cardStoryRaw from "../../library/components/CardShell/versions/1.0.8/story.json?raw";
import * as drawerStoryModule from "../../library/components/FullPageDrawer/versions/1.4.7/story";
import drawerStoryRaw from "../../library/components/FullPageDrawer/versions/1.4.7/story.json?raw";
import * as attachmentStoryModule from "../../library/components/ComposerAttachmentTray/versions/1.0.4/story";
import attachmentStoryRaw from "../../library/components/ComposerAttachmentTray/versions/1.0.4/story.json?raw";

type StoryRecord = {
  id: string;
  composition?: { specimen?: { module: string; export: string } };
  args?: Record<string, unknown>;
};

type StoryContract = { stories: StoryRecord[] };

function composedStory<TProps>(
  rawContract: string,
  module: object,
  storyID: string,
): { story: StoryRecord; Component: ComponentType<TProps> } {
  const contract = JSON.parse(rawContract) as StoryContract;
  const story = contract.stories.find((candidate) => candidate.id === storyID);
  if (!story) throw new Error(`story contract has no ${storyID} story`);
  const specimen = story.composition?.specimen;
  if (!specimen) throw new Error(`${storyID} story has no specimen composition`);
  expect(specimen.module).toBe("./story.tsx");
  const component = (module as Record<string, unknown>)[specimen.export];
  expect(typeof component).toBe("function");
  return { story, Component: component as ComponentType<TProps> };
}

describe("governed component draft specimens", () => {
  it("renders CardShell empty and overflow args through each story's authored composition", () => {
    const emptySpecimen = composedStory<{ args: { children?: string }; log: () => void }>(
      cardStoryRaw,
      cardStoryModule,
      "empty",
    );
    const EmptyStory = emptySpecimen.Component;
    const empty = renderWithProviders(
      <EmptyStory args={emptySpecimen.story.args as { children?: string }} log={() => undefined} />,
    );
    const emptyShell = screen.getByTestId("data-display.card-shell");
    expect(emptyShell.querySelector("[data-testid='card-shell-story-content']:empty")).toBeInTheDocument();
    expect(emptyShell).toBeVisible();

    empty.unmount();
    const overflowText = "This card contains deliberately long content so its shell demonstrates wrapping and keeps its hierarchy readable.";
    const overflowSpecimen = composedStory<{ args: { children?: string }; log: () => void }>(
      cardStoryRaw,
      cardStoryModule,
      "overflow",
    );
    const OverflowStory = overflowSpecimen.Component;
    expect(overflowSpecimen.story.args?.children).toBe(overflowText);
    renderWithProviders(
      <OverflowStory args={overflowSpecimen.story.args as { children?: string }} log={() => undefined} />,
    );
    expect(screen.getByTestId("card-shell-story-content")).toHaveTextContent(overflowText);
  });

  it("resolves and opens the declared empty, loading, error, and overflow drawer specimens", async () => {
    const renderSpecimen = (storyID: string) =>
      composedStory<Record<string, never>>(drawerStoryRaw, drawerStoryModule, storyID);

    const EmptyStory = renderSpecimen("empty").Component;
    const empty = renderWithProviders(<EmptyStory />);
    expect(screen.getByRole("dialog", { name: "Empty workspace" })).toBeVisible();
    expect(screen.getByTestId("overlays.full-page-drawer").querySelector(".empty-drawer-content"))
      .toBeEmptyDOMElement();
    empty.unmount();

    const LoadingStory = renderSpecimen("loading").Component;
    const loading = renderWithProviders(<LoadingStory />);
    expect(screen.getByRole("dialog", { name: "Loading workspace" })).toBeVisible();
    expect(screen.getByRole("status")).toHaveTextContent("Loading workspace data…");
    loading.unmount();

    const ErrorStory = renderSpecimen("error").Component;
    const error = renderWithProviders(<ErrorStory />);
    expect(screen.getByRole("dialog", { name: "Workspace unavailable" })).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent("Workspace could not be loaded.");
    error.unmount();

    const OverflowStory = renderSpecimen("overflow").Component;
    renderWithProviders(<OverflowStory />);
    expect(screen.getByRole("dialog", { name: "Long workspace" })).toBeVisible();
    expect(screen.getByTestId("drawer-overflow-story-body")).toHaveTextContent(
      "Overflow content remains inside the drawer's scroll region.",
    );
    expect(screen.getByTestId("drawer-overflow-story-body")).toHaveStyle({ blockSize: "1600px" });
  });

  it("renders ComposerAttachmentTray boundary stories from their declared specimen and item args", () => {
    const storyIDs = ["disabled", "one", "many", "overflow"] as const;
    const expectedCounts = { disabled: 1, one: 1, many: 3, overflow: 1 };
    for (const storyID of storyIDs) {
      const specimen = composedStory<{ args: Record<string, unknown>; log: () => void }>(
        attachmentStoryRaw,
        attachmentStoryModule,
        storyID,
      );
      expect(specimen.story.args?.items).toHaveLength(expectedCounts[storyID]);
      const TrayStory = specimen.Component;
      const rendered = renderWithProviders(
        <TrayStory args={specimen.story.args ?? {}} log={() => undefined} />,
      );
      const tray = screen.getByRole("group", { name: "Attachments" });
      expect(tray).toBeVisible();
      expect(tray.querySelectorAll("[data-rcl-composer-attachment]")).toHaveLength(expectedCounts[storyID]);
      if (storyID === "disabled") {
        expect(specimen.story.args?.disabled).toBe(true);
        expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
      }
      if (storyID === "one") {
        expect(screen.getByRole("button", { name: "Remove" })).toBeVisible();
      }
      if (storyID === "overflow") {
        expect(tray).toHaveTextContent("release-evidence-with-a-deliberately-long-filename-that-must-wrap.pdf");
      }
      rendered.unmount();
    }
  });

});
