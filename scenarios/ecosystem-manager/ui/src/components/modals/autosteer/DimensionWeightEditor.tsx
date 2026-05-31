/**
 * DimensionWeightEditor
 * Edits the objective's per-dimension weights. Higher weight tells the
 * controller to prioritize closing findings in that dimension. Dimensions are
 * drawn from the canonical vocabulary (SSOT) so weights never reference an
 * unknown dimension.
 */

import { useState } from 'react';
import { Plus, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { DimensionInfo } from '@/types/api';

interface DimensionWeightEditorProps {
  weights: Record<string, number>;
  onChange: (weights: Record<string, number>) => void;
  dimensions: DimensionInfo[];
  isLoading?: boolean;
}

export function DimensionWeightEditor({
  weights,
  onChange,
  dimensions,
  isLoading,
}: DimensionWeightEditorProps) {
  const [pendingDimension, setPendingDimension] = useState('');

  const entries = Object.entries(weights);
  const usedIds = new Set(entries.map(([id]) => id));
  const available = dimensions.filter((d) => !usedIds.has(d.id));
  const describe = (id: string) => dimensions.find((d) => d.id === id)?.description;

  const setWeight = (id: string, value: number) => {
    onChange({ ...weights, [id]: value });
  };

  const removeDimension = (id: string) => {
    const next = { ...weights };
    delete next[id];
    onChange(next);
  };

  const addDimension = () => {
    if (!pendingDimension || usedIds.has(pendingDimension)) return;
    onChange({ ...weights, [pendingDimension]: 1.0 });
    setPendingDimension('');
  };

  return (
    <div className="space-y-3">
      {entries.length === 0 ? (
        <p className="text-xs text-slate-500">
          No dimensions weighted yet. Unweighted dimensions still count toward the objective
          with a default weight of 1.0 — add a dimension to prioritize or de-prioritize it.
        </p>
      ) : (
        <div className="space-y-2">
          {entries.map(([id, weight]) => (
            <div key={id} className="flex items-center gap-2">
              <div className="flex-1 min-w-0">
                <div className="text-sm text-slate-200">{id}</div>
                {describe(id) && (
                  <div className="text-[11px] text-slate-500 truncate" title={describe(id)}>
                    {describe(id)}
                  </div>
                )}
              </div>
              <Input
                type="number"
                min={0}
                step={0.1}
                value={weight}
                onChange={(e) => setWeight(id, Number(e.target.value))}
                className="w-24"
                aria-label={`Weight for ${id}`}
              />
              <Button
                variant="outline"
                size="sm"
                className="h-8 w-8 p-0 text-red-400 hover:text-red-300 border-red-400/30"
                onClick={() => removeDimension(id)}
                aria-label={`Remove ${id}`}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>
          ))}
        </div>
      )}

      <div className="flex items-center gap-2">
        <Select value={pendingDimension} onValueChange={setPendingDimension} disabled={isLoading || available.length === 0}>
          <SelectTrigger className="flex-1">
            <SelectValue placeholder={isLoading ? 'Loading dimensions...' : available.length === 0 ? 'All dimensions added' : 'Add a dimension to weight'} />
          </SelectTrigger>
          <SelectContent>
            {available.map((d) => (
              <SelectItem key={d.id} value={d.id}>
                {d.id}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" onClick={addDimension} disabled={!pendingDimension}>
          <Plus className="h-3.5 w-3.5 mr-1.5" />
          Add
        </Button>
      </div>
    </div>
  );
}
