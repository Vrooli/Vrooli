import type { ReactNode } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";

export function Metric({ label, value, icon }: { label: string; value: number | undefined; icon: ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm uppercase text-app-muted-foreground">{icon}{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="font-mono text-3xl font-semibold">{value ?? "—"}</p>
      </CardContent>
    </Card>
  );
}
