import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormField } from '../components/FormField';
import { inputClassName } from '../components/formFieldClasses';
import { Callout } from '../components/Callout';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { useToast } from '../../../shared/ui/useToast';
import { Key, Trash2, RefreshCw, Power, PowerOff, Plus, Check, X } from 'lucide-react';
import { formatDateOnly } from '../../../shared/lib/dateFormatters';
import { useAPIKeysForm } from '../hooks/useAPIKeysForm';
import { getProviderLabel, getProviderDescription } from '../services/apiKeys.service';

export function APIKeysSettings() {
  const { addToast } = useToast();
  const {
    keys,
    testResults,
    availableProviders,
    showAddModal,
    newKeyProvider,
    newKeyValue,
    loading,
    testingProvider,
    addingKey,
    handleAddKey,
    handleDeleteKey,
    handleTestKey,
    handleToggleKey,
    setShowAddModal,
    setNewKeyProvider,
    setNewKeyValue,
    clearAddForm,
  } = useAPIKeysForm();

  const onAddKey = async () => {
    const result = await handleAddKey();
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message || (result.success ? 'Key added' : 'Failed to add key'),
    });
  };

  const onDeleteKey = async (provider: string) => {
    if (!confirm(`Are you sure you want to delete the ${provider} API key?`)) {
      return;
    }
    const result = await handleDeleteKey(provider);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message || (result.success ? 'Key deleted' : 'Failed to delete key'),
    });
  };

  const onTestKey = async (provider: string) => {
    const result = await handleTestKey(provider);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message,
    });
  };

  const onToggleKey = async (provider: string, currentActive: boolean) => {
    const result = await handleToggleKey(provider, currentActive);
    addToast({
      type: result.success ? 'success' : 'error',
      message: result.message || (result.success ? 'Key toggled' : 'Failed to toggle key'),
    });
  };

  return (
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          title="API Keys"
          description="Manage AI provider API keys for Vrooli-hosted AI services"
          icon={Key}
          iconBgClass="bg-yellow-500/10"
          iconColorClass="text-yellow-400"
          testId="apikeys-header"
          actions={
            <Button
              onClick={() => { setShowAddModal(true); }}
              disabled={availableProviders.length === 0}
              className="gap-2"
            >
              <Plus className="h-4 w-4" />
              Add API Key
            </Button>
          }
        />

        <Callout
          type="info"
          title="How API Keys Work"
          message="These API keys are used when customers don't provide their own keys (BYOK). When a user performs an AI operation, the system uses these keys and charges the user's credit balance. Keys are encrypted at rest."
        />

        {/* Keys List */}
        {loading ? (
          <div className="text-center py-8 text-slate-400">Loading API keys...</div>
        ) : keys.length === 0 ? (
          <Card className={`${LAYOUT.card.base} border-dashed`}>
            <CardContent className="pt-6 text-center">
              <Key className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400 mb-4">No API keys configured yet</p>
              <Button onClick={() => { setShowAddModal(true); }} className="gap-2">
                <Plus className="h-4 w-4" />
                Add Your First API Key
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className={LAYOUT.sectionSpacing}>
            {keys.map((key) => (
              <Card key={key.id} className={`${LAYOUT.card.base} ${!key.is_active ? 'opacity-60' : ''}`}>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div
                        className={`p-2 rounded-lg ${
                          key.is_active ? 'bg-green-500/20' : 'bg-slate-500/20'
                        }`}
                      >
                        <Key className={`h-5 w-5 ${key.is_active ? 'text-green-400' : 'text-slate-400'}`} />
                      </div>
                      <div>
                        <CardTitle className="text-lg">{getProviderLabel(key.provider)}</CardTitle>
                        <CardDescription>{getProviderDescription(key.provider)}</CardDescription>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {(() => {
                        const result = testResults[key.provider];
                        if (!result) return null;
                        return (
                          <span
                            className={`text-sm flex items-center gap-1 ${
                              result.success ? 'text-green-400' : 'text-red-400'
                            }`}
                          >
                            {result.success ? (
                              <Check className="h-4 w-4" />
                            ) : (
                              <X className="h-4 w-4" />
                            )}
                            {result.message}
                          </span>
                        );
                      })()}
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between">
                    <div className="space-y-1 text-sm text-slate-400">
                      <p>
                        <span className="text-slate-500">Key:</span>{' '}
                        <code className="bg-slate-800 px-2 py-0.5 rounded">{key.key_hint}</code>
                      </p>
                      {key.last_verified_at && (
                        <p>
                          <span className="text-slate-500">Last verified:</span>{' '}
                          {formatDateOnly(key.last_verified_at)}
                        </p>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => { void onTestKey(key.provider); }}
                        disabled={testingProvider === key.provider}
                        className="gap-1"
                      >
                        <RefreshCw
                          className={`h-4 w-4 ${testingProvider === key.provider ? 'animate-spin' : ''}`}
                        />
                        Test
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => { void onToggleKey(key.provider, key.is_active); }}
                        className="gap-1"
                      >
                        {key.is_active ? (
                          <>
                            <PowerOff className="h-4 w-4" />
                            Disable
                          </>
                        ) : (
                          <>
                            <Power className="h-4 w-4" />
                            Enable
                          </>
                        )}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => { void onDeleteKey(key.provider); }}
                        className="gap-1 text-red-400 hover:text-red-300 hover:border-red-500/50"
                      >
                        <Trash2 className="h-4 w-4" />
                        Delete
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        {/* Add Key Modal */}
        {showAddModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <Card className={`${LAYOUT.card.base} w-full max-w-md mx-4`}>
              <CardHeader>
                <CardTitle>Add API Key</CardTitle>
                <CardDescription>
                  Configure an AI provider API key for Vrooli-hosted services
                </CardDescription>
              </CardHeader>
              <CardContent className={LAYOUT.contentSpacing}>
                <FormField label="Provider" htmlFor="provider">
                  <select
                    id="provider"
                    value={newKeyProvider}
                    onChange={(e) => { setNewKeyProvider(e.target.value); }}
                    className={inputClassName}
                  >
                    <option value="">Select a provider...</option>
                    {availableProviders.map((p) => (
                      <option key={p.value} value={p.value}>
                        {p.label} - {p.description}
                      </option>
                    ))}
                  </select>
                </FormField>
                <FormField label="API Key" htmlFor="key" helpText="The key will be encrypted before storage">
                  <input
                    id="key"
                    type="password"
                    value={newKeyValue}
                    onChange={(e) => { setNewKeyValue(e.target.value); }}
                    placeholder="sk-..."
                    className={`${inputClassName} font-mono`}
                  />
                </FormField>
                <div className="flex justify-end gap-2 pt-4">
                  <Button variant="outline" onClick={clearAddForm}>
                    Cancel
                  </Button>
                  <Button onClick={() => { void onAddKey(); }} disabled={addingKey || !newKeyProvider || !newKeyValue}>
                    {addingKey ? 'Adding...' : 'Add Key'}
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
