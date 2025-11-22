import { Package } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";

export function Deployments() {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold">Deployments</h1>
        <p className="text-slate-400 mt-1">
          Monitor active and past deployments
        </p>
      </div>

      {/* Empty State */}
      <Card>
        <CardHeader>
          <CardTitle>Active Deployments</CardTitle>
          <CardDescription>
            No active deployments
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <Package className="h-12 w-12 text-slate-600 mb-4" />
            <p className="text-slate-400 mb-4">No deployments yet</p>
            <p className="text-sm text-slate-500">
              Create a deployment profile and deploy it to see active deployments here
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
