import { renderWithProviders } from "@/test-utils/renderWithProviders";
/**
 * Tests for Tabs component
 */

import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "./tabs";

describe("Tabs", () => {
  it("renders tabs root", () => {
    renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
      </Tabs>
    );

    expect(screen.getByTestId("tabs-root")).toBeInTheDocument();
  });

  it("renders tabs list", () => {
    renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
      </Tabs>
    );

    expect(screen.getByTestId("tabs-list")).toBeInTheDocument();
    expect(screen.getByRole("tablist")).toBeInTheDocument();
  });

  it("renders active tab content", () => {
    renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
        <TabsContent value="tab2">Content 2</TabsContent>
      </Tabs>
    );

    expect(screen.getByText("Content 1")).toBeInTheDocument();
    expect(screen.queryByText("Content 2")).not.toBeInTheDocument();
  });

  it("calls onValueChange when tab is clicked", () => {
    const onValueChange = vi.fn();
    renderWithProviders(
      <Tabs value="tab1" onValueChange={onValueChange}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
        <TabsContent value="tab2">Content 2</TabsContent>
      </Tabs>
    );

    fireEvent.click(screen.getByTestId("tab-trigger-tab2"));
    expect(onValueChange).toHaveBeenCalledWith("tab2");
  });

  it("marks selected tab with aria-selected", () => {
    renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
        <TabsContent value="tab2">Content 2</TabsContent>
      </Tabs>
    );

    expect(screen.getByTestId("tab-trigger-tab1")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("tab-trigger-tab2")).toHaveAttribute("aria-selected", "false");
  });

  it("renders tab panel with correct role", () => {
    renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
      </Tabs>
    );

    expect(screen.getByRole("tabpanel")).toBeInTheDocument();
    expect(screen.getByTestId("tab-content-tab1")).toBeInTheDocument();
  });

  it("disables tab trigger when disabled prop is true", () => {
    renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2" disabled>
            Tab 2
          </TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
        <TabsContent value="tab2">Content 2</TabsContent>
      </Tabs>
    );

    expect(screen.getByTestId("tab-trigger-tab2")).toBeDisabled();
  });

  it("does not call onValueChange when disabled tab is clicked", () => {
    const onValueChange = vi.fn();
    renderWithProviders(
      <Tabs value="tab1" onValueChange={onValueChange}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2" disabled>
            Tab 2
          </TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
        <TabsContent value="tab2">Content 2</TabsContent>
      </Tabs>
    );

    fireEvent.click(screen.getByTestId("tab-trigger-tab2"));
    expect(onValueChange).not.toHaveBeenCalled();
  });

  it("applies custom className to components", () => {
    renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()} className="custom-tabs">
        <TabsList className="custom-list">
          <TabsTrigger value="tab1" className="custom-trigger">
            Tab 1
          </TabsTrigger>
        </TabsList>
        <TabsContent value="tab1" className="custom-content">
          Content 1
        </TabsContent>
      </Tabs>
    );

    expect(screen.getByTestId("tabs-root")).toHaveClass("custom-tabs");
    expect(screen.getByTestId("tabs-list")).toHaveClass("custom-list");
    expect(screen.getByTestId("tab-trigger-tab1")).toHaveClass("custom-trigger");
    expect(screen.getByTestId("tab-content-tab1")).toHaveClass("custom-content");
  });

  it("switches content when value changes", () => {
    const { rerender } = renderWithProviders(
      <Tabs value="tab1" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
        <TabsContent value="tab2">Content 2</TabsContent>
      </Tabs>
    );

    expect(screen.getByText("Content 1")).toBeInTheDocument();
    expect(screen.queryByText("Content 2")).not.toBeInTheDocument();

    rerender(
      <Tabs value="tab2" onValueChange={vi.fn()}>
        <TabsList>
          <TabsTrigger value="tab1">Tab 1</TabsTrigger>
          <TabsTrigger value="tab2">Tab 2</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1">Content 1</TabsContent>
        <TabsContent value="tab2">Content 2</TabsContent>
      </Tabs>
    );

    expect(screen.queryByText("Content 1")).not.toBeInTheDocument();
    expect(screen.getByText("Content 2")).toBeInTheDocument();
  });
});
