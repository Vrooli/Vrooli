import React, { useEffect, useState, useCallback } from 'react';
import {
  Film,
  FileJson,
  Image,
  Globe,
  Download,
  Trash2,
  Eye,
  Pencil,
  RefreshCw,
  Loader2,
  Copy,
  CheckCircle2,
  XCircle,
  Clock,
  Sparkles,
  Share2,
} from 'lucide-react';
import { useExportStore, type Export } from '@stores/exportStore';
import { useDashboardStore } from '@stores/dashboardStore';
import { formatDistanceToNow } from 'date-fns';
import { toast } from 'react-hot-toast';
import { ResponsiveDialog } from '@shared/layout';
import { TabEmptyState } from './TabEmptyState';
import { ExportsEmptyPreview } from './ExportsEmptyPreview';

interface ExportsTabProps {
  onViewExecution: (executionId: string, workflowId: string) => void;
  onNavigateToWorkflow?: (projectId: string, workflowId: string) => void;
  onNavigateToExecutions?: () => void;
  onNavigateToHome?: () => void;
  onCreateWorkflow?: () => void;
}

const formatConfig: Record<string, {
  icon: typeof Film;
  color: string;
  bgColor: string;
  label: string;
}> = {
  mp4: {
    icon: Film,
    color: 'text-purple-400',
    bgColor: 'bg-purple-500/10',
    label: 'MP4 Video',
  },
  gif: {
    icon: Image,
    color: 'text-pink-400',
    bgColor: 'bg-pink-500/10',
    label: 'Animated GIF',
  },
  json: {
    icon: FileJson,
    color: 'text-blue-400',
    bgColor: 'bg-blue-500/10',
    label: 'JSON Package',
  },
  html: {
    icon: Globe,
    color: 'text-green-400',
    bgColor: 'bg-green-500/10',
    label: 'HTML Bundle',
  },
};

const statusConfig: Record<string, {
  icon: typeof Clock;
  color: string;
  label: string;
}> = {
  pending: {
    icon: Clock,
    color: 'text-gray-400',
    label: 'Pending',
  },
  processing: {
    icon: Loader2,
    color: 'text-yellow-400',
    label: 'Processing',
  },
  completed: {
    icon: CheckCircle2,
    color: 'text-green-400',
    label: 'Ready',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-400',
    label: 'Failed',
  },
};

const formatFileSize = (bytes?: number): string => {
  if (!bytes) return 'Unknown size';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

const formatDuration = (ms?: number): string => {
  if (!ms) return '';
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
};

export const ExportsTab: React.FC<ExportsTabProps> = ({
  onViewExecution,
  onNavigateToExecutions,
  onNavigateToHome,
  onCreateWorkflow,
}) => {
  const {
    exports,
    isLoading,
    isDeleting,
    fetchExports,
    deleteExport,
    updateExport,
  } = useExportStore();
  const {
    recentWorkflows,
    recentExecutions,
    runningExecutions,
  } = useDashboardStore();
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [selectedExport, setSelectedExport] = useState<Export | null>(null);
  const [isRenameDialogOpen, setIsRenameDialogOpen] = useState(false);
  const [newName, setNewName] = useState('');
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [exportToDelete, setExportToDelete] = useState<Export | null>(null);
  const hasExecutions = recentExecutions.length > 0 || runningExecutions.length > 0;
  const totalRuns = recentExecutions.length + runningExecutions.length;
  const hasWorkflows = recentWorkflows.length > 0;

  useEffect(() => {
    void fetchExports();
  }, [fetchExports]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await fetchExports();
    setIsRefreshing(false);
  }, [fetchExports]);

  const handleDownload = useCallback(async (export_: Export) => {
    if (!export_.storageUrl) {
      toast.error('Export file not available');
      return;
    }

    try {
      const response = await fetch(export_.storageUrl);
      if (!response.ok) throw new Error('Download failed');

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${export_.name}.${export_.format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      toast.success('Download started');
    } catch (error) {
      toast.error('Failed to download export');
    }
  }, []);

  const handleRename = useCallback(async () => {
    if (!selectedExport || !newName.trim()) return;

    const result = await updateExport(selectedExport.id, { name: newName.trim() });
    if (result) {
      toast.success('Export renamed');
      setIsRenameDialogOpen(false);
      setSelectedExport(null);
      setNewName('');
    }
  }, [selectedExport, newName, updateExport]);

  const handleDelete = useCallback(async () => {
    if (!exportToDelete) return;

    const success = await deleteExport(exportToDelete.id);
    if (success) {
      toast.success('Export deleted');
      setIsDeleteDialogOpen(false);
      setExportToDelete(null);
    }
  }, [exportToDelete, deleteExport]);

  const handleCopyCaption = useCallback(async (caption: string) => {
    try {
      await navigator.clipboard.writeText(caption);
      toast.success('Caption copied to clipboard');
    } catch (error) {
      toast.error('Failed to copy caption');
    }
  }, []);

  const openRenameDialog = useCallback((export_: Export) => {
    setSelectedExport(export_);
    setNewName(export_.name);
    setIsRenameDialogOpen(true);
  }, []);

  const openDeleteDialog = useCallback((export_: Export) => {
    setExportToDelete(export_);
    setIsDeleteDialogOpen(true);
  }, []);

  const renderExportCard = (export_: Export) => {
    const formatCfg = formatConfig[export_.format] ?? formatConfig.json;
    const statusCfg = statusConfig[export_.status] ?? statusConfig.completed;
    const FormatIcon = formatCfg.icon;
    const StatusIcon = statusCfg.icon;

    return (
      <div
        key={export_.id}
        className="group relative flex flex-col bg-gray-800/50 border border-gray-700/50 hover:border-gray-600 rounded-lg overflow-hidden transition-all"
      >
        {/* Thumbnail or format icon */}
        <div className={`relative h-32 ${formatCfg.bgColor} flex items-center justify-center`}>
          {export_.thumbnailUrl ? (
            <img
              src={export_.thumbnailUrl}
              alt={export_.name}
              className="w-full h-full object-cover"
            />
          ) : (
            <FormatIcon size={40} className={formatCfg.color} />
          )}

          {/* Status badge */}
          <div className={`absolute top-2 right-2 flex items-center gap-1 px-2 py-1 rounded-full bg-gray-900/80 backdrop-blur-sm ${statusCfg.color} text-xs`}>
            <StatusIcon size={12} className={export_.status === 'processing' ? 'animate-spin' : ''} />
            {statusCfg.label}
          </div>

          {/* Hover overlay with actions */}
          <div className="absolute inset-0 bg-gray-900/80 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
            {export_.status === 'completed' && export_.storageUrl && (
              <button
                onClick={() => handleDownload(export_)}
                className="p-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
                title="Download"
              >
                <Download size={18} />
              </button>
            )}
            <button
              onClick={() => onViewExecution(export_.executionId, export_.workflowId ?? '')}
              className="p-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600 transition-colors"
              title="View execution"
            >
              <Eye size={18} />
            </button>
            <button
              onClick={() => openRenameDialog(export_)}
              className="p-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600 transition-colors"
              title="Rename"
            >
              <Pencil size={18} />
            </button>
            <button
              onClick={() => openDeleteDialog(export_)}
              className="p-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
              title="Delete"
            >
              <Trash2 size={18} />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-3 flex-1 flex flex-col">
          <h4 className="font-medium text-white truncate" title={export_.name}>
            {export_.name}
          </h4>

          {export_.workflowName && (
            <p className="text-xs text-gray-500 truncate mt-0.5">
              {export_.workflowName}
            </p>
          )}

          <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
            <span className={formatCfg.color}>{formatCfg.label}</span>
            {export_.fileSizeBytes && (
              <>
                <span>·</span>
                <span>{formatFileSize(export_.fileSizeBytes)}</span>
              </>
            )}
            {export_.durationMs && (
              <>
                <span>·</span>
                <span>{formatDuration(export_.durationMs)}</span>
              </>
            )}
          </div>

          <p className="text-xs text-gray-500 mt-1">
            {formatDistanceToNow(export_.createdAt, { addSuffix: true })}
          </p>

          {/* AI Caption section */}
          {export_.aiCaption && (
            <div className="mt-3 p-2 bg-gray-700/50 rounded-lg border border-gray-600/50">
              <div className="flex items-center gap-1.5 text-xs text-purple-400 mb-1">
                <Sparkles size={12} />
                <span>AI Caption</span>
              </div>
              <p className="text-xs text-gray-300 line-clamp-2">
                {export_.aiCaption}
              </p>
              <button
                onClick={() => handleCopyCaption(export_.aiCaption!)}
                className="mt-1.5 flex items-center gap-1 text-xs text-gray-400 hover:text-white transition-colors"
              >
                <Copy size={10} />
                Copy
              </button>
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold text-white">Exports Library</h2>
          <p className="text-sm text-gray-400">
            Your exported replay videos, GIFs, and packages
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isRefreshing || isLoading}
          className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
        >
          <RefreshCw size={14} className={isRefreshing ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {/* Exports Grid */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 size={24} className="animate-spin text-gray-500" />
        </div>
      ) : exports.length === 0 ? (
        <TabEmptyState
          icon={<Film size={22} />}
          title="Share your automations with the world"
          subtitle={
            hasExecutions
              ? `You’ve run ${totalRuns} workflow${totalRuns !== 1 ? 's' : ''}. Export recordings as video, GIF, or data packages.`
              : hasWorkflows
                ? `Run a workflow to generate replay videos and exportable datasets.`
                : 'Create your first workflow, run it, then export ready-to-share artifacts.'
          }
          preview={<ExportsEmptyPreview />}
          primaryCta={{
            label: hasExecutions
              ? 'Open executions to export'
              : hasWorkflows
                ? 'Run a workflow'
                : 'Create your first workflow',
            onClick:
              (hasExecutions ? onNavigateToExecutions : hasWorkflows ? onNavigateToHome : onCreateWorkflow) ??
              onCreateWorkflow ??
              (() => {}),
          }}
          secondaryCta={
            hasExecutions && onNavigateToHome
              ? { label: 'Run another workflow', onClick: onNavigateToHome }
              : hasWorkflows && onCreateWorkflow
                ? { label: 'Create another workflow', onClick: onCreateWorkflow }
                : undefined
          }
          progressPath={[
            { label: 'Create workflow', completed: hasWorkflows || hasExecutions },
            { label: 'Run & record', active: hasWorkflows && !hasExecutions, completed: hasExecutions },
            { label: 'Export & share', active: hasExecutions },
          ]}
          features={[
            {
              title: 'Video replays',
              description: 'MP4/WebM with full browser context for walkthroughs and demos.',
              icon: <Film size={16} />,
            },
            {
              title: 'GIF highlights',
              description: 'Lightweight loops perfect for tickets and docs.',
              icon: <Image size={16} />,
            },
            {
              title: 'Data packages',
              description: 'JSON/CSV bundles with logs, timings, and captured values.',
              icon: <FileJson size={16} />,
            },
            {
              title: 'Share & sync',
              description: 'Send exports to teams or plug into CI pipelines.',
              icon: <Share2 size={16} />,
            },
          ]}
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {exports.map((export_) => renderExportCard(export_))}
        </div>
      )}

      {/* Rename Dialog */}
      <ResponsiveDialog
        isOpen={isRenameDialogOpen}
        onDismiss={() => {
          setIsRenameDialogOpen(false);
          setSelectedExport(null);
          setNewName('');
        }}
        ariaLabel="Rename Export"
      >
        <div className="space-y-4">
          <input
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Export name"
            className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder:text-gray-500 focus:border-flow-accent focus:outline-none"
            autoFocus
          />
          <div className="flex justify-end gap-2">
            <button
              onClick={() => {
                setIsRenameDialogOpen(false);
                setSelectedExport(null);
                setNewName('');
              }}
              className="px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleRename}
              disabled={!newName.trim()}
              className="px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50"
            >
              Save
            </button>
          </div>
        </div>
      </ResponsiveDialog>

      {/* Delete Confirmation Dialog */}
      <ResponsiveDialog
        isOpen={isDeleteDialogOpen}
        onDismiss={() => {
          setIsDeleteDialogOpen(false);
          setExportToDelete(null);
        }}
        ariaLabel="Delete Export"
      >
        <div className="space-y-4">
          <p className="text-gray-300">
            Are you sure you want to delete "{exportToDelete?.name}"? This action cannot be undone.
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => {
                setIsDeleteDialogOpen(false);
                setExportToDelete(null);
              }}
              className="px-4 py-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleDelete}
              disabled={isDeleting}
              className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
            >
              {isDeleting ? (
                <Loader2 size={16} className="animate-spin" />
              ) : (
                'Delete'
              )}
            </button>
          </div>
        </div>
      </ResponsiveDialog>
    </div>
  );
};

export default ExportsTab;
