import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react';
import { GripVertical } from 'lucide-react';
import { ProcessorStatusButton } from './ProcessorStatusButton';
import { FilterToggleButton } from './FilterToggleButton';
import { RefreshCountdown } from './RefreshCountdown';
import { ProcessMonitor } from './ProcessMonitor';
import { LogsButton } from './LogsButton';
import { SettingsButton } from './SettingsButton';
import { NewTaskButton } from './NewTaskButton';

export function FloatingControls() {
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const dragOffset = useRef({ x: 0, y: 0 });
  const containerRef = useRef<HTMLDivElement | null>(null);

  // Position the tray in the bottom-right by default and keep it on-screen when resizing
  useEffect(() => {
    const updatePosition = () => {
      const container = containerRef.current;
      if (!container || typeof window === 'undefined') return;
      const margin = 16;
      const width = container.offsetWidth || 340;
      const height = container.offsetHeight || 120;
      const maxX = window.innerWidth - width - margin;
      const maxY = window.innerHeight - height - margin;

      setPosition((prev) => {
        const fallbackX = maxX;
        const fallbackY = maxY;
        const nextX = prev.x === 0 ? fallbackX : Math.min(Math.max(prev.x, margin), maxX);
        const nextY = prev.y === 0 ? fallbackY : Math.min(Math.max(prev.y, margin), maxY);
        return { x: nextX, y: nextY };
      });
    };

    updatePosition();
    window.addEventListener('resize', updatePosition);
    return () => window.removeEventListener('resize', updatePosition);
  }, []);

  useEffect(() => {
    if (!isDragging) return;

    const handlePointerMove = (event: PointerEvent) => {
      const container = containerRef.current;
      if (!container) return;

      const margin = 12;
      const width = container.offsetWidth || 340;
      const height = container.offsetHeight || 120;
      const maxX = window.innerWidth - width - margin;
      const maxY = window.innerHeight - height - margin;

      setPosition({
        x: Math.min(Math.max(event.clientX - dragOffset.current.x, margin), maxX),
        y: Math.min(Math.max(event.clientY - dragOffset.current.y, margin), maxY),
      });
    };

    const handlePointerUp = () => {
      setIsDragging(false);
      document.body.style.userSelect = '';
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', handlePointerUp);
    document.body.style.userSelect = 'none';

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerUp);
      document.body.style.userSelect = '';
    };
  }, [isDragging]);

  const handlePointerDown = (event: ReactPointerEvent) => {
    if (event.button !== 0) return;
    const target = event.target as HTMLElement;
    if (target.closest('button')) return;

    const container = containerRef.current;
    if (!container) return;

    const rect = container.getBoundingClientRect();
    dragOffset.current = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    };

    setIsDragging(true);
    event.preventDefault();
  };

  return (
    <div
      ref={containerRef}
      className="fixed z-30 bg-background/95 backdrop-blur-sm border rounded-lg shadow-lg px-3 py-2 max-w-[calc(100vw-2rem)] sm:max-w-2xl cursor-grab active:cursor-grabbing"
      style={{ left: position.x, top: position.y }}
      onPointerDown={handlePointerDown}
    >
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex items-center text-muted-foreground">
          <GripVertical className="h-4 w-4" />
        </div>

        <div className="flex items-center gap-2">
          <ProcessorStatusButton />
          <RefreshCountdown />
        </div>

        <div className="flex items-center gap-2 pl-3 border-l border-border/60">
          <FilterToggleButton />
          <ProcessMonitor />
        </div>

        <div className="flex items-center gap-2">
          <LogsButton />
          <SettingsButton />
          <NewTaskButton />
        </div>
      </div>
    </div>
  );
}
