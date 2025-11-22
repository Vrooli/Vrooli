import { useState } from "react";
import { useAdoptions, useComponents, useCreateAdoption } from "../lib/api-client";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../components/ui/dialog";
import { GitBranch, Package, CheckCircle2, AlertCircle, Clock } from "lucide-react";

export function Adoptions() {
  const { data: adoptions, isLoading } = useAdoptions();
  const { data: components } = useComponents();
  const createAdoption = useCreateAdoption();

  const [selectedComponent, setSelectedComponent] = useState("");
  const [scenarioName, setScenarioName] = useState("");
  const [adoptedPath, setAdoptedPath] = useState("");

  const handleCreateAdoption = async () => {
    if (!selectedComponent || !scenarioName || !adoptedPath) return;

    try {
      await createAdoption.mutateAsync({
        componentId: selectedComponent,
        scenarioName,
        adoptedPath,
      });
      setSelectedComponent("");
      setScenarioName("");
      setAdoptedPath("");
    } catch (error) {
      console.error("Failed to create adoption:", error);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "current":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />;
      case "behind":
        return <Clock className="h-4 w-4 text-amber-500" />;
      case "modified":
        return <AlertCircle className="h-4 w-4 text-blue-500" />;
      default:
        return <AlertCircle className="h-4 w-4 text-slate-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "current":
        return <Badge variant="success">Current</Badge>;
      case "behind":
        return <Badge variant="warning">Behind</Badge>;
      case "modified":
        return <Badge variant="secondary">Modified</Badge>;
      default:
        return <Badge variant="outline">Unknown</Badge>;
    }
  };

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mb-4 inline-flex h-12 w-12 animate-spin items-center justify-center rounded-full border-4 border-blue-600 border-t-transparent" />
          <p className="text-slate-400">Loading adoptions...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Component Adoptions</h2>
          <p className="text-sm text-slate-400">
            Track which scenarios use which components
          </p>
        </div>
        <Dialog>
          <DialogTrigger asChild>
            <Button>
              <GitBranch className="mr-2 h-4 w-4" />
              New Adoption
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Adoption Record</DialogTitle>
              <DialogDescription>
                Track a component adoption in a scenario
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="component">Component</Label>
                <select
                  id="component"
                  className="flex h-10 w-full rounded-md border border-white/10 bg-slate-900/50 px-3 py-2 text-sm text-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                  value={selectedComponent}
                  onChange={(e) => setSelectedComponent(e.target.value)}
                >
                  <option value="">Select a component...</option>
                  {components?.map((comp) => (
                    <option key={comp.id} value={comp.id}>
                      {comp.displayName} (v{comp.version})
                    </option>
                  ))}
                </select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="scenario">Scenario Name</Label>
                <Input
                  id="scenario"
                  placeholder="e.g., landing-manager"
                  value={scenarioName}
                  onChange={(e) => setScenarioName(e.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="path">Adopted File Path</Label>
                <Input
                  id="path"
                  placeholder="e.g., /ui/src/components/Button.tsx"
                  value={adoptedPath}
                  onChange={(e) => setAdoptedPath(e.target.value)}
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                onClick={handleCreateAdoption}
                disabled={
                  !selectedComponent ||
                  !scenarioName ||
                  !adoptedPath ||
                  createAdoption.isPending
                }
              >
                {createAdoption.isPending ? "Creating..." : "Create Adoption"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {!adoptions || adoptions.length === 0 ? (
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <Package className="mx-auto mb-4 h-12 w-12 text-slate-600" />
            <h3 className="mb-2 text-lg font-semibold">No adoptions tracked yet</h3>
            <p className="text-sm text-slate-400">
              Start tracking component adoptions across scenarios
            </p>
          </div>
        </div>
      ) : (
        <div className="grid gap-4">
          {adoptions.map((adoption) => (
            <Card key={adoption.id}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div>
                    <CardTitle className="flex items-center space-x-2">
                      <span>{adoption.componentLibraryId}</span>
                      {getStatusIcon(adoption.status)}
                    </CardTitle>
                    <CardDescription>
                      Adopted in {adoption.scenarioName}
                    </CardDescription>
                  </div>
                  {getStatusBadge(adoption.status)}
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid gap-2 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-slate-400">Version</span>
                    <span className="font-mono">v{adoption.version}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-slate-400">Path</span>
                    <span className="truncate font-mono text-xs">
                      {adoption.adoptedPath}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-slate-400">Adopted</span>
                    <span className="text-xs">
                      {new Date(adoption.createdAt).toLocaleDateString()}
                    </span>
                  </div>
                  {adoption.status === "behind" && (
                    <div className="mt-2">
                      <Button variant="outline" size="sm" className="w-full">
                        Update to Latest Version
                      </Button>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
