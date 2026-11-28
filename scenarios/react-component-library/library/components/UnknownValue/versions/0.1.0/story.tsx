import { UnknownValue } from "./UnknownValue";

export function Default() {
  return (
    <p style={{ margin: 0 }}>
      Last write: <UnknownValue label="Unknown" reason="The file has no timestamp." />
    </p>
  );
}
