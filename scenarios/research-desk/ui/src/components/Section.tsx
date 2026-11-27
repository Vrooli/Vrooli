import type { ReactNode } from "react";
import { PageSection } from "@vrooli/react-component-library/page-section/0";
import { DescriptionList } from "@vrooli/react-component-library/DescriptionList/1.1.0";

/** Research Desk's section adapter delegates semantic layout to the shared asset. */
export function Section(props: { title: ReactNode; actions?: ReactNode; children: ReactNode; id?: string; note?: ReactNode; className?: string }) {
  return <PageSection {...props} />;
}

/** A definition list laid out for a phone: label above value, two columns on wide screens. */
export function Facts({ children }: { children: ReactNode }) {
  return <DescriptionList appearance="plain">{children}</DescriptionList>;
}
