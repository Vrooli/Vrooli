/**
 * @libraryId react-component-library:Heading
 * @displayName Heading
 * @description Heading selects semantic heading markup and one of the eight bundled text styles.
 * @version 1.1.1
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource primitives.heading */
import type { HTMLAttributes } from "react";
import {
  Text,
  type TextProps,
} from "@vrooli/react-component-library/Text/1";

export interface HeadingProps extends Omit<TextProps, "as"> {
  level?: 1 | 2 | 3 | 4 | 5 | 6;
}

export const Heading = withClassName(function Heading({
  level = 2,
  textStyle = "heading",
  ...props
}: HeadingProps & Omit<HTMLAttributes<HTMLHeadingElement>, "style">) {
  const Component = `h${level}` as keyof HTMLElementTagNameMap;
  return (
    <Text
      data-testid="primitives.heading"
      {...props}
      as={Component}
      textStyle={textStyle}
      data-heading-level={level}
    />
  );
});
