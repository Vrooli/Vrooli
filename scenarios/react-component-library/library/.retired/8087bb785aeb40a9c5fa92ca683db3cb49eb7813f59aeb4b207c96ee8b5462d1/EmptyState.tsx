/**
 * @libraryId react-component-library:EmptyState
 * @displayName Empty State
 * @description Compact empty-state panel with icon, copy, and optional action slots.
 * @version 1.7.9
 * @tags ["feedback","surface"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useId, type CSSProperties, type ReactNode } from "react";
import {
  Button,
  type ButtonProps,
} from "@vrooli/react-component-library/Button/2";
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
  background:
    "color-mix(in srgb, var(--color-primary) 10%, var(--color-surface))",
  color: "var(--color-primary)",
};

export interface EmptyStateProps {
  title: string;
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

  return (
    <Container data-testid="feedback.empty-state" width="comfortable">
      <Card
        data-rcl-empty-state
        className={className}
        role="region"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
      >
        <CardContent style={{ padding: 0 }}>
          <Stack
            gap="lg"
            align="center"
            textAlign="center"
            measure="wide"
            insetBlock="responsive"
            insetInline="lg"
          >
            {icon ? (
              <div
                data-rcl-empty-state-icon
                aria-hidden="true"
                style={iconBadgeStyle}
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
                textStyle="title"
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
                label={strings(
                  "feedback.empty-state.empty-state-actions",
                  "Empty state actions",
                )}
                data-rcl-empty-state-action
              >
                {resolvedAction}
              </ButtonGroup>
            ) : null}
          </Stack>
        </CardContent>
      </Card>
    </Container>
  );
});
