import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { SessionSectionTabs } from "./SessionSectionTabs";

describe("SessionSectionTabs layout", () => {
  it("bounds a tall panel inside a fixed-height parent", () => {
    render(
      <div className="flex h-96 min-h-0 flex-col">
        <SessionSectionTabs
          sections={[{
            value: "conversation",
            label: "Conversation",
            content: <div data-testid="tall-content" className="h-[1000px]">Tall content</div>,
          }]}
          activeValue="conversation"
          onValueChange={() => undefined}
          listLabel="Sections"
          contentClassName="overflow-y-auto"
        />
      </div>,
    );

    const panel = screen.getByTestId("tall-content").parentElement;
    expect(panel).toHaveClass("min-h-0", "flex-1", "overflow-y-auto");
  });
});
