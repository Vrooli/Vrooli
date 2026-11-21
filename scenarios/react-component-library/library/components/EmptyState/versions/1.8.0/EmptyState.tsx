/**
 * @libraryId react-component-library:EmptyState
 * @displayName EmptyState
 * @description The contextual no-results composition explaining why nothing is shown, distinguishing an empty collection from a filtered-out one, and offering the most relevant next action.
 * @version 1.8.0
 * @tags []
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useId, type CSSProperties, type ReactNode } from "react";
import { Button, type ButtonProps } from "@vrooli/react-component-library/Button/2";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Card, CardContent } from "@vrooli/react-component-library/Card/1";
import { Container } from "@vrooli/react-component-library/Container/1";
import { Heading } from "@vrooli/react-component-library/Heading/1";
import { Stack } from "@vrooli/react-component-library/Stack/1";
import { Text } from "@vrooli/react-component-library/Text/1";

const iconBadgeStyle: CSSProperties = {
  display: "grid",
  inlineSize: "var(--space-2xl)",
  blockSize: "var(--space-2xl)",
  flex: "0 0 auto",
  placeItems: "center",
  border:
    "var(--border-hairline) solid color-mix(in srgb, var(--color-primary) 22%, var(--color-border))",
  borderRadius: "var(--radius-pill)",
  background: "color-mix(in srgb, var(--color-primary) 10%, var(--color-surface))",
  color: "var(--color-primary)",
};

/**
 * 1.8.0 — `density="compact"` for dense contexts.
 *
 * The comfortable layout is a page-level moment: a centred card inside a
 * comfortable container with responsive block inset. Placed inside a list
 * panel, a side pane or another card it becomes a frame inside a frame, and its
 * inset pushes the message below the fold on a phone. Compact drops the card
 * and container (the host already supplies the surface), tightens the inset
 * and gap, shrinks the icon badge and uses the label text style for the title.
 * The heading level, region labelling and action group are unchanged.
 */
export type EmptyStateDensity = "comfortable" | "compact";

export interface EmptyStateProps {
  title: string;
  /** `compact` renders without its own card for lists, panes and cards. */
  density?: EmptyStateDensity;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
  actionLabel?: ReactNode;
  actionVariant?: ButtonProps["variant"];
  onAction?: ButtonProps["onClick"];
  className?: string;
}

export const EmptyState = withClassName(function EmptyState({
  title,
  description,
  icon,
  action,
  actionLabel,
  actionVariant = "primary",
  onAction,
  className,
  density = "comfortable",
}: EmptyStateProps) {
  const strings = useStrings();
  const id = `rcl-empty-state-${useId().replace(/:/g, "")}`;
  const titleId = `${id}-title`;
  const descriptionId = `${id}-description`;
  const resolvedAction =
    action ??
    (actionLabel ? (
      <Button type="button" variant={actionVariant} onClick={onAction}>
        {actionLabel}
      </Button>
    ) : null);

  const compact = density === "compact";
  const body = (
    <Stack
      gap={compact ? "sm" : "lg"}
      align="center"
      textAlign="center"
      measure="wide"
      insetBlock={compact ? "md" : "responsive"}
      insetInline={compact ? "md" : "lg"}
    >
      {icon ? (
        <div
          data-rcl-empty-state-icon
          aria-hidden="true"
          style={
            compact
              ? { ...iconBadgeStyle, inlineSize: "var(--space-xl)", blockSize: "var(--space-xl)" }
              : iconBadgeStyle
          }
        >
          {icon}
        </div>
      ) : null}
      <Stack
        gap="3xs"
        align="center"
        textAlign="center"
        measure="content"
        data-rcl-empty-state-copy
      >
        <Heading
          id={titleId}
          level={2}
          textStyle={compact ? "label" : "title"}
          balance
          data-rcl-empty-state-title
        >
          {title}
        </Heading>
        {description ? (
          <Text
            as="p"
            id={descriptionId}
            textStyle="body"
            tone="muted"
            balance
            data-rcl-empty-state-description
          >
            {description}
          </Text>
        ) : null}
      </Stack>
      {resolvedAction ? (
        <ButtonGroup
          align="center"
          collapse="sm"
          label={strings("feedback.empty-state.empty-state-actions", "Empty state actions")}
          data-rcl-empty-state-action
        >
          {resolvedAction}
        </ButtonGroup>
      ) : null}
    </Stack>
  );
  const region = {
    role: "region",
    "aria-labelledby": titleId,
    "aria-describedby": description ? descriptionId : undefined,
    "data-rcl-empty-state": "",
    "data-density": density,
  } as const;

  if (compact) {
    return (
      <div data-testid="feedback.empty-state" className={className} {...region}>
        {body}
      </div>
    );
  }
  return (
    <Container data-testid="feedback.empty-state" width="comfortable">
      <Card className={className} {...region}>
        <CardContent style={{ padding: 0 }}>{body}</CardContent>
      </Card>
    </Container>
  );
});
