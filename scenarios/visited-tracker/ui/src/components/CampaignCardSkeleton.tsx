import { Card, CardContent, CardFooter, CardHeader } from "./ui/card";

export function CampaignCardSkeleton() {
  return (
    <Card className="animate-pulse" data-testid="campaign-skeleton">
      <CardHeader className="pb-3">
        <div className="space-y-2">
          <div className="flex items-start justify-between gap-2">
            <div className="h-6 bg-slate-800 rounded w-2/3"></div>
            <div className="h-5 w-16 bg-slate-800 rounded"></div>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="h-3 w-3 bg-slate-800 rounded-full"></div>
            <div className="h-3 bg-slate-800 rounded w-24"></div>
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        <div className="mb-3 space-y-1.5">
          <div className="h-3 bg-slate-800 rounded w-full"></div>
          <div className="h-3 bg-slate-800 rounded w-3/4"></div>
        </div>

        <div className="grid grid-cols-3 gap-3 text-center">
          <div className="rounded-lg bg-slate-800/40 p-2">
            <div className="h-7 bg-slate-800 rounded mb-1"></div>
            <div className="h-3 bg-slate-800 rounded w-12 mx-auto"></div>
          </div>
          <div className="rounded-lg bg-slate-800/40 p-2">
            <div className="h-7 bg-slate-800 rounded mb-1"></div>
            <div className="h-3 bg-slate-800 rounded w-14 mx-auto"></div>
          </div>
          <div className="rounded-lg bg-slate-800/40 p-2">
            <div className="h-7 bg-slate-800 rounded mb-1"></div>
            <div className="h-3 bg-slate-800 rounded w-12 mx-auto"></div>
          </div>
        </div>

        <div className="mt-4 space-y-2">
          <div className="h-2 w-full bg-slate-800 rounded-full"></div>
          <div className="flex gap-1.5">
            <div className="h-5 bg-slate-800 rounded w-20"></div>
            <div className="h-5 bg-slate-800 rounded w-24"></div>
          </div>
        </div>

        <div className="mt-3">
          <div className="h-3 bg-slate-800 rounded w-32"></div>
        </div>
      </CardContent>

      <CardFooter className="justify-end gap-2 pt-3">
        <div className="h-8 w-16 bg-slate-800 rounded"></div>
        <div className="h-8 w-20 bg-slate-800 rounded"></div>
      </CardFooter>
    </Card>
  );
}
