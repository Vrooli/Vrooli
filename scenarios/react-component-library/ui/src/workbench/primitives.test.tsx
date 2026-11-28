import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { describe, expect, it, vi } from "vitest";

import {
  Banner,
  Chip,
  Dot,
  Empty,
  Field,
  Kpi,
  Meter,
  Panel,
  PanelBody,
  PanelHead,
  Ring,
  SearchBox,
  Segmented,
  Select,
  Skeleton,
  Tabs,
  TextArea,
  TextInput,
  Toggle,
  WbButton,
} from "./primitives";

function byText(value: string) {
  return screen.getByText((content) => content === value);
}

describe("field association", () => {
  const label = "Layer name";

  it("labels a control that has no id of its own", () => {
    renderWithProviders(
      <Field label={label}>
        <TextInput />
      </Field>,
    );
    expect(screen.getByLabelText(label)).toBeInstanceOf(HTMLInputElement);
  });

  it("keeps an id the control already declares", () => {
    renderWithProviders(
      <Field label={label}>
        <TextInput id="custom-id" />
      </Field>,
    );
    expect(screen.getByLabelText(label).id).toBe("custom-id");
  });

  it("renders common workbench status and layout primitives", () => {
    const buttonName = "Save changes";
    const panelTitle = "Review status";
    const emptyTitle = "Nothing here";
    const alertMessage = "Action required";
    const pendingLabel = "Pending";
    const warningLabel = "Warning";
    const neutralLabel = "Neutral";
    const headsUp = "Heads up";
    renderWithProviders(
      <>
        <WbButton variant="primary" size="sm" className="custom-action">
          {buttonName}
        </WbButton>
        <Panel className="custom-panel" aria-label={panelTitle}>
          <PanelHead
            title={panelTitle}
            meta="3 open"
            actions={<button type="button">Refresh</button>}
          />
          <PanelBody flush className="custom-body">
            <Kpi label="Ready" value={4} detail="available now" tone="green" />
          </PanelBody>
        </Panel>
        <Kpi label="Pending" value={2} />
        <Chip tone="amber" active title="Needs attention">
          Warning
        </Chip>
        <Chip>Neutral</Chip>
        <Dot tone="red" />
        <Ring value={140} size={80} tone="ice" label="Completion" />
        <Ring value={-5} />
        <Empty title={emptyTitle}>
          <span>Choose a workspace to continue</span>
          <button type="button">Create workspace</button>
        </Empty>
        <Skeleton height={24} width="60%" />
        <Banner tone="red" icon={<span aria-hidden>!</span>}>
          {alertMessage}
        </Banner>
        <Banner tone="amber">Heads up</Banner>
      </>,
      { withoutRouter: true },
    );

    expect(screen.getByRole("button", { name: buttonName })).toHaveAttribute(
      "data-control-variant",
      "primary",
    );
    expect(screen.getByRole("region", { name: panelTitle })).toHaveClass(
      "wb-panel",
      "custom-panel",
    );
    expect(byText("3 open")).toHaveClass("wb-panel-meta");
    expect(byText("available now")).toBeInTheDocument();
    expect(byText(pendingLabel)).toHaveClass("wb-kpi-k");
    expect(byText(pendingLabel).nextElementSibling).toHaveTextContent("2");
    expect(byText(warningLabel)).toHaveClass("wb-chip", "is-active", "is-amber");
    expect(byText(warningLabel)).toHaveAttribute("data-rcl-badge");
    expect(byText(warningLabel)).toHaveAttribute("role", "presentation");
    expect(byText(neutralLabel)).toHaveClass("wb-chip");
    expect(screen.getByRole("img", { name: "Completion" })).toHaveTextContent("100%");
    expect(screen.getByRole("img", { name: "0%" })).toHaveTextContent("0%");
    expect(byText(emptyTitle)).toBeInTheDocument();
    const skeleton = document.querySelector(".wb-skeleton");
    expect(skeleton).toHaveAttribute("data-rcl-skeleton");
    expect(skeleton).toHaveAttribute("aria-hidden", "true");
    expect(skeleton).toHaveStyle({ height: "24px", width: "60%" });
    expect(screen.getByRole("alert")).toHaveTextContent(alertMessage);
    expect(byText(headsUp).parentElement).not.toHaveAttribute("role", "alert");
  });

  it("forwards editable field, search, toggle, tab, and segmented-control interactions", () => {
    const onSearch = vi.fn();
    const onToggle = vi.fn();
    const onTab = vi.fn();
    const onSegment = vi.fn();
    const tabListName = "Workspace views";
    const searchLabel = "Filter components";
    const toggleLabel = "Include examples";
    const secondTabName = "History";
    const thirdTabName = "Settings";
    const thirdTabAccessibleName = "Settings";
    const overviewTabAccessibleName = "Overview";
    const segmentGroupName = "Assistance scope";
    const fieldLabel = "Category";
    const notesLabel = "Notes";
    const overviewTabName = "Overview";
    const frameSegmentName = "Frame";
    renderWithProviders(
      <>
        <Field label={fieldLabel} hint="Choose one">
          <Select defaultValue="component">
            <option value="component">Component</option>
            <option value="hook">Hook</option>
          </Select>
        </Field>
        <TextArea aria-label="Notes" className="notes" defaultValue="Initial" />
        <SearchBox
          label={searchLabel}
          placeholder="Search catalog"
          value="buttons"
          shortcut="/"
          onChange={onSearch}
        />
        <Toggle label={toggleLabel} checked={false} onChange={onToggle} />
        <Tabs
          label={tabListName}
          value="overview"
          onChange={onTab}
          items={[
            { id: "overview", label: overviewTabName, count: 2 },
            { id: "history", label: secondTabName },
            { id: "settings", label: thirdTabName, count: 1 },
          ]}
        />
        <Segmented
          label={segmentGroupName}
          value="design"
          onChange={onSegment}
          items={[
            { id: "design", label: "Design" },
            { id: "frame", label: frameSegmentName },
          ]}
        />
        <Meter value={140} tone="red" />
      </>,
      { withoutRouter: true },
    );

    const select = screen.getByLabelText(fieldLabel);
    fireEvent.change(select, { target: { value: "hook" } });
    expect(select).toHaveValue("hook");
    const hint = byText("Choose one");
    expect(hint).toBeInTheDocument();
    expect(hint).toHaveAttribute("id", `${select.id}-description`);
    expect(select).toHaveAttribute("aria-describedby", hint.id);
    expect(select).toHaveAttribute("data-rcl-form-field-control", "true");
    expect(screen.getByLabelText(notesLabel)).toHaveClass("wb-textarea", "notes");
    expect(screen.getByLabelText(searchLabel)).toHaveAttribute("placeholder", "Search catalog");
    fireEvent.change(screen.getByLabelText(searchLabel), { target: { value: "inputs" } });
    expect(onSearch).toHaveBeenCalledWith("inputs");
    const toggle = screen.getByRole("switch", { name: toggleLabel });
    expect(toggle).toHaveAttribute("data-rcl-selection-input");
    fireEvent.click(toggle);
    expect(onToggle).toHaveBeenCalledWith(true);
    expect(document.querySelectorAll("[data-rcl-tab-badge]")).toHaveLength(2);
    expect(document.querySelector("[data-rcl-tab-badge]")).toHaveTextContent("2");

    const historyTab = screen.getByRole("tab", { name: secondTabName });
    historyTab.focus();
    fireEvent.keyDown(historyTab, { key: "ArrowRight" });
    expect(onTab).toHaveBeenCalledWith("settings");
    expect(screen.getByRole("tab", { name: thirdTabAccessibleName })).toHaveFocus();
    fireEvent.keyDown(screen.getByRole("tab", { name: thirdTabAccessibleName }), { key: "End" });
    fireEvent.keyDown(screen.getByRole("tab", { name: thirdTabAccessibleName }), { key: "Home" });
    fireEvent.keyDown(screen.getByRole("tab", { name: overviewTabAccessibleName }), {
      key: "ArrowLeft",
    });
    expect(onTab.mock.calls.map(([tab]) => tab)).toEqual([
      "settings",
      "settings",
      "overview",
      "settings",
    ]);

    fireEvent.click(screen.getByRole("button", { name: frameSegmentName }));
    expect(onSegment).toHaveBeenCalledWith("frame");
    expect(screen.getByRole("group", { name: segmentGroupName })).toHaveAttribute(
      "data-rcl-button-group",
      "true",
    );
    expect(document.querySelector(".wb-meter-fill")).toHaveStyle({ width: "100%" });
  });
});
