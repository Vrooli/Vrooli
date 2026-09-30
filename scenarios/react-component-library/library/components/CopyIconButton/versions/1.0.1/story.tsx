import { CopyIconButton } from "./CopyIconButton";

type StoryProps = {
  args: Record<string, never>;
  log: (name: string, ...eventArgs: unknown[]) => void;
};

/** At rest: a ghost icon button named for what it copies. The write is stubbed so a preview never touches the clipboard. */
export function Default({ args, log }: StoryProps) {
  return (
    <CopyIconButton
      {...args}
      value="pnpm install"
      aria-label="Copy command"
      writeText={(text) => {
        log("copy", text);
        return true;
      }}
    />
  );
}

/** A write that fails turns the glyph into a cross in the danger colour and says so. */
export function Failed({ args, log }: StoryProps) {
  return (
    <CopyIconButton
      {...args}
      value="pnpm install"
      aria-label="Copy command"
      writeText={(text) => {
        log("copy-failed", text);
        return false;
      }}
    />
  );
}

/** Shown as copied without a press here, as when a menu row copied the same text. */
export function Controlled({ args, log }: StoryProps) {
  void log;
  return <CopyIconButton {...args} value="pnpm install" aria-label="Copy command" copied />;
}

export function SizeXSmall({ args, log }: StoryProps) {
  void log;
  return (
    <CopyIconButton
      {...args}
      value="pnpm install"
      aria-label="Copy command"
      size="xs"
      writeText={() => true}
    />
  );
}
