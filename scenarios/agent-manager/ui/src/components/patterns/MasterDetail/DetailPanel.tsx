import * as React from "react";
import { cn } from "../../../lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "../../ui/card";

interface DetailPanelProps {
  title: string;
  empty?: React.ReactNode;
  headerActions?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  hasSelection?: boolean;
}

export function DetailPanel({
  title,
  empty,
  headerActions,
  children,
  className,
  hasSelection = true,
}: DetailPanelProps) {
  return (
    <Card className={cn("lg:col-span-1", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{title}</CardTitle>
          {headerActions}
        </div>
      </CardHeader>
      <CardContent>
        {!hasSelection ? (
          empty || (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <p className="text-sm">Select an item to view details</p>
            </div>
          )
        ) : (
          children
        )}
      </CardContent>
    </Card>
  );
}
