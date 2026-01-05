import * as React from "react";
import { cn } from "../../../lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "../../ui/card";
import { ScrollArea } from "../../ui/scroll-area";

interface ListPanelProps {
  title: string;
  count: number;
  toolbar?: React.ReactNode;
  headerActions?: React.ReactNode;
  loading?: boolean;
  empty?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  scrollHeight?: string;
}

export function ListPanel({
  title,
  count,
  toolbar,
  headerActions,
  loading,
  empty,
  children,
  className,
  scrollHeight = "h-[500px]",
}: ListPanelProps) {
  const hasItems = React.Children.count(children) > 0;

  return (
    <Card className={cn("lg:col-span-1", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">
            {title} ({count})
          </CardTitle>
          {headerActions}
        </div>
        {toolbar && <div className="mt-3">{toolbar}</div>}
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
            <p className="mt-3 text-sm">Loading...</p>
          </div>
        ) : !hasItems || count === 0 ? (
          empty || (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <p className="text-sm">No items found</p>
            </div>
          )
        ) : (
          <ScrollArea className={scrollHeight}>
            <div className="space-y-2">{children}</div>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  );
}
