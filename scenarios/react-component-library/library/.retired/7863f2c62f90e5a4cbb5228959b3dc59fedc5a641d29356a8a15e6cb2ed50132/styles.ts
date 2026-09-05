export const resizeHandleStyles = `
[data-rcl-resize-handle] {
  --rcl-resize-handle-size: var(--space-xs, 12px);
  position: absolute;
  z-index: var(--layer-sticky, 100);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  background: transparent;
  forced-color-adjust: none;
}
[data-rcl-resize-handle][data-axis="inline"] { inset-block: 0; inline-size: var(--rcl-resize-handle-size); cursor: col-resize; }
[data-rcl-resize-handle][data-axis="inline"][data-edge="end"] { inset-inline-end: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][data-axis="inline"][data-edge="start"] { inset-inline-start: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][data-axis="block"] { inset-inline: 0; block-size: var(--rcl-resize-handle-size); cursor: row-resize; }
[data-rcl-resize-handle][data-axis="block"][data-edge="end"] { inset-block-end: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][data-axis="block"][data-edge="start"] { inset-block-start: calc(var(--rcl-resize-handle-size) / -2); }
[data-rcl-resize-handle][aria-disabled="true"] { cursor: default; }

[data-rcl-resize-handle] .rcl-resize-handle__bar {
  display: block;
  background: transparent;
  border-radius: var(--radius-control, 0.375rem);
  transition: background-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)),
              transform var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1));
}
[data-rcl-resize-handle][data-axis="inline"] .rcl-resize-handle__bar { inline-size: 1px; block-size: 100%; }
[data-rcl-resize-handle][data-axis="block"] .rcl-resize-handle__bar { block-size: 1px; inline-size: 100%; }

[data-rcl-resize-handle]:hover .rcl-resize-handle__bar,
[data-rcl-resize-handle][data-dragging="true"] .rcl-resize-handle__bar { background: var(--color-primary, #2563eb); }
[data-rcl-resize-handle][data-axis="inline"]:hover .rcl-resize-handle__bar,
[data-rcl-resize-handle][data-axis="inline"][data-dragging="true"] .rcl-resize-handle__bar { transform: scaleX(2); }
[data-rcl-resize-handle][data-axis="block"]:hover .rcl-resize-handle__bar,
[data-rcl-resize-handle][data-axis="block"][data-dragging="true"] .rcl-resize-handle__bar { transform: scaleY(2); }

[data-rcl-resize-handle][data-snapped="true"] .rcl-resize-handle__bar { background: var(--color-success, #16a34a); }
[data-rcl-resize-handle][data-collapsed="true"] .rcl-resize-handle__bar { background: var(--color-border, #cbd5e1); }
[data-rcl-resize-handle][aria-disabled="true"] .rcl-resize-handle__bar { background: transparent; }

`;
