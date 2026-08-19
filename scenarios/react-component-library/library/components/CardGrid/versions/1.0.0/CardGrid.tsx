/** @vrooliComponentSource patterns.card-grid */
import type { HTMLAttributes, ReactNode } from "react";
import { SurfaceProvider } from "../../../../foundations/Contracts/versions/1.0.0/Contracts";

const styles = `
[data-rcl-card-grid] { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 16rem), 1fr)); gap: var(--space-md); min-inline-size: 0; }
`;

export interface CardGridProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

export function CardGrid({ children, className, ...props }: CardGridProps) {
  return (
    <SurfaceProvider value={{ elevation: "raised" }}>
      <div
        {...props}
        className={className}
        data-rcl-card-grid
        data-rcl-context-elevation="raised"
      >
        <style
          data-rcl-card-grid-styles
          dangerouslySetInnerHTML={{ __html: styles }}
        />
        {children}
      </div>
    </SurfaceProvider>
  );
}
