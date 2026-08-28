import { useState } from "react";

import { Input } from "@vrooli/react-component-library/Input/1.3.0";
import { Textarea } from "@vrooli/react-component-library/Textarea/1.1.0";

import { InputGroup } from "./InputGroup";

const SendGlyph = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d="M3.7 3a.5.5 0 0 0-.68.63l2.84 7.63a2 2 0 0 1 0 1.4L3.02 20.3a.5.5 0 0 0 .68.63l18-8.5a.5.5 0 0 0 0-.9z" />
    <path d="M6 12h16" />
  </svg>
);

const ExpandGlyph = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d="M15 3h6v6" />
    <path d="M9 21H3v-6" />
    <path d="m21 3-7 7" />
    <path d="m3 21 7-7" />
  </svg>
);

const frame = { inlineSize: "min(100%, 420px)", padding: "var(--space-sm, 16px)" };

/** The plain case: a group is still a group with nothing attached to it. */
export function Default() {
  return (
    <div style={frame}>
      <InputGroup testId="story-group">
        <Input aria-label="Message" placeholder="Message" />
      </InputGroup>
    </div>
  );
}

/**
 * A leading adornment and a joined trailing segment — the launcher shape.
 * The `$` is a real part of the field rather than an absolutely-positioned
 * span with hand-tuned padding behind it.
 */
export function PrefixAndSegment() {
  return (
    <div style={frame}>
      <InputGroup testId="story-group">
        <InputGroup.Adornment side="leading">$</InputGroup.Adornment>
        <InputGroup.Field>
          <Input aria-label="Command" defaultValue="claude --resume" />
        </InputGroup.Field>
        <InputGroup.Segment side="trailing" emphasis="solid" aria-label="Launch" testId="story-launch">
          Launch
        </InputGroup.Segment>
      </InputGroup>
    </div>
  );
}

/**
 * Two actions on one growing field, each anchored differently. Expand holds
 * the top corner because a growing box keeps its top edge still; send holds
 * the bottom because that is where the eye and the thumb end up.
 */
export function ComposerActions() {
  return (
    <div style={frame}>
      <InputGroup testId="story-group">
        <InputGroup.Field>
          <Textarea aria-label="Draft" rows={3} defaultValue={"one\ntwo\nthree"} />
        </InputGroup.Field>
        <InputGroup.Action align="start" testId="story-expand-slot">
          <button type="button" aria-label="Expand" data-testid="story-expand">
            <ExpandGlyph />
          </button>
        </InputGroup.Action>
        <InputGroup.Action align="end" testId="story-send-slot">
          <button type="button" aria-label="Send" data-testid="story-send">
            <SendGlyph />
          </button>
        </InputGroup.Action>
      </InputGroup>
    </div>
  );
}

/** Steppers either side of a value, with the unit bound to it as a suffix. */
export function JoinedSteppers() {
  const [value, setValue] = useState(14);
  return (
    <div style={frame}>
      <InputGroup testId="story-group" shape="pill" size="sm" style={{ inlineSize: "max-content" }}>
        <InputGroup.Segment
          side="leading"
          aria-label="Decrease"
          testId="story-decrease"
          disabled={value <= 8}
          onClick={() => { setValue((current) => Math.max(8, current - 1)); }}
        >
          &minus;
        </InputGroup.Segment>
        <InputGroup.Field>
          <Input
            aria-label="Font size"
            data-testid="story-value"
            inputMode="numeric"
            readOnly
            value={String(value)}
            style={{ inlineSize: "3ch", textAlign: "center", paddingInline: 0 }}
          />
          <InputGroup.Adornment side="trailing">px</InputGroup.Adornment>
        </InputGroup.Field>
        <InputGroup.Segment
          side="trailing"
          aria-label="Increase"
          testId="story-increase"
          disabled={value >= 32}
          onClick={() => { setValue((current) => Math.min(32, current + 1)); }}
        >
          +
        </InputGroup.Segment>
      </InputGroup>
    </div>
  );
}

/** The invalid tone recolours the shared border rather than one child's. */
export function Invalid() {
  return (
    <div style={frame}>
      <InputGroup testId="story-group" tone="invalid">
        <InputGroup.Field>
          <Input aria-label="Port" aria-invalid defaultValue="not-a-port" />
        </InputGroup.Field>
        <InputGroup.Segment side="trailing" aria-label="Check" testId="story-check">
          Check
        </InputGroup.Segment>
      </InputGroup>
    </div>
  );
}

/** A disabled group blocks pointer input across every part at once. */
export function Disabled() {
  return (
    <div style={frame}>
      <InputGroup testId="story-group" disabled>
        <InputGroup.Field>
          <Input aria-label="Command" defaultValue="locked" disabled />
        </InputGroup.Field>
        <InputGroup.Segment side="trailing" aria-label="Launch" testId="story-launch">
          Launch
        </InputGroup.Segment>
      </InputGroup>
    </div>
  );
}

/**
 * The same composer once the field has grown. The actions leave their fixed
 * lane and take a row beneath the text — which is also what gives the text the
 * full width back, so a field that grew because it ran out of room stops
 * running out of room.
 */
export function GrownActions() {
  return (
    <div style={frame}>
      <InputGroup testId="story-group" shape="pill" grown>
        <InputGroup.Field>
          <Textarea aria-label="Draft" rows={4} defaultValue={"one\ntwo\nthree\nfour"} />
        </InputGroup.Field>
        <InputGroup.Action align="start" testId="story-expand-slot">
          <button type="button" aria-label="Expand" data-testid="story-expand">
            <ExpandGlyph />
          </button>
        </InputGroup.Action>
        <InputGroup.Action align="end" testId="story-send-slot">
          <button type="button" aria-label="Send" data-testid="story-send">
            <SendGlyph />
          </button>
        </InputGroup.Action>
      </InputGroup>
    </div>
  );
}

/** The resting pill: one row, actions still inline in their lane. */
export function PillAtRest() {
  return (
    <div style={frame}>
      <InputGroup testId="story-group" shape="pill">
        <InputGroup.Field>
          <Input aria-label="Message" placeholder="Message" />
        </InputGroup.Field>
        <InputGroup.Action align="end" testId="story-send-slot">
          <button type="button" aria-label="Send" data-testid="story-send">
            <SendGlyph />
          </button>
        </InputGroup.Action>
      </InputGroup>
    </div>
  );
}
