/**
 * @libraryId react-component-library:HeroReadout
 * @displayName HeroReadout
 * @description Reusable command-display primitive
 * @version 0.1.1
 * @tags ["ambient-display","command-center"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import * as React from "react";
export interface HeroReadoutProps {
  value: React.ReactNode;
  label?: string;
  unit?: string;
}
export default function HeroReadout({ value, label, unit }: HeroReadoutProps) {
  return (
    <section className="rcl-hero-readout">
      <strong>{value}</strong>
      {unit && <span>{unit}</span>}
      {label && <small>{label}</small>}
    </section>
  );
}
