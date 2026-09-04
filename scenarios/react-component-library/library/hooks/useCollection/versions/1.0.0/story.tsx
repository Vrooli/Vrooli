import { useCollection } from "./useCollection";

export function Default() {
  const collection = useCollection(["Alpha", "Beta"], { getKey: (item) => item, selection: { mode: "multi" } });
  return <div {...collection.getContainerProps()}>{collection.rows.map((item) => <div key={item} {...collection.getRowProps(item)}>{item}</div>)}</div>;
}
