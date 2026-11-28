import { DescriptionList } from "./DescriptionList";

export function Default() {
  return (
    <div style={{ display: "grid", gap: "var(--space-md)", maxInlineSize: "42rem" }}>
      <DescriptionList entries={[{ term: "Owner", description: "Platform" }]} />
      <DescriptionList appearance="plain">
        <div>
          <dt>Last research write</dt>
          <dd>12 minutes ago</dd>
        </div>
      </DescriptionList>
    </div>
  );
}

export function Empty() {
  return <DescriptionList entries={[]} />;
}
