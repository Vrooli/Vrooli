import * as React from "react";
import { cn } from "../../../lib/utils";
import { useViewportSize } from "../../../hooks/useViewportSize";
import { DetailModal } from "./DetailModal";

interface MasterDetailLayoutProps {
  listPanel: React.ReactNode;
  detailPanel: React.ReactNode;
  selectedId: string | null;
  onDeselect: () => void;
  detailTitle: string;
  className?: string;
}

export function MasterDetailLayout({
  listPanel,
  detailPanel,
  selectedId,
  onDeselect,
  detailTitle,
  className,
}: MasterDetailLayoutProps) {
  const { isDesktop } = useViewportSize();

  return (
    <>
      <div className={cn("grid gap-6 lg:grid-cols-2", className)}>
        {listPanel}
        {/* Desktop detail panel - hidden on mobile via CSS */}
        <div className="hidden lg:block">{detailPanel}</div>
      </div>

      {/* Mobile detail modal */}
      {!isDesktop && (
        <DetailModal
          open={!!selectedId}
          onClose={onDeselect}
          title={detailTitle}
        >
          {detailPanel}
        </DetailModal>
      )}
    </>
  );
}
