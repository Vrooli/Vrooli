/**
 * @libraryId react-component-library:CardGrid
 * @displayName Card Grid
 * @description A restrained responsive card arrangement that establishes a raised surface context for repeated records without turning every page section into a floating panel.
 * @version 1.0.4
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CardGrid
 * @vrooliComponentSourceSlot patterns.card-grid */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { HTMLAttributes, ReactNode } from "react";
import { SurfaceProvider } from "@vrooli/react-component-library/Contracts/1";

const styles = `
[data-rcl-card-grid] { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 16rem), 1fr)); gap: var(--space-md); min-inline-size: 0; }
`;

export interface CardGridProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

export const CardGrid = withClassName(function CardGrid({
  children,
  className,
  ...props
}: CardGridProps) {
  return (
    <SurfaceProvider data-testid="patterns.card-grid" value={{ elevation: "raised" }}>
      <div {...props} className={className} data-rcl-card-grid data-rcl-context-elevation="raised">
        <style data-rcl-card-grid-styles dangerouslySetInnerHTML={{ __html: styles }} />
        {children}
      </div>
    </SurfaceProvider>
  );
});
