import { useState, useEffect, useCallback } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { Label } from '../../../shared/ui/label';
import { useToast } from '../../../shared/ui/Toast';
import { AppWindow, Save, AlertCircle, Infinity, DollarSign, Plus, Trash2 } from 'lucide-react';
import {
  getAppLimits,
  updateTierLimit,
  createTierLimit,
  deleteTierLimit,
  TierLimit,
  TierLimitUpdate,
  formatDollars,
  TIER_OPTIONS,
} from '../../../shared/api';

const APP_OPTIONS = [
  { value: 'browser-automation-studio', label: 'Browser Automation Studio' },
  // Future apps can be added here
] as const;

export function AppLimitsSettings() {
  const { addToast } = useToast();
  const [selectedApp, setSelectedApp] = useState<string>(APP_OPTIONS[0].value);
  const [limits, setLimits] = useState<Record<string, TierLimit[]>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);
  const [editedValues, setEditedValues] = useState<Record<string, string>>({});
  const [showAddLimit, setShowAddLimit] = useState(false);
  const [newLimit, setNewLimit] = useState({
    tier_id: 'solo',
    limit_key: '',
    display_dollars: '',
  });

  const fetchLimits = useCallback(async () => {
    try {
      setLoading(true);
      const response = await getAppLimits(selectedApp);
      setLimits(response.limits || {});
      setEditedValues({});
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to load app limits',
      });
    } finally {
      setLoading(false);
    }
  }, [selectedApp, addToast]);

  useEffect(() => {
    fetchLimits();
  }, [fetchLimits]);

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

      await updateTierLimit(tierID, limit.limit_key, update, selectedApp);
      addToast({ type: 'success', message: `Limit for ${tierID}/${limit.limit_key} updated` });

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

  const handleAddLimit = async () => {
    if (!newLimit.limit_key.trim()) {
      addToast({ type: 'error', message: 'Please enter a limit key' });
      return;
    }

    try {
      setSaving('new');
      const dollars = parseFloat(newLimit.display_dollars) || 0;

      await createTierLimit({
        tier_id: newLimit.tier_id,
        limit_type: 'app_specific',
        limit_key: newLimit.limit_key.trim(),
        limit_value: Math.round(dollars * 100 * 1000000), // Convert to internal units
        cost_multiplier: 1000000,
        app_bundle_key: selectedApp,
        reset_period: 'monthly',
      });

      addToast({ type: 'success', message: 'New limit created' });
      setShowAddLimit(false);
      setNewLimit({ tier_id: 'solo', limit_key: '', display_dollars: '' });
      await fetchLimits();
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to create limit',
      });
    } finally {
      setSaving(null);
    }
  };

  const handleDeleteLimit = async (tierID: string, limitKey: string) => {
    if (!confirm(`Delete limit "${limitKey}" for tier "${tierID}"?`)) return;

    try {
      setSaving(`delete:${tierID}:${limitKey}`);
      await deleteTierLimit(tierID, limitKey, selectedApp);
      addToast({ type: 'success', message: 'Limit deleted' });
      await fetchLimits();
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to delete limit',
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

  // Collect all unique limit keys across tiers
  const limitKeys = new Set<string>();
  Object.values(limits).forEach((tierLimits) => {
    tierLimits.forEach((limit) => {
      limitKeys.add(limit.limit_key);
    });
  });

  return (
    <AdminLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">App Limits</h1>
          <p className="text-slate-400 mt-1">
            Configure per-app limits for each subscription tier
          </p>
        </div>

        {/* App Selector */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Select Application</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2 flex-wrap">
              {APP_OPTIONS.map((app) => (
                <Button
                  key={app.value}
                  variant={selectedApp === app.value ? 'default' : 'outline'}
                  onClick={() => setSelectedApp(app.value)}
                  className="gap-2"
                >
                  <AppWindow className="h-4 w-4" />
                  {app.label}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Info Card */}
        <Card className="bg-blue-500/10 border-blue-500/20">
          <CardContent className="pt-4">
            <div className="flex gap-3">
              <AlertCircle className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-slate-300">
                <p className="font-medium text-blue-400 mb-1">Understanding App Limits</p>
                <p>
                  App limits allow you to set specific restrictions for individual applications
                  beyond the global AI credit limits. These are app-specific quotas like
                  workflow exports, API calls, or feature usage counts.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {loading ? (
          <div className="text-center py-8 text-slate-400">Loading app limits...</div>
        ) : limitKeys.size === 0 ? (
          <Card className="border-dashed">
            <CardContent className="pt-6 text-center">
              <AppWindow className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400 mb-4">No app-specific limits configured for {APP_OPTIONS.find(a => a.value === selectedApp)?.label}</p>
              <Button onClick={() => setShowAddLimit(true)} className="gap-2">
                <Plus className="h-4 w-4" />
                Add First Limit
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-6">
            {/* Limits by Key */}
            {Array.from(limitKeys).map((limitKey) => (
              <Card key={limitKey}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-purple-500/20">
                        <DollarSign className="h-5 w-5 text-purple-400" />
                      </div>
                      <div>
                        <CardTitle className="capitalize">{limitKey.replace(/_/g, ' ')}</CardTitle>
                        <CardDescription>
                          Per-tier limits for {limitKey}
                        </CardDescription>
                      </div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4">
                    {TIER_OPTIONS.map((tierOption) => {
                      const tierID = tierOption.value;
                      const tierLimits = limits[tierID] || [];
                      const limit = tierLimits.find((l) => l.limit_key === limitKey);

                      if (!limit) return null;

                      const editKey = getEditKey(tierID, limitKey);
                      const isEdited = editedValues[editKey] !== undefined;
                      const currentValue = limit.limit_value;
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
                                  formatDollars(currentValue, limit.cost_multiplier)
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
                                      : limit.display_dollars?.toFixed(2) ?? '0')
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
                              onClick={() => handleSave(tierID, limit)}
                              disabled={!isEdited || saving === editKey}
                              className="gap-1"
                            >
                              <Save className="h-4 w-4" />
                              {saving === editKey ? 'Saving...' : 'Save'}
                            </Button>
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => handleDeleteLimit(tierID, limitKey)}
                              disabled={saving === `delete:${tierID}:${limitKey}`}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </CardContent>
              </Card>
            ))}

            {/* Add New Limit Button */}
            <Button onClick={() => setShowAddLimit(true)} variant="outline" className="gap-2 w-full">
              <Plus className="h-4 w-4" />
              Add New Limit Type
            </Button>
          </div>
        )}

        {/* Add Limit Modal */}
        {showAddLimit && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <Card className="w-full max-w-md mx-4">
              <CardHeader>
                <CardTitle>Add New Limit</CardTitle>
                <CardDescription>
                  Create a new app-specific limit for {APP_OPTIONS.find(a => a.value === selectedApp)?.label}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <Label htmlFor="new-tier">Tier</Label>
                  <select
                    id="new-tier"
                    value={newLimit.tier_id}
                    onChange={(e) => setNewLimit((prev) => ({ ...prev, tier_id: e.target.value }))}
                    className="w-full mt-1 px-3 py-2 bg-slate-800 border border-slate-700 rounded-md text-white"
                  >
                    {TIER_OPTIONS.map((tier) => (
                      <option key={tier.value} value={tier.value}>
                        {tier.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <Label htmlFor="new-limit-key">Limit Key</Label>
                  <Input
                    id="new-limit-key"
                    value={newLimit.limit_key}
                    onChange={(e) => setNewLimit((prev) => ({ ...prev, limit_key: e.target.value }))}
                    placeholder="e.g., workflow_exports"
                    className="mt-1"
                  />
                  <p className="text-xs text-slate-400 mt-1">
                    Use snake_case for limit keys (e.g., workflow_exports, api_calls)
                  </p>
                </div>
                <div>
                  <Label htmlFor="new-display-dollars">Dollar Value</Label>
                  <div className="relative mt-1">
                    <DollarSign className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                    <Input
                      id="new-display-dollars"
                      type="text"
                      value={newLimit.display_dollars}
                      onChange={(e) => setNewLimit((prev) => ({ ...prev, display_dollars: e.target.value }))}
                      placeholder="0.00"
                      className="pl-8"
                    />
                  </div>
                  <p className="text-xs text-slate-400 mt-1">
                    Enter "unlimited" or -1 for no limit
                  </p>
                </div>
                <div className="flex gap-2 justify-end pt-4">
                  <Button variant="outline" onClick={() => setShowAddLimit(false)}>
                    Cancel
                  </Button>
                  <Button onClick={handleAddLimit} disabled={saving === 'new'}>
                    {saving === 'new' ? 'Creating...' : 'Create Limit'}
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
