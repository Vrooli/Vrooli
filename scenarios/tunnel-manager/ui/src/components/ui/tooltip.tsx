import { useState, useRef, useEffect, useId, useCallback } from "react";

interface TooltipProps {
  content: string;
  children: React.ReactNode;
  className?: string;
}

export function Tooltip({ content, children, className = "" }: TooltipProps) {
  const [visible, setVisible] = useState(false);
  const [position, setPosition] = useState<"top" | "bottom">("top");
  const triggerRef = useRef<HTMLSpanElement>(null);
  const tooltipId = useId();

  useEffect(() => {
    if (visible && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      setPosition(rect.top < 60 ? "bottom" : "top");
    }
  }, [visible]);

  // Close tooltip when tapping outside on touch devices
  useEffect(() => {
    if (!visible) return;
    function handleOutside(e: PointerEvent) {
      if (triggerRef.current && !triggerRef.current.contains(e.target as Node)) {
        setVisible(false);
      }
    }
    document.addEventListener("pointerdown", handleOutside);
    return () => document.removeEventListener("pointerdown", handleOutside);
  }, [visible]);

  // Toggle on tap (pointerUp with no drag), show on hover/focus
  const handleClick = useCallback((e: React.MouseEvent) => {
    // Prevent click from bubbling to document pointerdown handler
    e.stopPropagation();
    setVisible((v) => !v);
  }, []);

  return (
    <span
      ref={triggerRef}
      className={`relative inline-flex ${className}`}
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
      onFocus={() => setVisible(true)}
      onBlur={() => setVisible(false)}
      onClick={handleClick}
      aria-describedby={visible ? tooltipId : undefined}
    >
      {children}
      {visible && (
        <span
          id={tooltipId}
          role="tooltip"
          className={`absolute left-1/2 z-50 -translate-x-1/2 whitespace-normal rounded-lg border border-white/10 bg-slate-800 px-3 py-2 text-xs font-normal text-slate-200 shadow-lg max-w-[240px] w-max ${
            position === "top" ? "bottom-full mb-2" : "top-full mt-2"
          }`}
        >
          {content}
        </span>
      )}
    </span>
  );
}
