import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Cloud, AlertCircle, CheckCircle, XCircle, RefreshCw, Plus, Info, Trash2 } from "lucide-react";
import { useState } from "react";
import {
  fetchDistributionTargets,
  fetchDistributionConfigPath,
  createDistributionTarget,
  updateDistributionTarget,
  deleteDistributionTarget,
  testDistributionTarget,
  validateDistributionTargets,
  type DistributionTarget,
  type DistributionTargetsResponse,
  type DistributionValidationResult,
  type DistributionTestResult,
} from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Button } from "../ui/button";
import { DistributionTargetForm } from "./DistributionTargetForm";
import { DistributionTargetCard } from "./DistributionTargetCard";
import { cn } from "../../lib/utils";

export function DistributionPage() {
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [editingTarget, setEditingTarget] = useState<DistributionTarget | null>(null);
  const [testResults, setTestResults] = useState<Record<string, DistributionTestResult>>({});
  const [validationResult, setValidationResult] = useState<DistributionValidationResult | null>(null);

  // Fetch targets
  const {
    data: targetsData,
    isLoading: targetsLoading,
    refetch: refetchTargets
  } = useQuery<DistributionTargetsResponse>({
    queryKey: ["distribution-targets"],
    queryFn: fetchDistributionTargets,
  });

  // Fetch config path
  const { data: configPathData } = useQuery<{ path: string }>({
    queryKey: ["distribution-config-path"],
    queryFn: fetchDistributionConfigPath,
  });

  // Create mutation
  const createMutation = useMutation({
    mutationFn: createDistributionTarget,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["distribution-targets"] });
      setShowForm(false);
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ name, target }: { name: string; target: Partial<DistributionTarget> }) =>
      updateDistributionTarget(name, target),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["distribution-targets"] });
      setEditingTarget(null);
      setShowForm(false);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: deleteDistributionTarget,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["distribution-targets"] });
    },
  });

  // Test mutation
  const testMutation = useMutation({
    mutationFn: testDistributionTarget,
    onSuccess: (result) => {
      setTestResults((prev) => ({ ...prev, [result.target_name]: result }));
    },
  });

  // Validate mutation
  const validateMutation = useMutation({
    mutationFn: () => validateDistributionTargets(),
    onSuccess: setValidationResult,
  });

  const handleCreate = (target: DistributionTarget) => {
    createMutation.mutate(target);
  };

  const handleUpdate = (target: DistributionTarget) => {
    if (!editingTarget) return;
    updateMutation.mutate({ name: editingTarget.name, target });
  };

  const handleDelete = (name: string) => {
    if (confirm(`Are you sure you want to delete the target "${name}"?`)) {
      deleteMutation.mutate(name);
    }
  };

  const handleTest = (name: string) => {
    testMutation.mutate(name);
  };

  const handleEdit = (target: DistributionTarget) => {
    setEditingTarget(target);
    setShowForm(true);
  };

  const handleCancelForm = () => {
    setShowForm(false);
    setEditingTarget(null);
  };

  const targets = targetsData?.targets || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <Card className="border-slate-800/80 bg-slate-900/70">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Cloud className="h-6 w-6 text-blue-400" />
            Distribution Services
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-slate-300">
            Configure S3-compatible storage targets for distributing your desktop application installers.
            Targets are stored globally and can be used across all scenarios.
          </p>

          <div className="flex items-center justify-between">
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => refetchTargets()}
                disabled={targetsLoading}
              >
                <RefreshCw className={cn("h-4 w-4 mr-1", targetsLoading && "animate-spin")} />
                Refresh
              </Button>
              <Button
                variant="outline"
                onClick={() => validateMutation.mutate()}
                disabled={validateMutation.isPending || targets.length === 0}
              >
                {validateMutation.isPending ? (
                  <RefreshCw className="h-4 w-4 mr-1 animate-spin" />
                ) : (
                  <CheckCircle className="h-4 w-4 mr-1" />
                )}
                Validate All
              </Button>
            </div>
            <Button
              onClick={() => {
                setEditingTarget(null);
                setShowForm(true);
              }}
              disabled={showForm}
            >
              <Plus className="h-4 w-4 mr-1" />
              Add Target
            </Button>
          </div>

          {configPathData?.path && (
            <p className="text-xs text-slate-500">
              Config file: <code className="text-slate-400">{configPathData.path}</code>
            </p>
          )}
        </CardContent>
      </Card>

      {/* Quick Start Guide */}
      <DistributionPrimer />

      {/* Validation Results */}
      {validationResult && (
        <ValidationResultsCard result={validationResult} onDismiss={() => setValidationResult(null)} />
      )}

      {/* Add/Edit Form */}
      {showForm && (
        <Card className="border-blue-800/50 bg-slate-900/70">
          <CardHeader>
            <CardTitle className="text-lg">
              {editingTarget ? `Edit Target: ${editingTarget.name}` : "Add Distribution Target"}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <DistributionTargetForm
              initialTarget={editingTarget}
              onSubmit={editingTarget ? handleUpdate : handleCreate}
              onCancel={handleCancelForm}
              isSubmitting={createMutation.isPending || updateMutation.isPending}
              error={(createMutation.error || updateMutation.error)?.message}
            />
          </CardContent>
        </Card>
      )}

      {/* Targets List */}
      <Card className="border-slate-800/80 bg-slate-900/70">
        <CardHeader>
          <CardTitle className="text-lg">Distribution Targets</CardTitle>
        </CardHeader>
        <CardContent>
          {targetsLoading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 animate-spin text-slate-400" />
            </div>
          ) : targets.length === 0 ? (
            <div className="text-center py-8">
              <Cloud className="h-12 w-12 mx-auto mb-4 text-slate-600" />
              <p className="text-slate-400">No distribution targets configured.</p>
              <p className="text-sm text-slate-500 mt-1">
                Add a target to start distributing your desktop apps.
              </p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {targets.map((target) => (
                <DistributionTargetCard
                  key={target.name}
                  target={target}
                  testResult={testResults[target.name]}
                  onEdit={() => handleEdit(target)}
                  onDelete={() => handleDelete(target.name)}
                  onTest={() => handleTest(target.name)}
                  isTesting={testMutation.isPending && testMutation.variables === target.name}
                  isDeleting={deleteMutation.isPending && deleteMutation.variables === target.name}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function DistributionPrimer() {
  return (
    <Card className="border-slate-800/80 bg-slate-900/70">
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <Info className="h-5 w-5 text-blue-300" />
          Getting Started
        </CardTitle>
      </CardHeader>
      <CardContent className="grid gap-6 md:grid-cols-2">
        <div className="space-y-3">
          <p className="text-sm text-slate-300">
            Distribution targets let you automatically upload built installers to cloud storage
            for easy sharing and deployment.
          </p>
          <ul className="list-disc space-y-2 pl-5 text-sm text-slate-200">
            <li>
              <strong>S3</strong>: AWS S3 buckets - requires region
            </li>
            <li>
              <strong>R2</strong>: Cloudflare R2 storage - requires endpoint URL
            </li>
            <li>
              <strong>S3-compatible</strong>: MinIO, DigitalOcean Spaces, etc.
            </li>
          </ul>
        </div>
        <div className="space-y-3">
          <p className="text-sm font-semibold text-slate-100">Credential Setup</p>
          <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 space-y-2 text-xs text-slate-300">
            <p>
              Credentials are stored as <strong>environment variable names</strong>, not actual values.
              Set these variables in your environment before uploading:
            </p>
            <pre className="bg-black/40 p-2 rounded text-[11px] overflow-x-auto">
{`# Example for R2
export R2_ACCESS_KEY_ID="your-access-key"
export R2_SECRET_ACCESS_KEY="your-secret-key"`}
            </pre>
            <p className="text-slate-400">
              This keeps your actual credentials secure and out of config files.
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function ValidationResultsCard({
  result,
  onDismiss,
}: {
  result: DistributionValidationResult;
  onDismiss: () => void;
}) {
  const hasErrors = Object.values(result.targets).some((t) => !t.valid);

  return (
    <Card
      className={cn(
        "border",
        result.valid ? "border-green-800/50 bg-green-950/20" : "border-red-800/50 bg-red-950/20"
      )}
    >
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-base flex items-center gap-2">
          {result.valid ? (
            <CheckCircle className="h-5 w-5 text-green-400" />
          ) : (
            <XCircle className="h-5 w-5 text-red-400" />
          )}
          Validation {result.valid ? "Passed" : "Failed"}
        </CardTitle>
        <Button variant="ghost" size="sm" onClick={onDismiss}>
          <Trash2 className="h-4 w-4" />
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {Object.entries(result.targets).map(([name, targetResult]) => (
          <div
            key={name}
            className={cn(
              "p-3 rounded-lg border",
              targetResult.valid
                ? "border-green-800/30 bg-green-950/30"
                : "border-red-800/30 bg-red-950/30"
            )}
          >
            <div className="flex items-center gap-2 mb-2">
              {targetResult.valid ? (
                <CheckCircle className="h-4 w-4 text-green-400" />
              ) : (
                <XCircle className="h-4 w-4 text-red-400" />
              )}
              <span className="font-medium text-slate-200">{name}</span>
            </div>

            {targetResult.errors.length > 0 && (
              <ul className="space-y-1 text-sm text-red-300">
                {targetResult.errors.map((err, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                    {err}
                  </li>
                ))}
              </ul>
            )}

            {targetResult.warnings.length > 0 && (
              <ul className="space-y-1 text-sm text-amber-300 mt-2">
                {targetResult.warnings.map((warn, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                    {warn}
                  </li>
                ))}
              </ul>
            )}

            {targetResult.valid && targetResult.errors.length === 0 && targetResult.warnings.length === 0 && (
              <p className="text-sm text-green-300">All checks passed</p>
            )}
          </div>
        ))}

        {hasErrors && (
          <p className="text-sm text-slate-400 mt-2">
            Fix the errors above and run validation again.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
