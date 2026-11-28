import { useState } from "react";

import { SearchInput, type SearchInputSuggestion } from "./SearchInput";

const frame = { display: "grid", gap: "var(--space-md)", padding: "var(--space-sm)" };

const SUGGESTIONS: SearchInputSuggestion[] = [
  { value: "Search inputs", description: "Fields that collect a query." },
  { value: "Search results", description: "The collection that answers it." },
  { value: "Search settings", description: "Scopes, limits and defaults." },
  { value: "Research notes", description: "Saved findings from the workbench." },
];

const RECENT = ["deployments", "continuity catalog", "voice capture"];

/** The form convention: a visible name, a rounded field, a comfortable width. */
export function Default() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-default"
        label="Search"
        placeholder="Search pages"
        defaultValue="release"
      />
    </div>
  );
}

/** The two public appearances side by side: labelled form and command strip. */
export function Appearances() {
  return (
    <div style={frame}>
      <div data-testid="story-form-wrap">
        <SearchInput testId="story-form" label="Search pages" placeholder="Search pages" />
      </div>
      <div data-testid="story-toolbar-wrap">
        <SearchInput
          testId="story-toolbar"
          label="Search the library"
          appearance="toolbar"
          placeholder="Search the library…"
          shortcut="⌘K"
        />
      </div>
    </div>
  );
}

/** The size axis resolves through the shared InputGroup geometry. */
export function Sizes() {
  return (
    <div style={frame}>
      <SearchInput testId="story-size-sm" label="Compact search" size="sm" placeholder="Compact" />
      <SearchInput
        testId="story-size-md"
        label="Standard search"
        size="md"
        placeholder="Standard"
      />
      <SearchInput
        testId="story-size-lg"
        label="Comfortable search"
        size="lg"
        placeholder="Comfortable"
      />
    </div>
  );
}

/** Shape is independent of appearance, so a form field can be a pill too. */
export function Shapes() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-shape-rounded"
        label="Rounded search"
        shape="rounded"
        placeholder="Rounded"
      />
      <SearchInput testId="story-shape-pill" label="Pill search" shape="pill" placeholder="Pill" />
    </div>
  );
}

/** A label may be hidden without disappearing from the accessibility tree. */
export function LabelVisibility() {
  return (
    <div style={frame}>
      <SearchInput testId="story-label-visible" label="Visible label" placeholder="Visible" />
      <SearchInput
        testId="story-label-hidden"
        label="Hidden label"
        labelVisibility="hidden"
        placeholder="Hidden"
      />
    </div>
  );
}

/** Selecting a suggestion writes the query and closes the panel. */
export function KeyboardSelection() {
  const [value, setValue] = useState("");
  return (
    <div style={frame}>
      <SearchInput
        testId="story-keyboard"
        label="Search the library"
        appearance="toolbar"
        value={value}
        onValueChange={setValue}
        suggestions={SUGGESTIONS}
        defaultOpen
      />
      <output data-testid="story-keyboard-value">{value}</output>
    </div>
  );
}

/** The clear action is one explicit control, not a browser-native extra. */
export function Clear() {
  const [value, setValue] = useState("release notes");
  return (
    <div style={frame}>
      <SearchInput
        testId="story-clear"
        label="Search"
        value={value}
        onValueChange={setValue}
        placeholder="Search pages"
      />
      <output data-testid="story-clear-value">{value}</output>
    </div>
  );
}

/** Recent queries appear when the field is empty, before any search runs. */
export function Recent() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-recent"
        label="Search the library"
        appearance="toolbar"
        recentQueries={RECENT}
        defaultOpen
      />
    </div>
  );
}

/** A host-owned loading state keeps the panel open and labelled. */
export function Loading() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-loading"
        label="Search the library"
        appearance="toolbar"
        suggestions={SUGGESTIONS}
        defaultOpen
        loading
      />
    </div>
  );
}

/** A failed search explains itself and offers one retry. */
export function Error() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-error"
        label="Search the library"
        appearance="toolbar"
        defaultOpen
        searchError="The index is unavailable. Try again in a moment."
        onRetry={() => undefined}
      />
    </div>
  );
}

/** A real query with no matches is distinct from a field with nothing to search. */
export function Empty() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-empty"
        label="Search the library"
        appearance="toolbar"
        defaultValue="zzzz"
        suggestions={SUGGESTIONS}
        defaultOpen
      />
    </div>
  );
}

/** Long suggestions wrap without widening the panel or clipping the description. */
export function Overflow() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-overflow"
        label="Search the library"
        appearance="toolbar"
        defaultValue="workflow"
        defaultOpen
        suggestions={[
          {
            value: "Workflow capture and the continuity projection",
            description:
              "A deliberately long suggestion that must wrap inside the field's panel rather than forcing the page wider.",
          },
        ]}
      />
    </div>
  );
}

/** Disabled blocks the query, the clear action and the panel together. */
export function Disabled() {
  return (
    <div style={frame}>
      <SearchInput testId="story-disabled" label="Search" defaultValue="locked query" disabled />
    </div>
  );
}

/** Description and validation live below the field and stay associated with it. */
export function DescriptionError() {
  return (
    <div style={frame}>
      <SearchInput
        testId="story-described"
        label="Search"
        description="Searches titles, descriptions and saved states."
        error="Enter a query before searching."
        defaultValue=""
      />
    </div>
  );
}

/** The editor is the whole field, so it can be used in a form without a wrapper. */
export function FormSubmit() {
  const [submitted, setSubmitted] = useState("");
  return (
    <form
      style={frame}
      onSubmit={(event) => {
        event.preventDefault();
        const data = new FormData(event.currentTarget);
        setSubmitted(String(data.get("query") ?? ""));
      }}
    >
      <SearchInput
        testId="story-submit"
        name="query"
        label="Search"
        placeholder="Press Enter to search"
        onSubmit={setSubmitted}
      />
      <output data-testid="story-submit-value">{submitted}</output>
    </form>
  );
}
