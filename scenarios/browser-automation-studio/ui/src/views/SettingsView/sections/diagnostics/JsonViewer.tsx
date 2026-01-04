/**
 * JsonViewer Component
 *
 * Displays JSON data with copy and download functionality.
 */

import { useState, useCallback, useMemo } from 'react';
import {
  ArrowLeft,
  Code,
  Copy,
  Check,
  Download,
} from 'lucide-react';

interface JsonViewerProps {
  data: unknown;
  onBack: () => void;
  title?: string;
}

export function JsonViewer({ data, onBack, title = 'JSON Data' }: JsonViewerProps) {
  const [copied, setCopied] = useState(false);
  const jsonString = useMemo(() => JSON.stringify(data, null, 2), [data]);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(jsonString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [jsonString]);

  const handleDownload = useCallback(() => {
    const blob = new Blob([jsonString], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${title.toLowerCase().replace(/\s+/g, '-')}-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [jsonString, title]);

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="flex items-center gap-1 text-gray-400 hover:text-surface transition-colors"
          >
            <ArrowLeft size={18} />
          </button>
          <div className="flex items-center gap-2">
            <Code size={20} className="text-flow-accent" />
            <h2 className="text-lg font-semibold text-surface">{title}</h2>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleCopy}
            className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-surface bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
            title="Copy to clipboard"
          >
            {copied ? <Check size={16} className="text-emerald-400" /> : <Copy size={16} />}
            {copied ? 'Copied!' : 'Copy'}
          </button>
          <button
            onClick={handleDownload}
            className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-surface bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
            title="Download JSON"
          >
            <Download size={16} />
            Download
          </button>
        </div>
      </div>

      {/* JSON Content */}
      <div className="bg-gray-900 border border-gray-700 rounded-lg overflow-hidden">
        <pre className="p-4 text-sm font-mono text-gray-300 overflow-auto max-h-[70vh] whitespace-pre-wrap break-words">
          {jsonString}
        </pre>
      </div>
    </div>
  );
}

export default JsonViewer;
