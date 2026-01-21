import { useState, useEffect } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { Label } from '../../../shared/ui/label';
import { useToast } from '../../../shared/ui/Toast';
import { Gauge, Save, AlertCircle, Infinity, DollarSign } from 'lucide-react';
import {
  getAllTierLimits,
  updateTierLimit,
  TierLimit,
  TierLimitUpdate,
  formatDollars,
  TIER_OPTIONS,
} from '../../../shared/api';

export function TierLimitsSettings() {
  const { addToast } = useToast();
  const [limits, setLimits] = useState<Record<string, TierLimit[]>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);
  const [editedValues, setEditedValues] = useState<Record<string, string>>({});

  const fetchLimits = async () => {
    try {
      setLoading(true);
      const response = await getAllTierLimits();
      setLimits(response.limits || {});
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to load tier limits',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLimits();
  }, []);

  const getEditKey = (tierID: string, limitKey: string) => `${tierID}:${limitKey}`;

  const handleSave = async (tierID: string, limit: TierLimit) => {
    const editKey = getEditKey(tierID, limit.limit_key);
    const editedValue = editedValues[editKey];

    if (editedValue === undefined) return;

    try {
      setSaving(editKey);

      let update: TierLimitUpdate;
      if (editedValue === 'unlimited' || editedValue === '-1') {
        update = { is_unlimited: true };
      } else {
        const dollars = parseFloat(editedValue);
        if (isNaN(dollars) || dollars < 0) {
          addToast({ type: 'error', message: 'Please enter a valid dollar amount' });
          return;
        }
        update = { display_dollars: dollars };
      }

      await updateTierLimit(tierID, limit.limit_key, update);
      addToast({ type: 'success', message: `Limit for ${tierID}/${limit.limit_key} updated` });

      // Clear edited value and refresh
      setEditedValues((prev) => {
        const next = { ...prev };
        delete next[editKey];
        return next;
      });
      await fetchLimits();
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to update limit',
      });
    } finally {
      setSaving(null);
    }
  };

  const getTierLabel = (tierID: string) => {
    const option = TIER_OPTIONS.find((t) => t.value === tierID);
    return option?.label || tierID;
  };

  const getTierColor = (tierID: string) => {
    switch (tierID) {
      case 'free':
        return 'text-slate-400';
      case 'solo':
        return 'text-blue-400';
      case 'pro':
        return 'text-purple-400';
      case 'studio':
        return 'text-amber-400';
      case 'business':
        return 'text-emerald-400';
      default:
        return 'text-slate-400';
    }
  };

  // Group limits by limit_key across all tiers
  const limitKeys = new Set<string>();
  Object.values(limits).forEach((tierLimits) => {
    tierLimits.forEach((limit) => {
      if (limit.limit_type === 'cost_based') {
        limitKeys.add(limit.limit_key);
      }
    });
  });

  return (
    <AdminLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Tier Limits</h1>
          <p className="text-slate-400 mt-1">
            Configure AI credit limits for each subscription tier
          </p>
        </div>

        {/* Info Card */}
        <Card className="bg-blue-500/10 border-blue-500/20">
          <CardContent className="pt-4">
            <div className="flex gap-3">
              <AlertCircle className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-slate-300">
                <p className="font-medium text-blue-400 mb-1">Understanding Credit Limits</p>
                <p>
                  Credits are shared across all Vrooli apps. Enter the monthly dollar value of AI
                  usage each tier should receive. The system converts this to internal units for
                  precise tracking. Set to "unlimited" for business tier or during promotions.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {loading ? (
          <div className="text-center py-8 text-slate-400">Loading tier limits...</div>
        ) : Object.keys(limits).length === 0 ? (
          <Card className="border-dashed">
            <CardContent className="pt-6 text-center">
              <Gauge className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400">No tier limits configured</p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-6">
            {/* AI Credits Section */}
            <Card>
              <CardHeader>
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-purple-500/20">
                    <DollarSign className="h-5 w-5 text-purple-400" />
                  </div>
                  <div>
                    <CardTitle>AI Credits (Cost-Based)</CardTitle>
                    <CardDescription>
                      Monthly AI usage limit per tier, measured in dollar value
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4">
                  {TIER_OPTIONS.map((tierOption) => {
                    const tierID = tierOption.value;
                    const tierLimits = limits[tierID] || [];
                    const aiCreditsLimit = tierLimits.find(
                      (l) => l.limit_key === 'ai_credits' && l.limit_type === 'cost_based'
                    );

                    if (!aiCreditsLimit) return null;

                    const editKey = getEditKey(tierID, 'ai_credits');
                    const isEdited = editedValues[editKey] !== undefined;
                    const currentValue = aiCreditsLimit.limit_value;
                    const isUnlimited = currentValue < 0;

                    return (
                      <div
                        key={tierID}
                        className="flex items-center justify-between p-4 bg-slate-800/50 rounded-lg border border-slate-700"
                      >
                        <div className="flex items-center gap-4">
                          <div className={`font-medium ${getTierColor(tierID)}`}>
                            {getTierLabel(tierID)}
                          </div>
                          <div className="text-sm text-slate-400">
                            Current:{' '}
                            <span className="text-white">
                              {isUnlimited ? (
                                <span className="flex items-center gap-1">
                                  <Infinity className="h-4 w-4" /> Unlimited
                                </span>
                              ) : (
                                formatDollars(currentValue, aiCreditsLimit.cost_multiplier)
                              )}
                            </span>
                            /month
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex items-center gap-2">
                            <Label htmlFor={editKey} className="sr-only">
                              Limit
                            </Label>
                            <div className="relative">
                              <DollarSign className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                              <Input
                                id={editKey}
                                type="text"
                                value={
                                  editedValues[editKey] ??
                                  (isUnlimited
                                    ? 'unlimited'
                                    : aiCreditsLimit.display_dollars?.toFixed(2) ?? '0')
                                }
                                onChange={(e) => {
                                  const val = e.target.value.toLowerCase();
                                  setEditedValues((prev) => ({
                                    ...prev,
                                    [editKey]: val,
                                  }));
                                }}
                                className="w-32 pl-8"
                                placeholder="0.00"
                              />
                            </div>
                          </div>
                          <Button
                            size="sm"
                            onClick={() => handleSave(tierID, aiCreditsLimit)}
                            disabled={!isEdited || saving === editKey}
                            className="gap-1"
                          >
                            <Save className="h-4 w-4" />
                            {saving === editKey ? 'Saving...' : 'Save'}
                          </Button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>

            {/* Quick Actions */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Quick Actions</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      // Set all tiers to typical values
                      setEditedValues({
                        'free:ai_credits': '0',
                        'solo:ai_credits': '5',
                        'pro:ai_credits': '20',
                        'studio:ai_credits': '100',
                        'business:ai_credits': 'unlimited',
                      });
                    }}
                  >
                    Reset to Defaults
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      // Double all limits
                      const doubled: Record<string, string> = {};
                      TIER_OPTIONS.forEach((tier) => {
                        const limit = limits[tier.value]?.find(
                          (l) => l.limit_key === 'ai_credits' && l.limit_type === 'cost_based'
                        );
                        if (limit && limit.display_dollars && limit.limit_value >= 0) {
                          doubled[`${tier.value}:ai_credits`] = (limit.display_dollars * 2).toString();
                        }
                      });
                      setEditedValues((prev) => ({ ...prev, ...doubled }));
                    }}
                  >
                    Double All Limits
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}
