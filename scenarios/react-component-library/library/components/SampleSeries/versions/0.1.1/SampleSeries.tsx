/**
 * @libraryId react-component-library:SampleSeries
 * @displayName SampleSeries
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
export default function SampleSeries({ values }: { values: number[] }) {
  return (
    <ol aria-label="Sample series">
      {values.map((v, i) => (
        <li key={i}>{v}</li>
      ))}
    </ol>
  );
}
