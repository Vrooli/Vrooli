import { useState, useCallback } from 'react';
import {
  Database,
  AlertTriangle,
  FolderX,
  FileX,
  RotateCcw,
  Trash2,
  RefreshCw,
  Loader2,
  Check,
} from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { useProjectStore } from '@stores/projectStore';
import { useExportStore } from '@stores/exportStore';
import { SettingSection } from './shared';

export function DataSection() {
  const {
    resetDisplaySettings,
    resetReplaySettings,
    resetWorkflowDefaults,
    clearApiKeys,
  } = useSettingsStore();

  const { projects, fetchProjects, deleteAllProjects } = useProjectStore();
  const { exports, fetchExports, deleteAllExports } = useExportStore();

  const [deleteConfirmation, setDeleteConfirmation] = useState<'projects' | 'exports' | 'settings' | 'all' | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteResult, setDeleteResult] = useState<{ success: boolean; message: string } | null>(null);

  const refreshDataCounts = useCallback(() => {
    fetchProjects();
    fetchExports();
  }, [fetchProjects, fetchExports]);

  const handleDeleteProjects = useCallback(async () => {
    setIsDeleting(true);
    setDeleteResult(null);
    try {
      const result = await deleteAllProjects();
      if (result.errors.length > 0) {
        setDeleteResult({
          success: false,
          message: `Deleted ${result.deleted} projects. Errors: ${result.errors.join(', ')}`,
        });
      } else {
        setDeleteResult({
          success: true,
          message: `Successfully deleted ${result.deleted} project${result.deleted !== 1 ? 's' : ''} and all associated workflows.`,
        });
      }
    } catch (error) {
      setDeleteResult({
        success: false,
        message: error instanceof Error ? error.message : 'Failed to delete projects',
      });
    } finally {
      setIsDeleting(false);
      setDeleteConfirmation(null);
    }
  }, [deleteAllProjects]);

  const handleDeleteExports = useCallback(async () => {
    setIsDeleting(true);
    setDeleteResult(null);
    try {
      const result = await deleteAllExports();
      if (result.errors.length > 0) {
        setDeleteResult({
          success: false,
          message: `Deleted ${result.deleted} exports. Errors: ${result.errors.join(', ')}`,
        });
      } else {
        setDeleteResult({
          success: true,
          message: `Successfully deleted ${result.deleted} export${result.deleted !== 1 ? 's' : ''}.`,
        });
      }
    } catch (error) {
      setDeleteResult({
        success: false,
        message: error instanceof Error ? error.message : 'Failed to delete exports',
      });
    } finally {
      setIsDeleting(false);
      setDeleteConfirmation(null);
    }
  }, [deleteAllExports]);

  const handleResetAllSettings = useCallback(() => {
    setIsDeleting(true);
    setDeleteResult(null);
    try {
      resetDisplaySettings();
      resetReplaySettings();
      resetWorkflowDefaults();
      clearApiKeys();
      // Clear all localStorage items with our prefix
      if (typeof window !== 'undefined') {
        const keysToRemove: string[] = [];
        for (let i = 0; i < window.localStorage.length; i++) {
          const key = window.localStorage.key(i);
          if (key && key.startsWith('browserAutomation.')) {
            keysToRemove.push(key);
          }
        }
        keysToRemove.forEach((key) => window.localStorage.removeItem(key));
      }
      setDeleteResult({
        success: true,
        message: 'All settings have been reset to defaults.',
      });
    } catch (error) {
      setDeleteResult({
        success: false,
        message: error instanceof Error ? error.message : 'Failed to reset settings',
      });
    } finally {
      setIsDeleting(false);
      setDeleteConfirmation(null);
    }
  }, [resetDisplaySettings, resetReplaySettings, resetWorkflowDefaults, clearApiKeys]);

  const handleDeleteAll = useCallback(async () => {
    setIsDeleting(true);
    setDeleteResult(null);
    const messages: string[] = [];
    let hasErrors = false;

    try {
      // Delete projects
      const projectResult = await deleteAllProjects();
      if (projectResult.errors.length > 0) {
        hasErrors = true;
        messages.push(`Projects: ${projectResult.deleted} deleted, ${projectResult.errors.length} errors`);
      } else {
        messages.push(`${projectResult.deleted} project${projectResult.deleted !== 1 ? 's' : ''} deleted`);
      }

      // Delete exports
      const exportResult = await deleteAllExports();
      if (exportResult.errors.length > 0) {
        hasErrors = true;
        messages.push(`Exports: ${exportResult.deleted} deleted, ${exportResult.errors.length} errors`);
      } else {
        messages.push(`${exportResult.deleted} export${exportResult.deleted !== 1 ? 's' : ''} deleted`);
      }

      // Reset settings
      resetDisplaySettings();
      resetReplaySettings();
      resetWorkflowDefaults();
      clearApiKeys();

      // Clear localStorage
      if (typeof window !== 'undefined') {
        const keysToRemove: string[] = [];
        for (let i = 0; i < window.localStorage.length; i++) {
          const key = window.localStorage.key(i);
          if (key && key.startsWith('browserAutomation.')) {
            keysToRemove.push(key);
          }
        }
        keysToRemove.forEach((key) => window.localStorage.removeItem(key));
      }
      messages.push('All settings reset');

      setDeleteResult({
        success: !hasErrors,
        message: messages.join('. ') + '.',
      });
    } catch (error) {
      setDeleteResult({
        success: false,
        message: error instanceof Error ? error.message : 'Failed to delete all data',
      });
    } finally {
      setIsDeleting(false);
      setDeleteConfirmation(null);
    }
  }, [deleteAllProjects, deleteAllExports, resetDisplaySettings, resetReplaySettings, resetWorkflowDefaults, clearApiKeys]);

  const handleConfirmDelete = useCallback(() => {
    switch (deleteConfirmation) {
      case 'projects':
        handleDeleteProjects();
        break;
      case 'exports':
        handleDeleteExports();
        break;
      case 'settings':
        handleResetAllSettings();
        break;
      case 'all':
        handleDeleteAll();
        break;
    }
  }, [deleteConfirmation, handleDeleteProjects, handleDeleteExports, handleResetAllSettings, handleDeleteAll]);

  const getDeleteConfirmationTitle = () => {
    switch (deleteConfirmation) {
      case 'projects':
        return 'Delete All Projects';
      case 'exports':
        return 'Delete All Exports';
      case 'settings':
        return 'Reset All Settings';
      case 'all':
        return 'Delete Everything';
      default:
        return '';
    }
  };

  const getDeleteConfirmationMessage = () => {
    switch (deleteConfirmation) {
      case 'projects':
        return `This will permanently delete all ${projects.length} project${projects.length !== 1 ? 's' : ''} and their associated workflows. This action cannot be undone.`;
      case 'exports':
        return `This will permanently delete all ${exports.length} export${exports.length !== 1 ? 's' : ''}. This action cannot be undone.`;
      case 'settings':
        return 'This will reset all display, replay, workflow, and API key settings to their default values. This action cannot be undone.';
      case 'all':
        return 'This will delete ALL projects, workflows, exports, and reset all settings to defaults. This action is irreversible.';
      default:
        return '';
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-6">
        <Database size={24} className="text-red-400" />
        <div>
          <h2 className="text-lg font-semibold text-surface">Data Management</h2>
          <p className="text-sm text-gray-400">Clear and manage your stored data</p>
        </div>
      </div>

      {/* Refresh button */}
      <div className="flex justify-end mb-4">
        <button
          onClick={refreshDataCounts}
          className="flex items-center gap-2 px-3 py-2 text-sm text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
        >
          <RefreshCw size={14} />
          Refresh Counts
        </button>
      </div>

      {/* Result notification */}
      {deleteResult && (
        <div
          className={`p-4 rounded-lg border mb-6 ${
            deleteResult.success
              ? 'bg-green-500/10 border-green-500/30 text-green-300'
              : 'bg-red-500/10 border-red-500/30 text-red-300'
          }`}
        >
          <div className="flex items-start gap-3">
            {deleteResult.success ? (
              <Check size={20} className="flex-shrink-0 mt-0.5" />
            ) : (
              <AlertTriangle size={20} className="flex-shrink-0 mt-0.5" />
            )}
            <p className="text-sm">{deleteResult.message}</p>
          </div>
        </div>
      )}

      {/* Warning banner */}
      <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-4 mb-6">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="text-amber-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-amber-200 font-medium">Danger Zone</p>
            <p className="text-xs text-amber-300/80 mt-1">
              The actions on this page will permanently delete your data. Please make sure you have backups before proceeding.
            </p>
          </div>
        </div>
      </div>

      <SettingSection title="Projects & Workflows" tooltip="Delete all projects and their workflows.">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <FolderX size={20} className="text-gray-400" />
            <div>
              <p className="text-sm text-surface">Delete All Projects</p>
              <p className="text-xs text-gray-500">
                {projects.length === 0
                  ? 'No projects to delete'
                  : `${projects.length} project${projects.length !== 1 ? 's' : ''} will be deleted along with all workflows`}
              </p>
            </div>
          </div>
          <button
            onClick={() => setDeleteConfirmation('projects')}
            disabled={projects.length === 0}
            className="px-4 py-2 text-sm text-red-400 border border-red-500/50 rounded-lg hover:bg-red-500/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Delete All
          </button>
        </div>
      </SettingSection>

      <SettingSection title="Exports" tooltip="Delete all exported recordings and files.">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <FileX size={20} className="text-gray-400" />
            <div>
              <p className="text-sm text-surface">Delete All Exports</p>
              <p className="text-xs text-gray-500">
                {exports.length === 0
                  ? 'No exports to delete'
                  : `${exports.length} export${exports.length !== 1 ? 's' : ''} will be permanently deleted`}
              </p>
            </div>
          </div>
          <button
            onClick={() => setDeleteConfirmation('exports')}
            disabled={exports.length === 0}
            className="px-4 py-2 text-sm text-red-400 border border-red-500/50 rounded-lg hover:bg-red-500/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Delete All
          </button>
        </div>
      </SettingSection>

      <SettingSection title="Settings" tooltip="Reset all settings to defaults.">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <RotateCcw size={20} className="text-gray-400" />
            <div>
              <p className="text-sm text-surface">Reset All Settings</p>
              <p className="text-xs text-gray-500">
                Display, replay, workflow defaults, API keys, and presets will be reset
              </p>
            </div>
          </div>
          <button
            onClick={() => setDeleteConfirmation('settings')}
            className="px-4 py-2 text-sm text-amber-400 border border-amber-500/50 rounded-lg hover:bg-amber-500/10 transition-colors"
          >
            Reset All
          </button>
        </div>
      </SettingSection>

      <div className="relative my-8">
        <div className="absolute inset-0 flex items-center" aria-hidden="true">
          <div className="w-full border-t border-red-500/30"></div>
        </div>
        <div className="relative flex justify-center">
          <span className="bg-flow-bg px-3 text-xs text-red-400 uppercase tracking-wider">Nuclear Option</span>
        </div>
      </div>

      <div className="border border-red-500/30 rounded-lg p-6 bg-red-500/5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            <Trash2 size={24} className="text-red-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-lg font-medium text-surface">Delete Everything</p>
              <p className="text-sm text-gray-400 mt-1">
                This will permanently delete all projects, workflows, exports, and reset all settings.
                Use this to completely start fresh.
              </p>
              <div className="flex flex-wrap gap-2 mt-3">
                <span className="px-2 py-1 text-xs bg-gray-800 text-gray-400 rounded">
                  {projects.length} projects
                </span>
                <span className="px-2 py-1 text-xs bg-gray-800 text-gray-400 rounded">
                  {exports.length} exports
                </span>
                <span className="px-2 py-1 text-xs bg-gray-800 text-gray-400 rounded">
                  All settings
                </span>
              </div>
            </div>
          </div>
          <button
            onClick={() => setDeleteConfirmation('all')}
            className="px-6 py-3 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors flex-shrink-0"
          >
            Delete Everything
          </button>
        </div>
      </div>

      {/* Confirmation Dialog */}
      {deleteConfirmation && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl animate-fade-in">
            <h3 className="text-lg font-semibold text-surface mb-4 flex items-center gap-2">
              <AlertTriangle size={20} className="text-red-400" />
              {getDeleteConfirmationTitle()}
            </h3>
            <p className="text-sm text-gray-400 mb-6">
              {getDeleteConfirmationMessage()}
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setDeleteConfirmation(null)}
                disabled={isDeleting}
                className="flex-1 px-4 py-2 text-subtle hover:text-surface hover:bg-gray-800 rounded-lg transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmDelete}
                disabled={isDeleting}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors flex items-center justify-center gap-2 disabled:opacity-50"
              >
                {isDeleting ? (
                  <>
                    <Loader2 size={16} className="animate-spin" />
                    Deleting...
                  </>
                ) : (
                  <>
                    <Trash2 size={16} />
                    {deleteConfirmation === 'settings' ? 'Reset' : 'Delete'}
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
