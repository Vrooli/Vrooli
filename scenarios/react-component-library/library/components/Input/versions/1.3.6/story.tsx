import { Input } from "./Input";

/** Canonical text-entry specimen used by the experience contract. */
export function Text() {
  return <Input aria-label="Component name" placeholder="Component name" />;
}

/** Search semantics are part of the public field contract. */
export function Search() {
  return <Input type="search" aria-label="Search components" placeholder="Search components" />;
}

/** A disabled value remains visible while editing is prevented. */
export function Disabled() {
  return <Input aria-label="Saved component" defaultValue="Existing value" disabled readOnly />;
}
