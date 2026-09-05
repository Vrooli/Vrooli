import {
  cloneElement,
  useId,
  useRef,
  useState,
  type ReactElement,
  type ReactNode,
} from "react";
import { cn } from "../../lib/utils";
import { Popover } from "./popover";

type TooltipChildProps = {
  ref?: React.Ref<HTMLElement | null>;
  "aria-describedby"?: string;
  onMouseEnter?: (event: React.MouseEvent<Element>) => void;
  onMouseLeave?: (event: React.MouseEvent<Element>) => void;
  onFocus?: (event: React.FocusEvent<Element>) => void;
  onBlur?: (event: React.FocusEvent<Element>) => void;
  onKeyDown?: (event: React.KeyboardEvent<Element>) => void;
};

interface TooltipProps {
  content: ReactNode;
  children: ReactElement<TooltipChildProps>;
  delayMs?: number;
  className?: string;
  testId?: string;
}

export function Tooltip({
  content,
  children,
  delayMs = 250,
  className,
  testId,
}: TooltipProps) {
  const id = useId();
  const triggerRef = useRef<HTMLElement | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [isOpen, setIsOpen] = useState(false);

  const clearTimer = () => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  };

  const show = () => {
    clearTimer();
    timerRef.current = setTimeout(() => setIsOpen(true), delayMs);
  };

  const hide = () => {
    clearTimer();
    setIsOpen(false);
  };

  const handleKeyDown = (event: React.KeyboardEvent) => {
    children.props.onKeyDown?.(event);
    if (event.key === "Escape") {
      hide();
    }
  };

  const trigger = cloneElement(children, {
    ref: (node: HTMLElement | null) => {
      triggerRef.current = node;
      const childRef = children.props.ref;
      if (typeof childRef === "function") {
        childRef(node);
      } else if (childRef && "current" in childRef) {
        (childRef as React.MutableRefObject<HTMLElement | null>).current = node;
      }
    },
    "aria-describedby": isOpen ? id : undefined,
    onMouseEnter: (event: React.MouseEvent) => {
      children.props.onMouseEnter?.(event);
      show();
    },
    onMouseLeave: (event: React.MouseEvent) => {
      children.props.onMouseLeave?.(event);
      hide();
    },
    onFocus: (event: React.FocusEvent) => {
      children.props.onFocus?.(event);
      show();
    },
    onBlur: (event: React.FocusEvent) => {
      children.props.onBlur?.(event);
      hide();
    },
    onKeyDown: handleKeyDown,
  });

  return (
    <>
      {trigger}
      <Popover
        isOpen={isOpen}
        onClose={hide}
        triggerRef={triggerRef}
        placement="top-start"
        offset={6}
        className={cn(
          "min-w-0 max-w-xs rounded-md border-slate-700 bg-slate-950 px-2.5 py-1.5 text-xs leading-relaxed text-slate-100 shadow-xl",
          className,
        )}
        testId={testId}
      >
        <span id={id} role="tooltip">
          {content}
        </span>
      </Popover>
    </>
  );
}
