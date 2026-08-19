/**
 * @libraryId react-component-library:EmptyState
 * @displayName Empty State
 * @description Compact empty-state panel with icon, copy, and optional action slots.
 * @version 1.7.0
 * @tags ["feedback","surface"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useId, type CSSProperties, type ReactNode } from "react";
import { Button, type ButtonProps } from "../../../Button/versions/2.2.0/Button";
import { ButtonGroup } from "../../../ButtonGroup/versions/1.1.0/ButtonGroup";
import { Card, CardContent } from "../../../Card/versions/1.1.0/Card";
import { Container } from "../../../../primitives/Container/versions/1.1.0/Container";
import { Heading } from "../../../../primitives/Heading/versions/1.1.0/Heading";
import { Stack } from "../../../../primitives/Stack/versions/1.2.0/Stack";
import { Text } from "../../../../primitives/Text/versions/1.1.0/Text";

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

export function EmptyState({
  title,
  description,
  icon,
  action,
  actionLabel,
  actionVariant = "primary",
  onAction,
  className,
}: EmptyStateProps) {
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
    <Container width="comfortable">
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
                label="Empty state actions"
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
}