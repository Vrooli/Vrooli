import type { ReactNode } from "react";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useBreakpoint } from "../hooks/useBreakpoint";

type Props = { headingId: string; eyebrow: string; title: string; description: string; leading?: ReactNode; className?: string };

function DesktopHeader(props: Props) {
  return <PageHeader {...props} className={`${props.className ?? ""} adaptive-header-desktop`} />;
}

function MobileHeader(props: Props) {
  return <header className={`${props.className ?? ""} adaptive-header-mobile planner-page-header`} aria-labelledby={props.headingId}>
    <div className="adaptive-mobile-heading">{props.leading}<div><p className="eyebrow">{props.eyebrow}</p><h1 id={props.headingId}>{props.title}</h1></div></div>
    <p className="adaptive-mobile-description">{props.description}</p>
  </header>;
}

/** Deliberate component divergence for mobile; CSS remains responsible only for polish. */
export function AdaptivePageHeader(props: Props) {
  const { isMobile } = useBreakpoint();
  return isMobile ? <MobileHeader {...props} /> : <DesktopHeader {...props} />;
}

