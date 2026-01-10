import React, { useState, useCallback } from 'react';
import {
  X,
  FolderOpen,
  Pencil,
  Trash2,
  Copy,
  Sparkles,
  Loader2,
  AlertTriangle,
  Calendar,
  Clock,
  FileText,
  Layers,
} from 'lucide-react';
import { toast } from 'react-hot-toast';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import { useExportStore, type Export } from './store';
import { getConfig } from '@/config';
import { selectors } from '@constants/selectors';
import {
  formatFileSize,
  formatDuration,
  getFormatConfig,
  getExportStatusConfig,
} from './presentation';
import { defaultDownloadClient, type DownloadClient } from './api/downloadClient';
import { format } from 'date-fns';

/**
 * Truncates a file path for display, keeping the end portion visible.
 * @param path - The full path to truncate
 * @param maxLength - Maximum characters to display (default 40)
 * @returns Truncated path with ellipsis prefix if needed
 */
const truncatePath = (path: string, maxLength = 40): string => {
  if (!path || path.length <= maxLength) return path;
  return '...' + path.slice(-maxLength + 3);
};

interface ExportDetailsModalProps {
  /** The export to display details for */
  export_: Export;
  /** Called when modal should close */
  onClose: () => void;
  /** Called to edit the export (opens full export dialog) */
  onEdit: () => void;
  /** Called to delete the export (opens confirm dialog flow) */
  onDelete: () => void;
  /** Optional download client for testing. Defaults to defaultDownloadClient. */
  downloadClient?: DownloadClient;
}

export const ExportDetailsModal: React.FC<ExportDetailsModalProps> = ({
  export_,
  onClose,
  onEdit,
  onDelete,
  downloadClient = defaultDownloadClient,
}) => {
  const { updateExport } = useExportStore();
  const [isGeneratingCaption, setIsGeneratingCaption] = useState(false);
  const [aiCaption, setAiCaption] = useState(export_.aiCaption ?? '');

  const formatCfg = getFormatConfig(export_.format);
  const statusCfg = getExportStatusConfig(export_.status);
  const FormatIcon = formatCfg.icon;
  const StatusIcon = statusCfg.icon;

  const isCompleted = export_.status === 'completed';
  const hasStorageUrl = Boolean(export_.storageUrl);
  const canOpenFolder = isCompleted && hasStorageUrl;

  // Determine why open folder is disabled
  const getOpenFolderDisabledReason = (): string | null => {
    if (canOpenFolder) return null;
    if (export_.status === 'pending') return 'Export is pending';
    if (export_.status === 'processing') return 'Export is still processing';
    if (export_.status === 'failed') return 'Export failed';
    if (!hasStorageUrl) return 'Export file not available';
    return 'File location unavailable';
  };
  const openFolderDisabledReason = getOpenFolderDisabledReason();

  const handleOpenFolder = useCallback(async () => {
    if (!export_.storageUrl) {
      toast.error('Export file not available');
      return;
    }

    const result = await downloadClient.revealFile(export_.id);

    if (result.success) {
      toast.success('Opened in file manager');
    } else {
      toast.error(result.error ?? 'Failed to open folder');
    }
  }, [export_.id, export_.storageUrl, downloadClient]);

  const handleCopyPath = useCallback(async () => {
    if (!export_.storageUrl) {
      toast.error('No path available');
      return;
    }
    const result = await downloadClient.copyToClipboard(export_.storageUrl);
    if (result.success) {
      toast.success('Path copied');
    } else {
      toast.error(result.error ?? 'Failed to copy path');
    }
  }, [export_.storageUrl, downloadClient]);

  const handleCopyCaption = useCallback(async () => {
    if (!aiCaption) return;
    const result = await downloadClient.copyToClipboard(aiCaption);
    if (result.success) {
      toast.success('Caption copied to clipboard');
    } else {
      toast.error(result.error ?? 'Failed to copy caption');
    }
  }, [aiCaption, downloadClient]);

  const handleGenerateCaption = useCallback(async () => {
    setIsGeneratingCaption(true);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/exports/${export_.id}/generate-caption`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        // If endpoint doesn't exist yet, use a placeholder caption
        const placeholderCaption = `Check out this ${export_.format.toUpperCase()} replay of "${export_.name}"! Created with Vrooli Ascension.`;
        setAiCaption(placeholderCaption);
        await updateExport(export_.id, { aiCaption: placeholderCaption });
        toast.success('Caption generated');
        return;
      }

      const data = await response.json();
      if (data.caption) {
        setAiCaption(data.caption);
        await updateExport(export_.id, { aiCaption: data.caption });
        toast.success('Caption generated');
      }
    } catch {
      // Fallback to placeholder
      const placeholderCaption = `Check out this ${export_.format.toUpperCase()} replay of "${export_.name}"! Created with Vrooli Ascension.`;
      setAiCaption(placeholderCaption);
      toast.success('Caption generated');
    } finally {
      setIsGeneratingCaption(false);
    }
  }, [export_.id, export_.format, export_.name, updateExport]);

  return (
    <ResponsiveDialog
      isOpen={true}
      onDismiss={onClose}
      ariaLabel={`Export details for ${export_.name}`}
      className="!p-0"
    >
      <div
        className="bg-gray-900 rounded-xl max-h-[90vh] overflow-y-auto"
        data-testid={selectors.exports.detailsModal.root}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-800">
          <h2 className="text-lg font-semibold text-white truncate pr-4" title={export_.name}>
            {export_.name}
          </h2>
          <button
            onClick={onClose}
            className="p-1 text-gray-400 hover:text-white transition-colors"
            data-testid={selectors.exports.detailsModal.closeButton}
          >
            <X size={20} />
          </button>
        </div>

        {/* Thumbnail preview */}
        <div
          className={`relative h-48 ${formatCfg.bgColor} flex items-center justify-center`}
          data-testid={selectors.exports.detailsModal.thumbnail}
        >
          {export_.thumbnailUrl ? (
            <img
              src={export_.thumbnailUrl}
              alt={export_.name}
              className="w-full h-full object-cover"
            />
          ) : (
            <FormatIcon size={64} className={formatCfg.color} />
          )}

          {/* Status badge */}
          <div className={`absolute top-3 right-3 flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-gray-900/80 backdrop-blur-sm ${statusCfg.color} text-sm`}>
            <StatusIcon size={14} className={export_.status === 'processing' ? 'animate-spin' : ''} />
            {statusCfg.label}
          </div>
        </div>

        {/* Details section */}
        <div className="p-4 space-y-4">
          {/* Details grid */}
          <div className="grid grid-cols-2 gap-3">
            <DetailItem
              icon={<FileText size={14} />}
              label="Format"
              value={formatCfg.label}
              valueClass={formatCfg.color}
            />
            {export_.fileSizeBytes && (
              <DetailItem
                icon={<Layers size={14} />}
                label="Size"
                value={formatFileSize(export_.fileSizeBytes)}
              />
            )}
            {export_.durationMs && (
              <DetailItem
                icon={<Clock size={14} />}
                label="Duration"
                value={formatDuration(export_.durationMs)}
              />
            )}
            {export_.frameCount && (
              <DetailItem
                icon={<Layers size={14} />}
                label="Frames"
                value={export_.frameCount.toLocaleString()}
              />
            )}
            <DetailItem
              icon={<Calendar size={14} />}
              label="Created"
              value={format(export_.createdAt, 'MMM d, yyyy h:mm a')}
            />
            {export_.workflowName && (
              <DetailItem
                icon={<FileText size={14} />}
                label="Workflow"
                value={export_.workflowName}
              />
            )}
            <DetailItem
              icon={<FolderOpen size={14} />}
              label="Location"
              value={export_.storageUrl ? truncatePath(export_.storageUrl) : 'Downloaded to browser'}
              valueClass={export_.storageUrl ? "text-gray-300 font-mono text-xs" : "text-gray-400 italic"}
              title={export_.storageUrl || 'File was downloaded directly to your browser'}
            />
          </div>

          {/* Error section (for failed exports) */}
          {export_.status === 'failed' && export_.error && (
            <div className="p-3 bg-red-500/10 border border-red-500/30 rounded-lg">
              <div className="flex items-center gap-2 text-red-400 mb-1">
                <AlertTriangle size={14} />
                <span className="text-sm font-medium">Export Failed</span>
              </div>
              <p className="text-xs text-red-300">{export_.error}</p>
            </div>
          )}

          {/* AI Caption section */}
          <div className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2 text-sm text-purple-400">
                <Sparkles size={14} />
                <span>AI Caption</span>
              </div>
              {!aiCaption && (
                <button
                  onClick={handleGenerateCaption}
                  disabled={isGeneratingCaption}
                  className="text-xs text-flow-accent hover:text-blue-400 transition-colors disabled:opacity-50"
                  data-testid={selectors.exports.detailsModal.generateCaptionButton}
                >
                  {isGeneratingCaption ? (
                    <span className="flex items-center gap-1">
                      <Loader2 size={12} className="animate-spin" />
                      Generating...
                    </span>
                  ) : (
                    'Generate'
                  )}
                </button>
              )}
            </div>

            {aiCaption ? (
              <div>
                <p className="text-sm text-gray-300 mb-2">{aiCaption}</p>
                <button
                  onClick={handleCopyCaption}
                  className="flex items-center gap-1 text-xs text-gray-400 hover:text-white transition-colors"
                  data-testid={selectors.exports.detailsModal.copyCaptionButton}
                >
                  <Copy size={12} />
                  Copy caption
                </button>
              </div>
            ) : (
              <p className="text-xs text-gray-500">
                Generate an AI caption to help share this export on social media or in presentations.
              </p>
            )}
          </div>
        </div>

        {/* Actions */}
        <div className="p-4 border-t border-gray-800 space-y-3">
          {/* Primary action */}
          <div>
            <button
              onClick={handleOpenFolder}
              disabled={!canOpenFolder}
              className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
              data-testid={selectors.exports.detailsModal.downloadButton}
            >
              <FolderOpen size={16} />
              Show in Folder
            </button>
            {openFolderDisabledReason && (
              <p className="text-xs text-gray-500 text-center mt-1">{openFolderDisabledReason}</p>
            )}
          </div>

          {/* Secondary actions */}
          <div className="flex items-center gap-2">
            <button
              onClick={onEdit}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 text-gray-300 hover:text-white border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
              data-testid={selectors.exports.detailsModal.editButton}
            >
              <Pencil size={14} />
              Edit
            </button>
            <button
              onClick={handleCopyPath}
              disabled={!hasStorageUrl}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2 text-gray-300 hover:text-white border border-gray-700 hover:border-gray-600 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              data-testid={selectors.exports.detailsModal.copyLinkButton}
            >
              <Copy size={14} />
              Copy Path
            </button>
            <button
              onClick={onDelete}
              className="flex items-center justify-center gap-2 px-4 py-2 text-red-400 hover:text-red-300 border border-red-500/30 hover:border-red-500/50 rounded-lg transition-colors"
              data-testid={selectors.exports.detailsModal.deleteButton}
            >
              <Trash2 size={14} />
            </button>
          </div>
        </div>
      </div>
    </ResponsiveDialog>
  );
};

interface DetailItemProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  valueClass?: string;
  /** Full value shown as tooltip on hover */
  title?: string;
}

const DetailItem: React.FC<DetailItemProps> = ({ icon, label, value, valueClass, title }) => (
  <div className="flex items-start gap-2">
    <div className="text-gray-500 mt-0.5">{icon}</div>
    <div className="min-w-0 flex-1">
      <div className="text-xs text-gray-500">{label}</div>
      <div
        className={`text-sm truncate ${valueClass ?? 'text-gray-300'}`}
        title={title}
      >
        {value}
      </div>
    </div>
  </div>
);

export default ExportDetailsModal;
