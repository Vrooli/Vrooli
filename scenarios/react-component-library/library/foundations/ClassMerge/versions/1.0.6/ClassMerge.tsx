/**
 * @libraryId react-component-library:ClassMerge
 * @displayName Class Merge
 * @description A single conditional class composition helper with Tailwind conflict resolution; consumer overrides win over defaults.
 * @version 1.0.6
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/**
 * @vrooliComponentSource react-component-library:ClassMerge
 * @vrooliComponentSourceSlot foundations.class-merge
 * @version 1.0.0
 * @status released
 * @deps {"clsx":"^2.1.0","tailwind-merge":"^2.2.0"}
 */
import { clsx, type ClassValue } from "clsx";
import {
  Children,
  Fragment,
  cloneElement,
  forwardRef,
  isValidElement,
  type ComponentType,
  type Ref,
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
  if (!className || !isValidElement<{ children?: ReactNode; className?: unknown }>(node))
    return node;
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

/** Attach a consumer ref to the first host element in a composed tree. */
function addRef(node: ReactNode, ref: Ref<HTMLElement>): ReactNode {
  if (!ref || !isValidElement<{ children?: ReactNode; ref?: Ref<HTMLElement> }>(node)) return node;
  if (node.type === Fragment) {
    let attached = false;
    return cloneElement(
      node,
      undefined,
      Children.map(node.props.children, (child) => {
        if (attached) return child;
        const next = addRef(child, ref);
        attached = next !== child;
        return next;
      }),
    );
  }
  return cloneElement(node, { ref });
}

/** Wrap a component with the stable className seam used by linked adopters. */
export function withClassName<P extends object>(
  Component: ComponentType<P>,
): ComponentType<Omit<P, "className"> & { className?: string }>;
export function withClassName<C extends (...args: never[]) => ReactNode>(
  Component: C,
): ComponentType<Omit<Parameters<C>[0], "className"> & { className?: string }> & C;
export function withClassName<P extends object>(Component: ComponentType<P>) {
  return forwardRef<HTMLElement, Omit<P, "className"> & { className?: string }>(
    function ClassNameBoundary(forwardedProps, ref) {
      const { className, ...rest } = forwardedProps as Omit<P, "className"> & {
        className?: string;
      };
      return addRef(addClassName(<Component {...(rest as P)} />, className), ref);
    },
  );
}

export type { ClassValue };
