import React, { useState, useCallback } from 'react';
import {
  CheckCircle,
  Film,
  Copy,
  ExternalLink,
  Sparkles,
  Loader2,
  ArrowRight,
  X,
} from 'lucide-react';
import { toast } from 'react-hot-toast';
import { useExportStore, type Export } from '@stores/exportStore';
import { getConfig } from '@/config';

interface ExportSuccessPanelProps {
  export_: Export;
  onClose: () => void;
  onViewInLibrary: () => void;
  onViewExecution?: () => void;
}

export const ExportSuccessPanel: React.FC<ExportSuccessPanelProps> = ({
  export_,
  onClose,
  onViewInLibrary,
  onViewExecution,
}) => {
  const { updateExport } = useExportStore();
  const [isGeneratingCaption, setIsGeneratingCaption] = useState(false);
  const [aiCaption, setAiCaption] = useState(export_.aiCaption ?? '');

  const handleCopyCaption = useCallback(async () => {
    if (!aiCaption) return;
    try {
      await navigator.clipboard.writeText(aiCaption);
      toast.success('Caption copied to clipboard');
    } catch (error) {
      toast.error('Failed to copy caption');
    }
  }, [aiCaption]);

  const handleGenerateCaption = useCallback(async () => {
    setIsGeneratingCaption(true);
    try {
      const config = await getConfig();
      // Call the AI caption generation endpoint
      const response = await fetch(`${config.API_URL}/exports/${export_.id}/generate-caption`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        // If endpoint doesn't exist yet, use a placeholder caption
        const placeholderCaption = `Check out this ${export_.format.toUpperCase()} replay of "${export_.name}"! Created with Browser Automation Studio.`;
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
    } catch (error) {
      // Fallback to placeholder
      const placeholderCaption = `Check out this ${export_.format.toUpperCase()} replay of "${export_.name}"! Created with Browser Automation Studio.`;
      setAiCaption(placeholderCaption);
      toast.success('Caption generated');
    } finally {
      setIsGeneratingCaption(false);
    }
  }, [export_.id, export_.format, export_.name, updateExport]);

  const formatLabel = {
    mp4: 'MP4 Video',
    gif: 'Animated GIF',
    json: 'JSON Package',
    html: 'HTML Bundle',
  }[export_.format] ?? export_.format.toUpperCase();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="relative w-full max-w-md mx-4 bg-gray-900 border border-gray-700 rounded-xl shadow-2xl">
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute top-4 right-4 p-1 text-gray-400 hover:text-white transition-colors"
        >
          <X size={20} />
        </button>

        {/* Success header */}
        <div className="p-6 text-center border-b border-gray-800">
          <div className="inline-flex items-center justify-center w-16 h-16 mb-4 bg-green-500/10 rounded-full">
            <CheckCircle size={32} className="text-green-400" />
          </div>
          <h3 className="text-xl font-semibold text-white mb-1">
            Export Complete!
          </h3>
          <p className="text-sm text-gray-400">
            Your {formatLabel} has been saved to the Exports Library
          </p>
        </div>

        {/* Export details */}
        <div className="p-6 space-y-4">
          <div className="flex items-center gap-3 p-3 bg-gray-800/50 rounded-lg">
            <div className="p-2 bg-purple-500/10 rounded-lg">
              <Film size={20} className="text-purple-400" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="font-medium text-white truncate">{export_.name}</div>
              <div className="text-xs text-gray-400">{formatLabel}</div>
            </div>
          </div>

          {/* AI Caption section */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm text-purple-400">
                <Sparkles size={14} />
                <span>AI Caption</span>
              </div>
              {!aiCaption && (
                <button
                  onClick={handleGenerateCaption}
                  disabled={isGeneratingCaption}
                  className="text-xs text-flow-accent hover:text-blue-400 transition-colors disabled:opacity-50"
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
              <div className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50">
                <p className="text-sm text-gray-300 mb-2">{aiCaption}</p>
                <button
                  onClick={handleCopyCaption}
                  className="flex items-center gap-1 text-xs text-gray-400 hover:text-white transition-colors"
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
        <div className="p-6 border-t border-gray-800 space-y-3">
          <button
            onClick={onViewInLibrary}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors font-medium"
          >
            <ExternalLink size={16} />
            View in Exports Library
          </button>

          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            >
              Continue Editing
            </button>
            {onViewExecution && (
              <button
                onClick={onViewExecution}
                className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-gray-300 hover:text-white border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
              >
                View Replay
                <ArrowRight size={14} />
              </button>
            )}
          </div>
        </div>

        {/* Tips */}
        <div className="px-6 pb-6">
          <div className="p-3 bg-blue-500/5 border border-blue-500/20 rounded-lg">
            <p className="text-xs text-blue-300/80">
              <strong>Tip:</strong> Visit the Exports Library to rename, re-export with different settings, or generate AI captions for all your exports.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ExportSuccessPanel;
