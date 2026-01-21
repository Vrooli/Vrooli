import { useState, useEffect } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { Label } from '../../../shared/ui/label';
import { useToast } from '../../../shared/ui/Toast';
import { Key, Trash2, RefreshCw, Power, PowerOff, Plus, Check, X, AlertCircle } from 'lucide-react';
import {
  listAPIKeys,
  createAPIKey,
  deleteAPIKey,
  testAPIKey,
  toggleAPIKey,
  APIKey,
  PROVIDER_OPTIONS,
} from '../../../shared/api';

export function APIKeysSettings() {
  const { addToast } = useToast();
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [testingProvider, setTestingProvider] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});

  // Add key modal state
  const [showAddModal, setShowAddModal] = useState(false);
  const [newKeyProvider, setNewKeyProvider] = useState('');
  const [newKeyValue, setNewKeyValue] = useState('');
  const [addingKey, setAddingKey] = useState(false);

  const fetchKeys = async () => {
    try {
      setLoading(true);
      const response = await listAPIKeys();
      setKeys(response.keys || []);
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to load API keys',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchKeys();
  }, []);

  const handleAddKey = async () => {
    if (!newKeyProvider || !newKeyValue) {
      addToast({ type: 'error', message: 'Provider and key are required' });
      return;
    }

    try {
      setAddingKey(true);
      await createAPIKey({ provider: newKeyProvider, key: newKeyValue });
      addToast({ type: 'success', message: `API key for ${newKeyProvider} added successfully` });
      setShowAddModal(false);
      setNewKeyProvider('');
      setNewKeyValue('');
      await fetchKeys();
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to add API key',
      });
    } finally {
      setAddingKey(false);
    }
  };

  const handleDeleteKey = async (provider: string) => {
    if (!confirm(`Are you sure you want to delete the ${provider} API key?`)) {
      return;
    }

    try {
      await deleteAPIKey(provider);
      addToast({ type: 'success', message: `API key for ${provider} deleted` });
      await fetchKeys();
      // Clear test results for this provider
      setTestResults((prev) => {
        const next = { ...prev };
        delete next[provider];
        return next;
      });
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to delete API key',
      });
    }
  };

  const handleTestKey = async (provider: string) => {
    try {
      setTestingProvider(provider);
      const result = await testAPIKey(provider);
      setTestResults((prev) => ({
        ...prev,
        [provider]: { success: result.success, message: result.message },
      }));
      addToast({
        type: result.success ? 'success' : 'error',
        message: result.message,
      });
    } catch (error) {
      setTestResults((prev) => ({
        ...prev,
        [provider]: { success: false, message: error instanceof Error ? error.message : 'Test failed' },
      }));
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to test API key',
      });
    } finally {
      setTestingProvider(null);
    }
  };

  const handleToggleKey = async (provider: string, currentActive: boolean) => {
    try {
      await toggleAPIKey(provider, !currentActive);
      addToast({
        type: 'success',
        message: `API key for ${provider} ${!currentActive ? 'enabled' : 'disabled'}`,
      });
      await fetchKeys();
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to toggle API key',
      });
    }
  };

  const getProviderLabel = (provider: string) => {
    const option = PROVIDER_OPTIONS.find((p) => p.value === provider);
    return option?.label || provider;
  };

  const getProviderDescription = (provider: string) => {
    const option = PROVIDER_OPTIONS.find((p) => p.value === provider);
    return option?.description || '';
  };

  // Providers that are not yet configured
  const availableProviders = PROVIDER_OPTIONS.filter(
    (p) => !keys.some((k) => k.provider === p.value)
  );

  return (
    <AdminLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">API Keys</h1>
            <p className="text-slate-400 mt-1">
              Manage AI provider API keys for Vrooli-hosted AI services
            </p>
          </div>
          <Button
            onClick={() => setShowAddModal(true)}
            disabled={availableProviders.length === 0}
            className="gap-2"
          >
            <Plus className="h-4 w-4" />
            Add API Key
          </Button>
        </div>

        {/* Info Card */}
        <Card className="bg-blue-500/10 border-blue-500/20">
          <CardContent className="pt-4">
            <div className="flex gap-3">
              <AlertCircle className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-slate-300">
                <p className="font-medium text-blue-400 mb-1">How API Keys Work</p>
                <p>
                  These API keys are used when customers don't provide their own keys (BYOK).
                  When a user performs an AI operation, the system uses these keys and charges
                  the user's credit balance. Keys are encrypted at rest.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Keys List */}
        {loading ? (
          <div className="text-center py-8 text-slate-400">Loading API keys...</div>
        ) : keys.length === 0 ? (
          <Card className="border-dashed">
            <CardContent className="pt-6 text-center">
              <Key className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400 mb-4">No API keys configured yet</p>
              <Button onClick={() => setShowAddModal(true)} className="gap-2">
                <Plus className="h-4 w-4" />
                Add Your First API Key
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4">
            {keys.map((key) => (
              <Card key={key.id} className={!key.is_active ? 'opacity-60' : ''}>
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
                      {testResults[key.provider] && (
                        <span
                          className={`text-sm flex items-center gap-1 ${
                            testResults[key.provider].success ? 'text-green-400' : 'text-red-400'
                          }`}
                        >
                          {testResults[key.provider].success ? (
                            <Check className="h-4 w-4" />
                          ) : (
                            <X className="h-4 w-4" />
                          )}
                          {testResults[key.provider].message}
                        </span>
                      )}
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
                          {new Date(key.last_verified_at).toLocaleDateString()}
                        </p>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleTestKey(key.provider)}
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
                        onClick={() => handleToggleKey(key.provider, key.is_active)}
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
                        onClick={() => handleDeleteKey(key.provider)}
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
            <Card className="w-full max-w-md mx-4">
              <CardHeader>
                <CardTitle>Add API Key</CardTitle>
                <CardDescription>
                  Configure an AI provider API key for Vrooli-hosted services
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="provider">Provider</Label>
                  <select
                    id="provider"
                    value={newKeyProvider}
                    onChange={(e) => setNewKeyProvider(e.target.value)}
                    className="w-full bg-slate-800 border border-slate-700 rounded-md px-3 py-2 text-white"
                  >
                    <option value="">Select a provider...</option>
                    {availableProviders.map((p) => (
                      <option key={p.value} value={p.value}>
                        {p.label} - {p.description}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="key">API Key</Label>
                  <Input
                    id="key"
                    type="password"
                    value={newKeyValue}
                    onChange={(e) => setNewKeyValue(e.target.value)}
                    placeholder="sk-..."
                    className="font-mono"
                  />
                  <p className="text-xs text-slate-400">
                    The key will be encrypted before storage
                  </p>
                </div>
                <div className="flex justify-end gap-2 pt-4">
                  <Button
                    variant="outline"
                    onClick={() => {
                      setShowAddModal(false);
                      setNewKeyProvider('');
                      setNewKeyValue('');
                    }}
                  >
                    Cancel
                  </Button>
                  <Button onClick={handleAddKey} disabled={addingKey || !newKeyProvider || !newKeyValue}>
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
