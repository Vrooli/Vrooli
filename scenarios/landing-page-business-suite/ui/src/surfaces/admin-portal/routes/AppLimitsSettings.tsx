import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { FormField, inputClassName } from '../components/FormField';
import { Callout } from '../components/Callout';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { Label } from '../../../shared/ui/label';
import { useToast } from '../../../shared/ui/Toast';
import { AppWindow, Save, Infinity, DollarSign, Plus, Trash2 } from 'lucide-react';
import { formatDollars, TIER_OPTIONS } from '../../../shared/api';
import {
  getEditKey,
  getTierLabel,
  getTierColor,
  isUnlimitedValue,
} from '../../../shared/lib/tierUtils';
import { useAppLimitsForm } from '../hooks/useAppLimitsForm';
import {
  APP_OPTIONS,
  getSelectedAppLabel,
} from '../services/appLimits.service';

export function AppLimitsSettings() {
  const { addToast } = useToast();
  const {
    selectedApp,
    limits,
    newLimit,
    limitKeys,
    loading,
    saving,
    showAddLimit,
    setSelectedApp,
    setEditedValue,
    setNewLimit,
    setShowAddLimit,
    handleSave,
    handleAddLimit,
    handleDeleteLimit,
    resetNewLimitForm,
    getEditedOrDisplayValue,
    isEdited,
  } = useAppLimitsForm();

  const onSave = async (tierID: string, limit: typeof limits[string][number]) => {
    const result = await handleSave(tierID, limit);
    addToast({ type: result.success ? 'success' : 'error', message: result.message });
  };

  const onAddLimit = async () => {
    const result = await handleAddLimit();
    addToast({ type: result.success ? 'success' : 'error', message: result.message });
  };

  const onDeleteLimit = async (tierID: string, limitKey: string) => {
    if (!confirm(`Delete limit "${limitKey}" for tier "${tierID}"?`)) return;
    const result = await handleDeleteLimit(tierID, limitKey);
    addToast({ type: result.success ? 'success' : 'error', message: result.message });
  };

  return (
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="App Limits"
          description="Configure per-app limits for each subscription tier"
          icon={AppWindow}
          iconBgClass="bg-red-500/10"
          iconColorClass="text-red-400"
          testId="applimits-header"
        />

        <FormSection
          title="Select Application"
          description="Choose an application to configure its limits"
          icon={AppWindow}
          iconColorClass="text-red-300"
        >
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
        </FormSection>

        <Callout
          type="info"
          title="Understanding App Limits"
          message="App limits allow you to set specific restrictions for individual applications beyond the global AI credit limits. These are app-specific quotas like workflow exports, API calls, or feature usage counts."
        />

        {loading ? (
          <div className="text-center py-8 text-slate-400">Loading app limits...</div>
        ) : limitKeys.size === 0 ? (
          <Card className={`${LAYOUT.card.base} border-dashed`}>
            <CardContent className="pt-6 text-center">
              <AppWindow className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400 mb-4">
                No app-specific limits configured for {getSelectedAppLabel(selectedApp)}
              </p>
              <Button onClick={() => setShowAddLimit(true)} className="gap-2">
                <Plus className="h-4 w-4" />
                Add First Limit
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className={LAYOUT.sectionSpacing}>
            {/* Limits by Key */}
            {Array.from(limitKeys).map((limitKey) => (
              <FormSection
                key={limitKey}
                title={limitKey.replace(/_/g, ' ')}
                description={`Per-tier limits for ${limitKey}`}
                icon={DollarSign}
                iconColorClass="text-purple-300"
                className="capitalize"
              >
                  <div className="grid gap-4">
                    {TIER_OPTIONS.map((tierOption) => {
                      const tierID = tierOption.value;
                      const tierLimits = limits[tierID] || [];
                      const limit = tierLimits.find((l) => l.limit_key === limitKey);

                      if (!limit) return null;

                      const editKey = getEditKey(tierID, limitKey);
                      const currentValue = limit.limit_value;
                      const isUnlimited = isUnlimitedValue(currentValue);

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
                                  value={getEditedOrDisplayValue(editKey, limit)}
                                  onChange={(e) => setEditedValue(editKey, e.target.value)}
                                  className="w-32 pl-8"
                                  placeholder="0.00"
                                />
                              </div>
                            </div>
                            <Button
                              size="sm"
                              onClick={() => onSave(tierID, limit)}
                              disabled={!isEdited(editKey) || saving === editKey}
                              className="gap-1"
                            >
                              <Save className="h-4 w-4" />
                              {saving === editKey ? 'Saving...' : 'Save'}
                            </Button>
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => onDeleteLimit(tierID, limitKey)}
                              disabled={saving === `delete:${tierID}:${limitKey}`}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
              </FormSection>
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
            <Card className={`${LAYOUT.card.base} w-full max-w-md mx-4`}>
              <CardHeader>
                <CardTitle>Add New Limit</CardTitle>
                <CardDescription>
                  Create a new app-specific limit for {getSelectedAppLabel(selectedApp)}
                </CardDescription>
              </CardHeader>
              <CardContent className={LAYOUT.contentSpacing}>
                <FormField label="Tier" htmlFor="new-tier">
                  <select
                    id="new-tier"
                    value={newLimit.tier_id}
                    onChange={(e) => setNewLimit({ tier_id: e.target.value })}
                    className={inputClassName}
                  >
                    {TIER_OPTIONS.map((tier) => (
                      <option key={tier.value} value={tier.value}>
                        {tier.label}
                      </option>
                    ))}
                  </select>
                </FormField>
                <FormField
                  label="Limit Key"
                  htmlFor="new-limit-key"
                  helpText="Use snake_case for limit keys (e.g., workflow_exports, api_calls)"
                >
                  <input
                    id="new-limit-key"
                    value={newLimit.limit_key}
                    onChange={(e) => setNewLimit({ limit_key: e.target.value })}
                    placeholder="e.g., workflow_exports"
                    className={inputClassName}
                  />
                </FormField>
                <FormField
                  label="Dollar Value"
                  htmlFor="new-display-dollars"
                  helpText='Enter "unlimited" or -1 for no limit'
                >
                  <div className="relative">
                    <DollarSign className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                    <input
                      id="new-display-dollars"
                      type="text"
                      value={newLimit.display_dollars}
                      onChange={(e) => setNewLimit({ display_dollars: e.target.value })}
                      placeholder="0.00"
                      className={`${inputClassName} pl-8`}
                    />
                  </div>
                </FormField>
                <div className="flex gap-2 justify-end pt-4">
                  <Button
                    variant="outline"
                    onClick={() => {
                      setShowAddLimit(false);
                      resetNewLimitForm();
                    }}
                  >
                    Cancel
                  </Button>
                  <Button onClick={onAddLimit} disabled={saving === 'new'}>
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
