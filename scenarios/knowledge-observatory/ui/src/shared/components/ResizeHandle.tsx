import { useCallback, useRef, useEffect } from "react";

export type ResizeHandleProps = {
  /** Direction of resize - vertical means column resize (left/right) */
  direction?: "vertical" | "horizontal";
  /** Called during drag with the delta in pixels */
  onResize: (delta: number) => void;
  /** Called when drag ends */
  onResizeEnd?: () => void;
  /** Additional CSS classes */
  className?: string;
};

export function ResizeHandle({
  direction = "vertical",
  onResize,
  onResizeEnd,
  className = "",
}: ResizeHandleProps) {
  const isDragging = useRef(false);
  const lastPosition = useRef(0);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isDragging.current = true;
    lastPosition.current = direction === "vertical" ? e.clientX : e.clientY;
    document.body.style.cursor = direction === "vertical" ? "col-resize" : "row-resize";
    document.body.style.userSelect = "none";
  }, [direction]);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging.current) return;
      const currentPosition = direction === "vertical" ? e.clientX : e.clientY;
      const delta = currentPosition - lastPosition.current;
      lastPosition.current = currentPosition;
      onResize(delta);
    };

    const handleMouseUp = () => {
      if (!isDragging.current) return;
      isDragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
      onResizeEnd?.();
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [direction, onResize, onResizeEnd]);

  const baseClass = direction === "vertical" ? "ko-resize-handle" : "ko-resize-handle-horizontal";

  return (
    <div
      className={`${baseClass} ${className}`}
      onMouseDown={handleMouseDown}
      role="separator"
      aria-orientation={direction}
      aria-label={direction === "vertical" ? "Resize columns" : "Resize rows"}
      tabIndex={0}
    />
  );
}
