import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import type { TemplateNode, TemplateNodeType } from "../templates/spec";
import { NodeInspector } from "./NodeInspector";

function node(type: TemplateNodeType, props: TemplateNode["props"]): TemplateNode {
  return { id: "selected", type, props };
}

describe("NodeInspector", () => {
  it("edits shared text, visual, layout, and boolean properties", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <NodeInspector
        node={node("text", {
          text: "Hello",
          variant: "body",
          tone: "ink",
          shape: "none",
          align: "left",
          gap: 8,
          cols: 2,
          pad: 12,
          h: 80,
          elevated: true,
        })}
        onChange={onChange}
      />,
    );

    fireEvent.change(screen.getByLabelText(strings.workbench.studio.text), {
      target: { value: "Updated" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.variant), {
      target: { value: "title" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.tone), {
      target: { value: "accent" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.shape), {
      target: { value: "arch" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.align), {
      target: { value: "center" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.gap), {
      target: { value: "16" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.cols), {
      target: { value: "9" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.pad), {
      target: { value: "24" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.height), {
      target: { value: "96" },
    });
    fireEvent.click(screen.getByRole("switch", { name: strings.workbench.studio.elevated }));

    expect(onChange.mock.calls).toEqual([
      ["text", "Updated"],
      ["variant", "title"],
      ["tone", "accent"],
      ["shape", "arch"],
      ["align", "center"],
      ["gap", 16],
      ["cols", 6],
      ["pad", 24],
      ["h", 96],
      ["elevated", false],
    ]);
    expect(screen.getByText(strings.workbench.studio.editNote)).toBeInTheDocument();
  });

  it("edits image accessibility and fit, and enforces the minimum image height", () => {
    const onChange = vi.fn();
    renderWithProviders(<NodeInspector node={node("image", {})} onChange={onChange} />);

    expect(screen.getByLabelText(strings.workbench.studio.imageHeight)).toHaveValue(220);
    expect(screen.getByLabelText(strings.workbench.studio.imageFit)).toHaveValue("cover");
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.imageAlt), {
      target: { value: "A mountain" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.imageFit), {
      target: { value: "contain" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.imageHeight), {
      target: { value: "20" },
    });

    expect(onChange.mock.calls).toEqual([
      ["alt", "A mountain"],
      ["fit", "contain"],
      ["h", 40],
    ]);
  });

  it("exposes the correct button variants and edits its label and placeholder", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <NodeInspector
        node={node("button", { label: "Save", placeholder: "Optional", variant: "primary" })}
        onChange={onChange}
      />,
    );

    const variant = screen.getByLabelText(strings.workbench.studio.variant);
    expect(
      Array.from((variant as HTMLSelectElement).options).map((option) => option.value),
    ).toEqual(["primary", "secondary", "ghost"]);
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.label), {
      target: { value: "Continue" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.placeholder), {
      target: { value: "Later" },
    });
    fireEvent.change(variant, { target: { value: "ghost" } });

    expect(onChange.mock.calls).toEqual([
      ["label", "Continue"],
      ["placeholder", "Later"],
      ["variant", "ghost"],
    ]);
  });

  it("edits stat and metric value fields without confusing their distinct controls", () => {
    const onStatChange = vi.fn();
    const { rerender } = renderWithProviders(
      <NodeInspector
        node={node("stat", { value: "42", detail: "users" })}
        onChange={onStatChange}
      />,
    );
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.value), {
      target: { value: "43" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.detail), {
      target: { value: "teams" },
    });
    expect(onStatChange.mock.calls).toEqual([
      ["value", "43"],
      ["detail", "teams"],
    ]);

    const onMetricChange = vi.fn();
    rerender(
      <NodeInspector
        node={node("metric", {
          value: "12",
          prefix: "$",
          suffix: "k",
          detail: "revenue",
          size: "md",
        })}
        onChange={onMetricChange}
      />,
    );
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.metricPrefix), {
      target: { value: "€" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.metricSuffix), {
      target: { value: "m" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.metricSize), {
      target: { value: "xl" },
    });
    expect(onMetricChange.mock.calls).toEqual([
      ["prefix", "€"],
      ["suffix", "m"],
      ["size", "xl"],
    ]);
  });

  it("clamps icon sizing and meter values to their supported ranges", () => {
    const onIconChange = vi.fn();
    const { rerender } = renderWithProviders(
      <NodeInspector node={node("icon", {})} onChange={onIconChange} />,
    );

    expect(screen.getByLabelText(strings.workbench.studio.iconName)).toHaveValue("spark");
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.iconName), {
      target: { value: "chart" },
    });
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.iconSize), {
      target: { value: "90" },
    });
    fireEvent.click(screen.getByRole("switch", { name: strings.workbench.studio.iconBox }));
    expect(onIconChange.mock.calls).toEqual([
      ["name", "chart"],
      ["size", 64],
      ["box", true],
    ]);

    const onMeterChange = vi.fn();
    rerender(<NodeInspector node={node("meter", { value: 50 })} onChange={onMeterChange} />);
    fireEvent.change(screen.getByLabelText(strings.workbench.studio.value), {
      target: { value: "-5" },
    });
    expect(onMeterChange).toHaveBeenCalledWith("value", 0);
  });

  it("does not show optional editors when a node has no editable properties", () => {
    const onChange = vi.fn();
    renderWithProviders(<NodeInspector node={node("divider", {})} onChange={onChange} />);

    expect(screen.getByText((content) => content === "divider")).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.studio.editNote)).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(onChange).not.toHaveBeenCalled();
  });
});
