import type { AnchorHTMLAttributes, ButtonHTMLAttributes, ReactElement, ReactNode } from "react";
import { Button as RclButton } from "@vrooli/react-component-library/Button/2.2.14";
import { Link as RclLink } from "@vrooli/react-component-library/Link/1.0.3";
import { cn } from "../../lib/utils";

type LegacyVariant = "default" | "outline" | "ghost" | "destructive";
type LegacySize = "default" | "sm";
type ButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color" | "children"> & {
  children: ReactNode;
  variant?: LegacyVariant;
  size?: LegacySize;
  asChild?: false;
};
type ButtonLinkProps = Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "children"> & {
  children: ReactElement<AnchorHTMLAttributes<HTMLAnchorElement>, "a">;
  variant?: LegacyVariant;
  size?: LegacySize;
  asChild: true;
};

const variantMap = { default: "primary", outline: "secondary", ghost: "ghost", destructive: "danger" } as const;
const buttonClass = "inline-flex items-center justify-center rounded-full text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/50 disabled:pointer-events-none disabled:opacity-60";
const variantClass = { default: "bg-slate-50 text-slate-900 hover:bg-white", outline: "border border-white/30 text-white hover:bg-white/10", ghost: "text-slate-400 hover:text-slate-200 hover:bg-white/5", destructive: "bg-red-600 text-white hover:bg-red-500" } as const;
const sizeClass = { default: "h-11 px-5", sm: "h-9 px-4" } as const;

export function Button(props: ButtonProps | ButtonLinkProps) {
  const { variant = "default", size = "default", className, children } = props;
  if (props.asChild) {
    if (children.type !== "a") {
      throw new Error("Button asChild currently supports a single anchor child.");
    }
    const rclClass = cn(buttonClass, variantClass[variant], sizeClass[size], className, children.props.className);
    // Link retains native anchor navigation and keyboard behavior while carrying
    // the caller's legacy button appearance for this compatibility-only path.
    return <RclLink {...children.props} className={rclClass}>{children.props.children}</RclLink>;
  }
  const { asChild: _asChild, ...buttonProps } = props;
  return <RclButton {...buttonProps} variant={variantMap[variant]} size={size === "default" ? "lg" : "sm"}>{children}</RclButton>;
}
