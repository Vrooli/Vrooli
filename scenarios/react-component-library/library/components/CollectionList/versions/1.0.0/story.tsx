import { CollectionList } from "./CollectionList";

export function Default() {
  return <CollectionList items={[{ id: "1", name: "Example" }]} getKey={(item) => item.id} label="Examples" renderItem={(item) => <span>{item.name}</span>} />;
}
