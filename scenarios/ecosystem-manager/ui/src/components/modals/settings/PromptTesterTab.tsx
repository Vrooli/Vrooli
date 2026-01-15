/**
 * PromptTesterTab Component
 * Test and preview assembled prompts for tasks
 */

import { useMemo, useState } from 'react';
import { FileText, Copy, CheckCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs';
import { usePromptPreview } from '@/hooks/usePromptTester';
import { markdownToHtml } from '@/lib/markdown';
import { PromptLibraryPanel } from './PromptLibraryPanel';
import { SteeringConfigPicker, extractSteeringFields } from '@/components/steer/SteeringConfigPicker';
import type { TaskType, OperationType, Priority, SteeringConfig } from '@/types/api';

const PRIORITIES: Priority[] = ['critical', 'high', 'medium', 'low'];

export function PromptTesterTab() {
  const [activeSubTab, setActiveSubTab] = useState('preview');
  const [type, setType] = useState<TaskType>('resource');
  const [operation, setOperation] = useState<OperationType>('generator');
  const [title, setTitle] = useState('Test Task');
  const [priority, setPriority] = useState<Priority>('medium');
  const [notes, setNotes] = useState('');
  const [target, setTarget] = useState('');
  const [steeringConfig, setSteeringConfig] = useState<SteeringConfig>({ strategy: 'none' });
  const [copied, setCopied] = useState(false);

  const { mutate: previewPrompt, data: promptData, isPending } = usePromptPreview();

  const promptText = promptData?.prompt || '';
  const characterCount = promptText.length;
  const wordCount = promptText.trim() ? promptText.trim().split(/\s+/).length : 0;
  const lineCount = promptText ? promptText.split(/\r?\n/).length : 0;
  const promptPreviewHtml = useMemo(() => markdownToHtml(promptText), [promptText]);

  const handlePreview = () => {
    const steeringFields = type === 'scenario' && operation === 'improver'
      ? extractSteeringFields(steeringConfig)
      : {};

    previewPrompt({
      type,
      operation,
      title,
      priority,
      notes: notes || undefined,
      target: type === 'scenario' ? target.trim() || undefined : undefined,
      targets: type === 'scenario' && target.trim() ? [target.trim()] : undefined,
      ...steeringFields,
    });
  };

  const handleCopy = () => {
    if (promptData?.prompt) {
      navigator.clipboard.writeText(promptData.prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <Tabs value={activeSubTab} onValueChange={setActiveSubTab} className="space-y-6">
      <TabsList className="grid w-full grid-cols-2">
        <TabsTrigger value="preview">Prompt Preview</TabsTrigger>
        <TabsTrigger value="library">Prompt Library</TabsTrigger>
      </TabsList>

      <TabsContent value="preview" className="space-y-6">
        {/* Tester Configuration */}
        <div className="space-y-4">
          <div className="flex items-center gap-2 text-sm text-slate-400">
            <FileText className="h-4 w-4" />
            <span>Configure a test task to preview its assembled prompt</span>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="pt-type">Type</Label>
              <Select
                value={type}
                onValueChange={(val: string) => {
                  setType(val as TaskType);
                  if (val !== 'scenario') {
                    setTarget('');
                    setSteeringConfig({ strategy: 'none' });
                  }
                }}
              >
                <SelectTrigger id="pt-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="resource">Resource</SelectItem>
                  <SelectItem value="scenario">Scenario</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="pt-operation">Operation</Label>
              <Select
                value={operation}
                onValueChange={(val: string) => {
                  setOperation(val as OperationType);
                  if (val !== 'improver') {
                    setSteeringConfig({ strategy: 'none' });
                  }
                }}
              >
                <SelectTrigger id="pt-operation">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="generator">Generator</SelectItem>
                  <SelectItem value="improver">Improver</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {type === 'scenario' && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="pt-target">Scenario Target</Label>
                <Input
                  id="pt-target"
                  value={target}
                  onChange={(e) => setTarget(e.target.value)}
                  placeholder="e.g. system-monitor"
                />
              </div>

              {operation === 'improver' && (
                <div className="space-y-2">
                  <Label>Steering Configuration</Label>
                  <SteeringConfigPicker
                    value={steeringConfig}
                    onChange={setSteeringConfig}
                  />
                </div>
              )}
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="pt-title">Title</Label>
            <Input
              id="pt-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Test task title"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="pt-priority">Priority</Label>
            <Select value={priority} onValueChange={(val: string) => setPriority(val as Priority)}>
              <SelectTrigger id="pt-priority">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PRIORITIES.map(p => (
                  <SelectItem key={p} value={p}>
                    {p.charAt(0).toUpperCase() + p.slice(1)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="pt-notes">Notes (Optional)</Label>
            <Textarea
              id="pt-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Additional context..."
              rows={3}
            />
          </div>

          <Button onClick={handlePreview} disabled={isPending || !title.trim()}>
            {isPending ? 'Generating Preview...' : 'Preview Prompt'}
          </Button>
        </div>

        {/* Preview Results */}
        {promptData && (
          <div className="space-y-4 pt-4 border-t border-white/10">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-medium">Assembled Prompt</h4>
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopy}
                disabled={!promptText}
              >
                {copied ? (
                  <>
                    <CheckCircle className="h-4 w-4 mr-2" />
                    Copied!
                  </>
                ) : (
                  <>
                    <Copy className="h-4 w-4 mr-2" />
                    Copy
                  </>
                )}
              </Button>
            </div>

            {/* Metadata */}
            <div className="grid grid-cols-3 gap-4 text-xs">
              <div className="bg-slate-900 border border-white/10 rounded-md px-3 py-2">
                <div className="text-slate-400">Character Count</div>
                <div className="text-lg font-semibold text-slate-100">
                  {characterCount.toLocaleString()}
                </div>
              </div>
              <div className="bg-slate-900 border border-white/10 rounded-md px-3 py-2">
                <div className="text-slate-400">Word Count</div>
                <div className="text-lg font-semibold text-slate-100">
                  {wordCount.toLocaleString()}
                </div>
              </div>
              <div className="bg-slate-900 border border-white/10 rounded-md px-3 py-2">
                <div className="text-slate-400">Line Count</div>
                <div className="text-lg font-semibold text-slate-100">
                  {lineCount.toLocaleString()}
                </div>
              </div>
            </div>

            {/* Prompt Preview */}
            <div className="space-y-2">
              <h5 className="text-xs font-medium text-slate-400">Full Prompt</h5>
              <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
                <div className="border border-white/10 rounded-md p-4 max-h-[420px] overflow-y-auto bg-slate-900 font-mono text-xs whitespace-pre-wrap">
                  {promptText || 'No prompt generated'}
                </div>
                <div className="border border-white/10 rounded-md p-4 max-h-[420px] overflow-y-auto bg-card">
                  <div className="text-xs uppercase text-slate-400 mb-2">Rendered preview</div>
                  {promptText ? (
                    <div
                      className="text-sm leading-relaxed space-y-3 [&_code]:bg-black/40 [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded [&_pre]:text-xs [&_pre]:leading-relaxed [&_ul]:list-disc [&_ul]:pl-5"
                      dangerouslySetInnerHTML={{ __html: promptPreviewHtml }}
                    />
                  ) : (
                    <div className="text-slate-500 text-sm">No prompt generated</div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </TabsContent>

      <TabsContent value="library">
        <PromptLibraryPanel />
      </TabsContent>
    </Tabs>
  );
}
