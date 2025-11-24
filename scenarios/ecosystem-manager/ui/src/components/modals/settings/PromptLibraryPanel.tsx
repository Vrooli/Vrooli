import { useEffect, useMemo, useState } from 'react';
import { FileText, RefreshCw, Save, FolderOpen, Eye, Copy } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { usePromptFile, usePromptFiles, useSavePromptFile } from '@/hooks/usePromptFiles';
import { markdownToHtml } from '@/lib/markdown';
import type { PromptFileInfo } from '@/types/api';

function formatBytes(size?: number) {
  if (!size) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let s = size;
  let idx = 0;
  while (s >= 1024 && idx < units.length - 1) {
    s /= 1024;
    idx++;
  }
  return `${s.toFixed(idx === 0 ? 0 : 2)} ${units[idx]}`;
}

export function PromptLibraryPanel() {
  const { data: files = [], isLoading: filesLoading } = usePromptFiles();
  const [selectedId, setSelectedId] = useState<string>();
  const { data: file, isFetching: fileLoading, refetch } = usePromptFile(selectedId);
  const savePrompt = useSavePromptFile();
  const [draft, setDraft] = useState('');

  useEffect(() => {
    if (files.length > 0 && !selectedId) {
      setSelectedId(files[0].id);
    }
  }, [files, selectedId]);

  useEffect(() => {
    setDraft(file?.content ?? '');
  }, [file?.id, file?.content]);

  const currentInfo: PromptFileInfo | undefined = files.find((f) => f.id === selectedId);
  const isDirty = draft !== (file?.content ?? '');
  const renderedPreview = useMemo(() => markdownToHtml(draft), [draft]);

  const handleSave = () => {
    if (!selectedId) return;
    savePrompt.mutate(
      { id: selectedId, content: draft },
      {
        onSuccess: () => {
          refetch();
        },
      }
    );
  };

  const handleReset = () => {
    if (file) {
      setDraft(file.content);
      refetch();
    }
  };

  const modifiedLabel = currentInfo?.modified_at
    ? new Date(currentInfo.modified_at).toLocaleString()
    : '—';

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 text-sm text-slate-400">
        <FileText className="h-4 w-4" />
        <span>Browse, edit, and preview raw prompt files</span>
      </div>

      <div className="grid gap-4 md:grid-cols-[280px_1fr]">
        <div className="space-y-3">
          <div className="space-y-2">
            <Label>Prompt File</Label>
            <Select
              value={selectedId || ''}
              onValueChange={setSelectedId}
              disabled={filesLoading || files.length === 0}
            >
              <SelectTrigger>
                <SelectValue placeholder={filesLoading ? 'Loading...' : 'Select file'} />
              </SelectTrigger>
              <SelectContent>
                {files.map((fileInfo) => (
                  <SelectItem key={fileInfo.id} value={fileInfo.id}>
                    {fileInfo.display_name || fileInfo.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2 rounded-md border border-white/10 bg-slate-900 p-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-slate-400">Path</span>
              <code className="text-xs bg-white/5 px-2 py-1 rounded border border-white/10">
                {currentInfo?.path || '—'}
              </code>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-slate-400">Type</span>
              <span className="text-slate-100">{currentInfo?.type || '—'}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-slate-400">Size</span>
              <span className="text-slate-100">{formatBytes(currentInfo?.size)}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-slate-400">Last Modified</span>
              <span className="text-slate-100">{modifiedLabel}</span>
            </div>
          </div>

          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigator.clipboard.writeText(currentInfo?.path || '')}
              disabled={!currentInfo}
            >
              <FolderOpen className="h-4 w-4 mr-2" />
              Copy Path
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigator.clipboard.writeText(draft || '')}
              disabled={!draft}
            >
              <Copy className="h-4 w-4 mr-2" />
              Copy content
            </Button>
          </div>
        </div>

        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label className="text-sm">Editor</Label>
              <p className="text-xs text-slate-400">
                Changes are saved directly to the prompt file. Keep markdown formatting intact.
              </p>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={handleReset} disabled={!isDirty || fileLoading}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Revert
              </Button>
              <Button
                size="sm"
                onClick={handleSave}
                disabled={!isDirty || savePrompt.isPending || !selectedId}
              >
                {savePrompt.isPending ? (
                  'Saving...'
                ) : (
                  <>
                    <Save className="h-4 w-4 mr-2" />
                    Save
                  </>
                )}
              </Button>
            </div>
          </div>

          <Textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            className="font-mono text-sm min-h-[320px] bg-slate-900"
            spellCheck={false}
            placeholder={fileLoading ? 'Loading prompt...' : 'Select a prompt file to edit'}
          />

          <div className="space-y-2">
            <div className="flex items-center gap-2 text-xs uppercase text-slate-400">
              <Eye className="h-4 w-4" />
              <span>Rendered Preview</span>
            </div>
            <div className="rounded-md border border-white/10 bg-card p-4 max-h-[260px] overflow-y-auto">
              {draft ? (
                <div
                  className="prose prose-invert prose-sm max-w-none space-y-3 [&_code]:bg-black/40 [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded [&_pre]:text-xs [&_pre]:leading-relaxed [&_ul]:list-disc [&_ul]:pl-5"
                  dangerouslySetInnerHTML={{ __html: renderedPreview }}
                />
              ) : (
                <div className="text-slate-500 text-sm">
                  {fileLoading ? 'Loading preview...' : 'No content to preview'}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
