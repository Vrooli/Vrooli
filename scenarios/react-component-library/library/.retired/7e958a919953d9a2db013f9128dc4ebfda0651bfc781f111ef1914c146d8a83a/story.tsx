import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useEffect, useState } from "react";
import {
  Avatar,
  AvatarGroup,
  type AvatarPresence,
  type AvatarProps,
} from "./Avatar";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card } from "@vrooli/react-component-library/Card/1.2.2";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
  usePopover,
} from "@vrooli/react-component-library/Popover/1.1.0";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@vrooli/react-component-library/Tooltip/1.0.1";
import { Heading } from "@vrooli/react-component-library/Heading/1.1.0";
import { Stack } from "@vrooli/react-component-library/Stack/1.2.1";
import { Text } from "@vrooli/react-component-library/Text/1";

const image =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 120 120'%3E%3Cdefs%3E%3ClinearGradient id='g' x1='0' x2='1'%3E%3Cstop stop-color='%2338bdf8'/%3E%3Cstop offset='1' stop-color='%237c3aed'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='120' height='120' fill='url(%23g)'/%3E%3Ccircle cx='60' cy='47' r='23' fill='white' fill-opacity='.8'/%3E%3Cpath d='M18 116c4-29 20-42 42-42s38 13 42 42' fill='white' fill-opacity='.8'/%3E%3C/svg%3E";

type AvatarStoryArgs = Pick<
  AvatarProps,
  "name" | "size" | "shape" | "presence" | "presenceLabel" | "imageState"
>;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: React.ReactNode;
}) {
  const libraryStrings = useStrings();
  return (
    <Card
      style={{ inlineSize: "100%", maxInlineSize: "35rem", minInlineSize: 0 }}
    >
      <Stack gap="lg" inset="xl">
        <Stack gap="2xs">
          <Text textStyle="overline" tone="accent">
            {libraryStrings(
              "primitives.avatar.identity-primitive",
              "Identity primitive",
            )}
          </Text>
          <Heading level={2} textStyle="title" balance className="min-w-0">
            {title}
          </Heading>
          <Text tone="muted" balance className="min-w-0 break-words">
            {detail}
          </Text>
        </Stack>
        {children}
      </Stack>
    </Card>
  );
}

function avatarProps(args: AvatarStoryArgs): AvatarProps {
  return args;
}

export function Default({ args }: StoryHarnessProps<AvatarStoryArgs>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.avatar.title",
        "Identity without ambiguity",
      )}
      detail="A named avatar keeps the person accessible while presence remains an additional, non-color-only signal."
    >
      <Avatar {...avatarProps(args)} src={image} />
    </Showcase>
  );
}

function DetailsClose() {
  const libraryStrings = useStrings();
  const { setOpen } = usePopover();
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      onClick={() => setOpen(false)}
    >
      {libraryStrings("primitives.avatar.close-details", "Close details")}
    </Button>
  );
}

export function IdentityDetails({ args }: StoryHarnessProps<AvatarStoryArgs>) {
  const libraryStrings = useStrings();
  const presence = args.presence ?? "online";
  return (
    <Showcase
      title={libraryStrings(
        "primitives.avatar.title.details-without-a-nested-control-trap",
        "Details without a nested-control trap",
      )}
      detail="Hover or focus the identity for a concise label. Open the profile card when the user needs more context; both overlays reuse shared interaction primitives."
    >
      <Stack gap="md" align="center">
        <Tooltip delay={0}>
          <TooltipTrigger asChild aria-label={`Show details for ${args.name}`}>
            <button
              type="button"
              style={{
                border: 0,
                padding: "var(--space-2xs)",
                borderRadius: "var(--radius-pill)",
                background: "transparent",
                cursor: "pointer",
              }}
            >
              <Avatar
                {...avatarProps(args)}
                size={args.size ?? "xl"}
                src={image}
              />
            </button>
          </TooltipTrigger>
          <TooltipContent>
            {args.name} · {presence}
          </TooltipContent>
        </Tooltip>
        <Popover>
          <PopoverTrigger asChild>
            <Button type="button" variant="secondary">
              {libraryStrings(
                "primitives.avatar.open-profile-card",
                "Open profile card",
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent
            initialFocus="none"
            aria-label={`${args.name} profile details`}
          >
            <Stack gap="sm" inset="md">
              <Stack gap="2xs">
                <Text textStyle="overline" tone="accent">
                  {libraryStrings("primitives.avatar.profile", "Profile")}
                </Text>
                <Heading level={3} textStyle="heading">
                  {args.name}
                </Heading>
                <Text tone="muted">
                  Currently {presence}. The avatar remains the identity anchor.
                </Text>
              </Stack>
              <DetailsClose />
            </Stack>
          </PopoverContent>
        </Popover>
      </Stack>
    </Showcase>
  );
}

export function PresenceCycle({
  args,
  log,
}: StoryHarnessProps<AvatarStoryArgs>) {
  const libraryStrings = useStrings();
  const [presence, setPresence] = useState<AvatarPresence>(
    args.presence ?? "online",
  );
  useEffect(() => setPresence(args.presence ?? "online"), [args.presence]);
  const nextPresence: Record<AvatarPresence, AvatarPresence> = {
    online: "away",
    away: "busy",
    busy: "offline",
    offline: "online",
  };
  const updatePresence = () => {
    const next = nextPresence[presence];
    setPresence(next);
    log("presence-change", next);
  };
  return (
    <Showcase
      title={libraryStrings(
        "primitives.avatar.title.status-that-feels-alive",
        "Status that feels alive",
      )}
      detail="The indicator enters and exits with the existing Presence primitive while the identity remains stable."
    >
      <Stack gap="sm" align="center">
        <Avatar
          {...avatarProps(args)}
          presence={presence}
          presenceLabel={`${args.name} is ${presence}`}
          src={image}
        />
        <Button type="button" variant="secondary" onClick={updatePresence}>
          {libraryStrings("primitives.avatar.cycle-status", "Cycle status")}
        </Button>
        <output
          aria-live="polite"
          className="text-xs text-app-muted-foreground"
        >
          Current status: {presence}
        </output>
      </Stack>
    </Showcase>
  );
}

export function Loading({ args }: StoryHarnessProps<AvatarStoryArgs>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.avatar.title.space-reserved-before-arrival",
        "Space reserved before arrival",
      )}
      detail="The controlled image state keeps the frame stable and lets the loading treatment be inspected instead of disappearing immediately."
    >
      <Avatar {...avatarProps(args)} src={image} imageState="loading" />
    </Showcase>
  );
}

export function RequestError({ args }: StoryHarnessProps<AvatarStoryArgs>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.avatar.title.a-useful-image-failure",
        "A useful image failure",
      )}
      detail="When the image cannot load, deterministic initials preserve identity and the error surface remains announced."
    >
      <Avatar {...avatarProps(args)} src={image} imageState="error" />
    </Showcase>
  );
}

export function Group({ args }: StoryHarnessProps<AvatarStoryArgs>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.avatar.title.groups-preserve-the-people-count",
        "Groups preserve the people count",
      )}
      detail="The overflow affordance names how many additional people are present and remains keyboard-readable."
    >
      <AvatarGroup
        maxVisible={3}
        label={libraryStrings("primitives.avatar.label", "Reviewers")}
      >
        <Avatar {...avatarProps(args)} name="Maya Chen" src={image} />
        <Avatar
          {...avatarProps(args)}
          name="Ravi Shah"
          src={image}
          presence={undefined}
        />
        <Avatar
          {...avatarProps(args)}
          name="Ada Lovelace"
          src={image}
          presence="away"
        />
        <Avatar
          {...avatarProps(args)}
          name="Noah Williams"
          src={image}
          presence={undefined}
        />
      </AvatarGroup>
    </Showcase>
  );
}
