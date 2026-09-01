/**
 * @libraryId react-component-library:RollingNumber
 * @displayName RollingNumber
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
export default function RollingNumber({ value }: { value: number }) {
  return <span aria-live="polite">{new Intl.NumberFormat().format(value)}</span>;
}
