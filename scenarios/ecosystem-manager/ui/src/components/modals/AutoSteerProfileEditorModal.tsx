/**
 * AutoSteerProfileEditorModal
 * Create/edit an Auto Steer *objective profile*: the controller's objective
 * function (dimension weights + targets), the skills it may select, and its
 * loop budget. There is no phase list — the controller derives the path.
 * See docs/concepts/CONTROL-MODEL.md.
 */

import { useEffect, useMemo, useState } from 'react';
import { AlertCircle, Save, X } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  useAutoSteerDimensions,
  useCreateAutoSteerProfile,
  useUpdateAutoSteerProfile,
} from '@/hooks/useAutoSteer';
import { useMergedSkillNames } from '@/hooks/usePromptFiles';
import type { AutoSteerProfile } from '@/types/api';
import { TagEditor } from './autosteer/TagEditor';
import { DimensionWeightEditor } from './autosteer/DimensionWeightEditor';
import { AllowedSkillsPicker } from './autosteer/AllowedSkillsPicker';
import { getApiErrorMessage, normalizeSkillId } from '@/lib/utils';

interface AutoSteerProfileEditorModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profile: AutoSteerProfile | null; // null = create new, non-null = edit existing
  prefillData?: Partial<AutoSteerProfile>; // For templates / duplication
}

interface SaveError {
  title: string;
  detail?: string;
  recovery?: string;
}

// Severity tiers offered for the "no finding above" target. Maps to the
// validMaxOpenSeverity set on the API; "" means no severity gate.
const SEVERITY_OPTIONS: Array<{ value: string; label: string }> = [
  { value: 'none', label: 'No severity gate' },
  { value: 'info', label: 'Info (allow info findings)' },
  { value: 'warning', label: 'Warning (allow warnings)' },
  { value: 'error', label: 'Error (allow errors)' },
	{ value: 'blocker', label: 'Blocker (allow blockers)' },
];

const DEFAULT_OBJECTIVE: AutoSteerProfile['objective'] = {
	dimension_weights: {},
	targets: { max_open_severity: 'warning', operational_targets_pct: 0 },
};

const DEFAULT_BUDGET: AutoSteerProfile['budget'] = {
	max_iterations: 40,
	diminishing_returns_floor: 0.02,
	reaudit_cadence: 5,
};

function emptyProfile(): Partial<AutoSteerProfile> {
	return {
		name: '',
		description: '',
		tags: [],
		objective: structuredClone(DEFAULT_OBJECTIVE),
		allowed_skills: [],
		budget: { ...DEFAULT_BUDGET },
		audit_preset: 'comprehensive',
	};
}

function hydrate(source: Partial<AutoSteerProfile>): Partial<AutoSteerProfile> {
	const base = emptyProfile();
	const clone = JSON.parse(JSON.stringify(source)) as Partial<AutoSteerProfile>;
	const sourceObjective = clone.objective ?? DEFAULT_OBJECTIVE;
	return {
		...base,
		...clone,
		objective: {
			dimension_weights: sourceObjective.dimension_weights,
			targets: {
				max_open_severity: sourceObjective.targets.max_open_severity ?? DEFAULT_OBJECTIVE.targets.max_open_severity,
				operational_targets_pct: sourceObjective.targets.operational_targets_pct ?? 0,
			},
		},
		allowed_skills: (clone.allowed_skills ?? []).map((s) => normalizeSkillId(s)).filter(Boolean),
		budget: { ...DEFAULT_BUDGET, ...clone.budget },
		tags: clone.tags ?? [],
	};
}

export function AutoSteerProfileEditorModal({
  open,
  onOpenChange,
  profile,
  prefillData,
}: AutoSteerProfileEditorModalProps) {
  const createProfile = useCreateAutoSteerProfile();
	const updateProfile = useUpdateAutoSteerProfile();
	const { data: dimensions = [], isLoading: dimensionsLoading } = useAutoSteerDimensions();
	const { data: skillOptions, isLoading: skillsLoading } = useMergedSkillNames();

  const [local, setLocal] = useState<Partial<AutoSteerProfile>>(emptyProfile());
  const [saveError, setSaveError] = useState<SaveError | null>(null);

  useEffect(() => {
    if (!open) return;
    if (profile) {
      setLocal(hydrate(profile));
    } else if (prefillData) {
      setLocal(hydrate(prefillData));
    } else {
      setLocal(emptyProfile());
    }
    setSaveError(null);
  }, [open, profile, prefillData]);

	const objective = local.objective ?? DEFAULT_OBJECTIVE;
	const budget = local.budget ?? DEFAULT_BUDGET;

  const skillPickerOptions = useMemo(
    () => skillOptions.map((s) => ({ id: s.id, name: s.name })),
    [skillOptions],
  );

	const setObjective = (next: Partial<AutoSteerProfile['objective']>) => {
		setLocal((prev) => ({
			...prev,
			objective: { ...(prev.objective ?? DEFAULT_OBJECTIVE), ...next },
		}));
		if (saveError) setSaveError(null);
	};

  const setTargets = (next: Partial<AutoSteerProfile['objective']['targets']>) => {
    setObjective({ targets: { ...objective.targets, ...next } });
  };

	const setBudget = (next: Partial<AutoSteerProfile['budget']>) => {
		setLocal((prev) => ({
			...prev,
			budget: { ...(prev.budget ?? DEFAULT_BUDGET), ...next },
		}));
		if (saveError) setSaveError(null);
	};

  const updateField = (field: keyof AutoSteerProfile, value: unknown) => {
    setLocal((prev) => ({ ...prev, [field]: value }));
    if (saveError) setSaveError(null);
  };

	const buildPayload = (): AutoSteerProfile => {
		const allowed = (local.allowed_skills ?? []).map((s) => normalizeSkillId(s)).filter(Boolean);
		const name = local.name?.trim() ?? '';
		return {
			...(local as AutoSteerProfile),
			name,
      description: local.description?.trim() ?? '',
      tags: local.tags ?? [],
      allowed_skills: Array.from(new Set(allowed)),
      objective: {
        dimension_weights: objective.dimension_weights,
        targets: {
          max_open_severity: objective.targets.max_open_severity || 'none',
          operational_targets_pct: objective.targets.operational_targets_pct ?? 0,
        },
      },
      budget: {
        max_iterations: budget.max_iterations,
        diminishing_returns_floor: budget.diminishing_returns_floor,
        reaudit_cadence: budget.reaudit_cadence,
      },
      audit_preset: local.audit_preset?.trim() || 'comprehensive',
    };
  };

  const handleSave = () => {
    setSaveError(null);

    if (!local.name?.trim()) {
      setSaveError({ title: 'Profile name is required' });
      return;
    }
    if (!local.allowed_skills || local.allowed_skills.length === 0) {
      setSaveError({
        title: 'At least one allowed skill is required',
        recovery: 'The controller can only select from the skills you allow here.',
      });
      return;
    }
    if (!budget.max_iterations || budget.max_iterations < 1) {
      setSaveError({ title: 'Max iterations must be at least 1' });
      return;
    }
    const pct = objective.targets.operational_targets_pct ?? 0;
    if (pct < 0 || pct > 100) {
      setSaveError({ title: 'Operational-target completion must be between 0 and 100' });
      return;
    }

    const payload = buildPayload();
    const onError = (error: unknown) => setSaveError(buildSaveError(error));

    if (profile?.id) {
      updateProfile.mutate(
        { id: profile.id, updates: payload },
        { onSuccess: () => onOpenChange(false), onError },
      );
    } else {
      createProfile.mutate(payload, { onSuccess: () => onOpenChange(false), onError });
    }
  };

  const isLoading = createProfile.isPending || updateProfile.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{profile ? 'Edit Auto Steer Profile' : 'Create Auto Steer Profile'}</DialogTitle>
          <DialogDescription>
            An objective profile tells the controller what "done" means and which skills it may use —
            it weights improvement dimensions and sets targets, not a fixed sequence of phases.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Basic Info */}
          <div className="space-y-4">
            <div>
              <Label htmlFor="profile-name">Name *</Label>
              <Input
                id="profile-name"
                value={local.name || ''}
                onChange={(e) => updateField('name', e.target.value)}
                placeholder="e.g., Balanced"
                required
              />
            </div>
            <div>
              <Label htmlFor="profile-description">Description</Label>
              <Textarea
                id="profile-description"
                value={local.description || ''}
                onChange={(e) => updateField('description', e.target.value)}
                placeholder="What this objective optimizes for"
                rows={2}
              />
            </div>
          </div>

          <div>
            <Label>Tags</Label>
            <TagEditor tags={local.tags || []} onChange={(tags) => updateField('tags', tags)} />
          </div>

          {/* Allowed skills */}
          <div className="space-y-2">
            <Label>Allowed skills *</Label>
            <p className="text-xs text-slate-500">The controller selects only from these skills.</p>
            <AllowedSkillsPicker
              selected={local.allowed_skills || []}
              onChange={(skills) => updateField('allowed_skills', skills)}
              options={skillPickerOptions}
              isLoading={skillsLoading}
            />
          </div>

          {/* Dimension weights */}
          <div className="space-y-2">
            <Label>Dimension weights</Label>
            <p className="text-xs text-slate-500">Prioritize which improvement dimensions the controller closes first.</p>
            <DimensionWeightEditor
              weights={objective.dimension_weights}
              onChange={(weights) => setObjective({ dimension_weights: weights })}
              dimensions={dimensions}
              isLoading={dimensionsLoading}
            />
          </div>

          {/* Targets */}
          <div className="space-y-3">
            <Label>Targets (what "done" means)</Label>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <Label htmlFor="target-severity" className="text-xs text-slate-400">
                  Max open severity
                </Label>
                <Select
                  value={objective.targets.max_open_severity || 'none'}
                  onValueChange={(v) => setTargets({ max_open_severity: v })}
                >
                  <SelectTrigger id="target-severity">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SEVERITY_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="target-ops" className="text-xs text-slate-400">
                  Operational-target completion (%)
                </Label>
                <Input
                  id="target-ops"
                  type="number"
                  min={0}
                  max={100}
                  value={objective.targets.operational_targets_pct ?? 0}
                  onChange={(e) => setTargets({ operational_targets_pct: Number(e.target.value) })}
                />
              </div>
            </div>
          </div>

          {/* Budget */}
          <div className="space-y-3">
            <Label>Budget</Label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <Label htmlFor="budget-max-iter" className="text-xs text-slate-400">
                  Max iterations
                </Label>
                <Input
                  id="budget-max-iter"
                  type="number"
                  min={1}
                  value={budget.max_iterations}
                  onChange={(e) => setBudget({ max_iterations: Number(e.target.value) })}
                />
              </div>
              <div>
                <Label htmlFor="budget-floor" className="text-xs text-slate-400">
                  Diminishing-returns floor
                </Label>
                <Input
                  id="budget-floor"
                  type="number"
                  min={0}
                  step={0.01}
                  value={budget.diminishing_returns_floor}
                  onChange={(e) => setBudget({ diminishing_returns_floor: Number(e.target.value) })}
                />
              </div>
              <div>
                <Label htmlFor="budget-cadence" className="text-xs text-slate-400">
                  Full re-audit cadence
                </Label>
                <Input
                  id="budget-cadence"
                  type="number"
                  min={0}
                  value={budget.reaudit_cadence ?? 0}
                  onChange={(e) => setBudget({ reaudit_cadence: Number(e.target.value) })}
                />
              </div>
            </div>
          </div>

          {/* Audit preset */}
          <div>
            <Label htmlFor="audit-preset">Audit preset</Label>
            <Input
              id="audit-preset"
              value={local.audit_preset || ''}
              onChange={(e) => updateField('audit_preset', e.target.value)}
              placeholder="comprehensive"
            />
            <p className="text-xs text-slate-500 mt-1">
              The test-genie preset used for the full re-audit at the termination gate.
            </p>
          </div>
        </div>

        {saveError && (
          <div className="rounded-md border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-100">
            <div className="flex items-start gap-2">
              <AlertCircle className="h-4 w-4 mt-0.5 text-red-300" />
              <div className="space-y-1">
                <p className="font-semibold">{saveError.title}</p>
                {saveError.detail && <p className="text-xs text-red-200/90">{saveError.detail}</p>}
                {saveError.recovery && <p className="text-xs text-red-200/80">{saveError.recovery}</p>}
              </div>
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isLoading}>
            <X className="h-4 w-4 mr-2" />
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={isLoading}>
            <Save className="h-4 w-4 mr-2" />
            {isLoading ? 'Saving...' : 'Save Profile'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function buildSaveError(error: unknown): SaveError {
  const detail = getApiErrorMessage(error);
  const lowerDetail = detail.toLowerCase();

  if (lowerDetail.includes('dimension') || lowerDetail.includes('skill') || lowerDetail.includes('severity')) {
    return {
      title: 'Profile validation failed',
      detail,
      recovery: 'Check the dimension weights, allowed skills, and targets.',
    };
  }

  return {
    title: 'Unable to save profile',
    detail,
    recovery: 'Try again in a moment, or check the server logs if this keeps happening.',
  };
}
