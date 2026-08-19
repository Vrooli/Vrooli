/**
 * @libraryId react-component-library:EmptyState
 * @displayName Empty State
 * @description Compact empty-state panel with icon, copy, and optional action slots.
 * @version 1.4.0
 * @tags ["feedback","surface"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useId, type ReactNode } from "react";
import { Button, type ButtonProps } from "../../../Button/versions/2.2.0/Button";
import { Card, CardContent } from "../../../Card/versions/1.1.0/Card";
import { Heading } from "../../../Heading/versions/1.0.0/Heading";
import { Text } from "../../../Text/versions/1.0.0/Text";
import { emptyStateStyles } from "./styles";

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
    <>
      <style
        data-rcl-empty-state-styles
        dangerouslySetInnerHTML={{ __html: emptyStateStyles }}
      />
      <Card
        data-rcl-empty-state
        className={className}
        role="region"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
      >
        <CardContent className="rcl-empty-state__content">
          {icon ? (
            <div data-rcl-empty-state-icon aria-hidden="true">
              {icon}
            </div>
          ) : null}
          <div data-rcl-empty-state-copy>
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
          </div>
          {resolvedAction ? (
            <div data-rcl-empty-state-action>{resolvedAction}</div>
          ) : null}
        </CardContent>
      </Card>
    </>
  );
}