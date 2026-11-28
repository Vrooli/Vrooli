import { useCollection } from "./useCollection";

export function Default() {
  const collection = useCollection(["Alpha", "Beta"], {
    getKey: (item) => item,
    selection: { mode: "multi" },
  });
  const rows = collection.rows;
  return (
    <div {...collection.getContainerProps()}>
      {rows.map((row) => (
        <div
          key={row}
          {...collection.getRowProps(row)}
          data-testid={`collection-option-${row.toLowerCase()}`}
        >
          {row}
        </div>
      ))}
    </div>
  );
}
