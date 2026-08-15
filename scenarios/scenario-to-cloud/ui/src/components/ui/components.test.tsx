import "@testing-library/jest-dom";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CheckCircle, Info, Rocket } from "lucide-react";
import { vi } from "vitest";

import { Alert } from "./alert";
import { AnnotatedCodeBlock, type LineAnnotation } from "./annotated-code-block";
import { Badge } from "./badge";
import { Button } from "./button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./card";
import { CodeBlock } from "./code-block";
import { Collapsible, StatusBadge } from "./collapsible";
import { EmptyState } from "./empty-state";
import { Input, Textarea } from "./input";
import { CompactSelectableCard, SelectableCard, type SelectableCardConfig } from "./selectable-card";
import { LoadingState, Spinner } from "./spinner";
import { Stepper, VerticalStepper, type Step } from "./stepper";
import { HelpTooltip, Tooltip } from "./tooltip";

// provider-free-exception: these primitive components intentionally have no provider dependencies.
vi.mock("shiki", () => ({
  createHighlighter: vi.fn(async () => ({
    getLoadedLanguages: () => ["json", "typescript", "text"],
    codeToHtml: (code: string) => `<pre><code>${code}</code></pre>`,
  })),
}));

const steps: Step[] = [
  { id: "one", label: "One", description: "First step" },
  { id: "two", label: "Two", description: "Second step" },
  { id: "three", label: "Three" },
];

const cardConfig: SelectableCardConfig = {
  id: "launch",
  name: "Launch",
  description: "Deploy the selected scenario",
  icon: Rocket,
};

describe("primitive UI components", () => {
  let writeText: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
  });

  it("renders alert variants, title, content, and dismiss action", () => {
    const onDismiss = vi.fn();
    render(
      <Alert variant="warning" title="Heads up" onDismiss={onDismiss}>
        Check the host first.
      </Alert>,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("Heads upCheck the host first.");
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("renders badges, cards, buttons, inputs, and textareas with their states", () => {
    const onClick = vi.fn();
    render(
      <>
        <Badge variant="success">Ready</Badge>
        <Button variant="destructive" size="sm" onClick={onClick}>Delete</Button>
        <Button asChild><a href="#target">Link</a></Button>
        <Card>
          <CardHeader><CardTitle>Title</CardTitle><CardDescription>Description</CardDescription></CardHeader>
          <CardContent>Content</CardContent>
          <CardFooter>Footer</CardFooter>
        </Card>
        <Input label="Host name" hint="DNS or IP" isLoading />
        <Input label="Bad value" warning="Needs review" />
        <Input label="Broken value" error="Invalid" />
        <Textarea label="Manifest" hint="JSON input" defaultValue="{}" />
        <Textarea label="Broken manifest" error="Malformed JSON" />
      </>,
    );

    expect(screen.getByText("Ready")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(onClick).toHaveBeenCalledOnce();
    expect(screen.getByRole("link", { name: "Link" })).toHaveAttribute("href", "#target");
    expect(screen.getByLabelText("Host name")).toBeInTheDocument();
    expect(screen.getByText("DNS or IP")).toBeInTheDocument();
    expect(screen.getByText("Needs review")).toBeInTheDocument();
    expect(screen.getByText("Invalid")).toBeInTheDocument();
    expect(screen.getByText("Malformed JSON")).toBeInTheDocument();
  });

  it("supports empty-state actions and loading states", () => {
    const action = vi.fn();
    render(
      <>
        <EmptyState icon={<Info />} title="Nothing here" description="Create a deployment." action={{ label: "Create", onClick: action }} />
        <Spinner size="sm" />
        <LoadingState message="Working" />
      </>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Create" }));
    expect(action).toHaveBeenCalledOnce();
    expect(screen.getAllByRole("status")).toHaveLength(2);
    expect(screen.getByText("Working")).toBeInTheDocument();
  });

  it("opens collapsibles and renders status badges", () => {
    render(
      <>
        <Collapsible title="Details" badge={<Badge>1</Badge>}>
          <p>Hidden detail</p>
        </Collapsible>
        <Collapsible title="Open" defaultOpen><p>Visible detail</p></Collapsible>
        <StatusBadge status="success">Complete</StatusBadge>
      </>,
    );

    expect(screen.queryByText("Hidden detail")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Details/ }));
    expect(screen.getByText("Hidden detail")).toBeInTheDocument();
    expect(screen.getByText("Visible detail")).toBeInTheDocument();
    expect(screen.getByText("Complete")).toBeInTheDocument();
  });

  it("supports selectable card radio and checkbox modes", () => {
    const onSelect = vi.fn();
    render(
      <>
        <SelectableCard config={cardConfig} selected onSelect={onSelect} selectionMode="radio" />
        <SelectableCard config={{ ...cardConfig, id: "disabled", name: "Disabled" }} selected={false} onSelect={onSelect} selectionMode="checkbox" disabled />
        <CompactSelectableCard config={{ ...cardConfig, name: "Compact" }} selected={false} onSelect={onSelect} selectionMode="checkbox" />
      </>,
    );

    fireEvent.click(screen.getByRole("button", { name: /Launch/ }));
    expect(onSelect).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: /Disabled/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /Compact/ })).toBeInTheDocument();
  });

  it("renders horizontal and vertical stepper states and allows permitted navigation", () => {
    const onStepClick = vi.fn();
    render(
      <>
        <Stepper
          steps={steps}
          currentStep={1}
          displayedStep={2}
          onStepClick={onStepClick}
          allowFutureClicks
          stepStates={["completed", "running", "skipped"]}
        />
        <VerticalStepper steps={steps} currentStep={1} onStepClick={onStepClick} />
      </>,
    );

    const twoLabels = screen.getAllByRole("button", { name: /Two/ });
    const twoLabel = twoLabels[0];
    if (!twoLabel) throw new Error("expected a horizontal or vertical Two step");
    fireEvent.click(twoLabel);
    expect(onStepClick).toHaveBeenCalledWith(1);
    expect(screen.getAllByRole("navigation", { name: "Progress" })).toHaveLength(2);
  });

  it("renders tooltips on focus and keeps placement bounded", async () => {
    render(
      <>
        <Tooltip content="More details"><span>?</span></Tooltip>
        <HelpTooltip content="Help text" />
      </>,
    );

    const triggers = screen.getAllByRole("button", { name: "More information" });
    const trigger = triggers[0];
    if (!trigger) throw new Error("expected a tooltip trigger");
    fireEvent.focus(trigger);
    expect(await screen.findByRole("tooltip")).toHaveTextContent("More details");
    fireEvent.blur(trigger);
    await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument());
  });

  it("copies code and exercises annotated editing, annotations, and scroll synchronization", async () => {
    const onAnnotationClick = vi.fn();
    const onChange = vi.fn();
    const annotations: LineAnnotation[] = [
      { line: 1, severity: "error", message: "Invalid host", hint: "Use a hostname", path: "target.host", fixable: true },
      { line: 2, severity: "warn", message: "Optional", path: "target.port" },
    ];
    render(
      <>
        <CodeBlock code={'{"host":"x"}\n{"port":443}'} filename="manifest.json" />
        <AnnotatedCodeBlock
          code={'{"host":"x"}\n{"port":443}'}
          filename="manifest.json"
          annotations={annotations}
          editable
          onChange={onChange}
          onAnnotationClick={onAnnotationClick}
          testId="manifest-editor"
        />
      </>,
    );

    expect(screen.queryByText("2 errors")).not.toBeInTheDocument();
    expect(screen.getByText("1 error")).toBeInTheDocument();
    expect(screen.getByText("1 warning")).toBeInTheDocument();
    fireEvent.change(screen.getByTestId("manifest-editor"), { target: { value: "{}" } });
    expect(onChange).toHaveBeenCalledWith("{}");
    const codeButtons = screen.getAllByRole("button", { name: "Copy code" });
    const codeButton = codeButtons[0];
    if (!codeButton) throw new Error("expected a code copy button");
    fireEvent.click(codeButton);
    await waitFor(() => expect(writeText).toHaveBeenCalled());
    const annotationLine = Array.from(document.querySelectorAll("div.cursor-pointer")).find((element) => element.querySelector("svg"));
    if (!annotationLine) throw new Error("expected an annotated code line");
    fireEvent.click(annotationLine);
    expect(onAnnotationClick).toHaveBeenCalledWith(annotations[0]);
  });

  it("supports plain fallback rendering, warning tooltips, and clipboard failures", async () => {
    const onAnnotationClick = vi.fn();
    const warning: LineAnnotation = { line: 1, severity: "warn", message: "Review value", path: "target.port" };
    const { rerender } = render(
      <AnnotatedCodeBlock
        code="plain text"
        language="unsupported-language"
        showHeader={false}
        showLineNumbers={false}
        annotations={[warning]}
        onAnnotationClick={onAnnotationClick}
        testId="plain-editor"
      />,
    );
    expect(screen.getByTestId("plain-editor")).toHaveTextContent("plain text");
    const line = document.querySelector("div.cursor-pointer");
    if (!line) throw new Error("expected warning line");
    fireEvent.mouseEnter(line);
    expect(await screen.findByText("Review value")).toBeInTheDocument();
    fireEvent.mouseLeave(line);
    fireEvent.click(line);
    expect(onAnnotationClick).toHaveBeenCalledWith(warning);
    writeText.mockRejectedValueOnce(new Error("clipboard denied"));
    rerender(<AnnotatedCodeBlock code="copy me" language="json" />);
    fireEvent.click(screen.getByRole("button", { name: "Copy code" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("copy me"));
  });
});
