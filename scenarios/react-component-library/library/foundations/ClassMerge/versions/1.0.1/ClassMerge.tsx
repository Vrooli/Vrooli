/**
 * @libraryId react-component-library:ClassMerge
 * @displayName Class Merge
 * @description The single class composition primitive used by library assets; Tailwind conflicts resolve in favor of the consumer override.
 * @version 1.0.1
 * @tags ["foundations","styling","composition"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/**
 * @vrooliComponentSource foundations.class-merge
 * @version 1.0.0
 * @status released
 * @deps {"clsx":"^2.1.0","tailwind-merge":"^2.2.0"}
 */
import { clsx, type ClassValue } from "clsx";
import {
  Children,
  Fragment,
  cloneElement,
  createElement,
  isValidElement,
  type ComponentType,
  type ReactNode,
} from "react";
import { twMerge } from "tailwind-merge";

/**
 * Combines conditional class values and resolves Tailwind conflicts so the
 * consumer's explicit override wins over a component default.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/** Add a consumer-owned class to the first rendered host element. */
export function addClassName(node: ReactNode, className?: string): ReactNode {
  if (!className || !isValidElement(node)) return node;
  if (node.type === Fragment) {
    return cloneElement(
      node,
      undefined,
      Children.map(node.props.children, (child) => addClassName(child, className)),
    );
  }
  const existing = typeof node.props.className === "string" ? node.props.className : undefined;
  return cloneElement(node, { className: cn(existing, className) });
}

/** Wrap a component with the stable className seam used by linked adopters. */
export function withClassName<C extends ComponentType<any>>(Component: C): C {
  return function ClassNameBoundary(props: any) {
    const { className, ...rest } = props;
    return addClassName(createElement(Component, rest), className);
  } as C;
}

export type { ClassValue };
