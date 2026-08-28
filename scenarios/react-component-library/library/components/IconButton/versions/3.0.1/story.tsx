import { useState } from "react";
import { IconButton } from "./IconButton";

type StoryProps = {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
};

/**
 * Icons are declared as distinct module-scope components, which is how every
 * icon library builds them. The swap detection keys off that identity, so
 * declaring them inline would defeat the thing these stories demonstrate.
 */
function BubbleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      <path d="M13 8H7" />
      <path d="M17 12H7" />
    </svg>
  );
}

function TerminalIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m7 11 2-2-2-2" />
      <path d="M11 13h4" />
      <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
    </svg>
  );
}

function CloseIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M18 6 6 18" />
      <path d="m6 6 12 12" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
      strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 6h18" />
      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
      <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
    </svg>
  );
}

export function Default({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Close panel">
      <CloseIcon />
    </IconButton>
  );
}

export function SurfaceSoft({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Show messages" surface="soft">
      <BubbleIcon />
    </IconButton>
  );
}

export function SurfaceSolid({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Send" surface="solid">
      <BubbleIcon />
    </IconButton>
  );
}

export function SurfaceDanger({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Delete" surface="danger">
      <TrashIcon />
    </IconButton>
  );
}

export function ShapeRounded({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Rounded" shape="rounded">
      <CloseIcon />
    </IconButton>
  );
}

export function ShapeSquare({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Square" shape="square">
      <CloseIcon />
    </IconButton>
  );
}

export function SizeSmall({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Small" size="sm">
      <CloseIcon />
    </IconButton>
  );
}

export function SizeLarge({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Large" size="lg">
      <CloseIcon />
    </IconButton>
  );
}

/** A toggle that is on. The state is announced, not merely coloured. */
export function Selected({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Show messages" surface="soft" selected>
      <BubbleIcon />
    </IconButton>
  );
}

export function Pending({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Saving" pending pendingLabel="Saving…">
      <BubbleIcon />
    </IconButton>
  );
}

export function Disabled({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Unavailable" disabled>
      <CloseIcon />
    </IconButton>
  );
}

/** A 2.x call site, unchanged. It must still compile and still look right. */
export function LegacyVariant({ args, log }: StoryProps) {
  void log;
  return (
    <IconButton {...args} aria-label="Legacy secondary" variant="secondary">
      <BubbleIcon />
    </IconButton>
  );
}

/**
 * The view toggle this component was rebuilt for. Nothing here asks for motion:
 * the icon simply changes, and the control animates it.
 */
export function IconSwap({ args, log }: StoryProps) {
  const [terminal, setTerminal] = useState(false);
  return (
    <IconButton
      {...args}
      aria-label={terminal ? "Show messages" : "Show terminal"}
      surface="soft"
      onClick={() => {
        setTerminal((value) => !value);
        log("toggle");
      }}
    >
      {terminal ? <TerminalIcon /> : <BubbleIcon />}
    </IconButton>
  );
}
