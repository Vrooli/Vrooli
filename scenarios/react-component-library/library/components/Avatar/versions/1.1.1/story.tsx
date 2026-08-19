import { Avatar, AvatarGroup } from "./Avatar";
import { Card } from "../../../../components/Card/versions/1.1.0/Card";
import { Heading } from "../../../../primitives/Heading/versions/1.1.0/Heading";
import { Stack } from "../../../../primitives/Stack/versions/1.2.1/Stack";
import { Text } from "../../../../primitives/Text/versions/1.1.0/Text";

const image =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 120 120'%3E%3Cdefs%3E%3ClinearGradient id='g' x1='0' x2='1'%3E%3Cstop stop-color='%2338bdf8'/%3E%3Cstop offset='1' stop-color='%237c3aed'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='120' height='120' fill='url(%23g)'/%3E%3Ccircle cx='60' cy='47' r='23' fill='white' fill-opacity='.8'/%3E%3Cpath d='M18 116c4-29 20-42 42-42s38 13 42 42' fill='white' fill-opacity='.8'/%3E%3C/svg%3E";

const shell = {
  inlineSize: "100%",
  maxInlineSize: "35rem",
  minInlineSize: 0,
  minBlockSize: "18rem",
} as const;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: React.ReactNode;
}) {
  return (
    <Card style={shell}>
      <Stack gap="lg" inset="xl">
        <Stack gap="2xs">
          <Text textStyle="overline" tone="accent">
            Identity primitive
          </Text>
          <Heading
            level={2}
            textStyle="title"
            balance
            style={{ minInlineSize: 0 }}
          >
            {title}
          </Heading>
          <Text
            tone="muted"
            balance
            style={{ minInlineSize: 0, overflowWrap: "anywhere" }}
          >
            {detail}
          </Text>
        </Stack>
        {children}
      </Stack>
    </Card>
  );
}

export function Default() {
  return (
    <Showcase
      title="Identity without ambiguity"
      detail="A named avatar keeps the person accessible while presence remains an additional, non-color-only signal."
    >
      <Avatar name="Maya Chen" src={image} size="lg" presence="online" />
    </Showcase>
  );
}

export function Loading() {
  return (
    <Showcase
      title="Space reserved before arrival"
      detail="Loading imagery does not move the surrounding transcript or hide the identity fallback contract."
    >
      <Avatar
        name="Maya Chen"
        src={image}
        size="lg"
        presence="away"
        placeholder={<span data-rcl-avatar-loading aria-hidden="true" />}
      />
    </Showcase>
  );
}

export function RequestError() {
  return (
    <Showcase
      title="A useful image failure"
      detail="When the image cannot load, deterministic initials preserve identity instead of leaving a broken rectangle."
    >
      <Avatar name="Maya Chen" size="lg" presence="busy" />
    </Showcase>
  );
}

export function Group() {
  return (
    <Showcase
      title="Groups preserve the people count"
      detail="The overflow affordance names how many additional people are present and remains keyboard-readable."
    >
      <AvatarGroup maxVisible={3} label="Reviewers">
        <Avatar name="Maya Chen" size="lg" />
        <Avatar name="Ravi Shah" size="lg" />
        <Avatar name="Ada Lovelace" size="lg" />
        <Avatar name="Noah Williams" />
      </AvatarGroup>
    </Showcase>
  );
}
