/** @vrooliComponentSource react-component-library:TreeView */
export function TreeView({ nodes = [] }: { nodes?: string[] }) {
  return (
    <div role="tree" aria-label="Tree" style={{ display: "grid", gap: 4 }}>
      {nodes.map((node, index) => (
        <div
          role="treeitem"
          aria-level={1}
          aria-selected={index === 0}
          key={node}
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            minHeight: 44,
            borderRadius: 8,
            background:
              index === 0
                ? "var(--color-surface-muted, #f1f5f9)"
                : "transparent",
            paddingInline: 12,
          }}
        >
          {node}
        </div>
      ))}
    </div>
  );
}
