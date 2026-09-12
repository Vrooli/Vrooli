import { useCollection } from "./useCollection";

export function Default() {
  const collection = useCollection(["Alpha", "Beta"], { getKey: (item) => item, selection: { mode: "multi" } });
  return <div {...collection.getContainerProps()}><div {...collection.getRowProps(collection.rows[0]!)}>Alpha</div><div {...collection.getRowProps(collection.rows[1]!)}>Beta</div></div>;
}
