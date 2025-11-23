/**
 * PromptTesterTab Component
 * Test and preview assembled prompts for tasks
 */

import { useState } from 'react';
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
import { usePromptPreview } from '@/hooks/usePromptTester';
import type { TaskType, OperationType, Priority } from '@/types/api';

const PRIORITIES: Priority[] = ['critical', 'high', 'medium', 'low'];

export function PromptTesterTab() {
  const [type, setType] = useState<TaskType>('resource');
  const [operation, setOperation] = useState<OperationType>('generator');
  const [title, setTitle] = useState('Test Task');
  const [priority, setPriority] = useState<Priority>('medium');
  const [notes, setNotes] = useState('');
  const [copied, setCopied] = useState(false);

  const { mutate: previewPrompt, data: promptData, isPending } = usePromptPreview();

  const handlePreview = () => {
    previewPrompt({
      type,
      operation,
      title,
      priority,
      notes: notes || undefined,
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
    <div className="space-y-6">
      {/* Tester Configuration */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <FileText className="h-4 w-4" />
          <span>Configure a test task to preview its assembled prompt</span>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="pt-type">Type</Label>
            <Select value={type} onValueChange={(val: string) => setType(val as TaskType)}>
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
            <Select value={operation} onValueChange={(val: string) => setOperation(val as OperationType)}>
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
              disabled={!promptData.prompt}
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
              <div className="text-slate-400">Token Count</div>
              <div className="text-lg font-semibold text-slate-100">
                {promptData.token_count?.toLocaleString() || 'N/A'}
              </div>
            </div>
            <div className="bg-slate-900 border border-white/10 rounded-md px-3 py-2">
              <div className="text-slate-400">Sections</div>
              <div className="text-lg font-semibold text-slate-100">
                {promptData.sections ? Object.keys(promptData.sections).length : 0}
              </div>
            </div>
            <div className="bg-slate-900 border border-white/10 rounded-md px-3 py-2">
              <div className="text-slate-400">Character Count</div>
              <div className="text-lg font-semibold text-slate-100">
                {promptData.prompt?.length || 0}
              </div>
            </div>
          </div>

          {/* Section Breakdown */}
          {promptData.sections && Object.keys(promptData.sections).length > 0 && (
            <div className="space-y-2">
              <h5 className="text-xs font-medium text-slate-400">Section Breakdown</h5>
              <div className="space-y-1">
                {Object.entries(promptData.sections).map(([name, content]) => (
                  <div
                    key={name}
                    className="flex items-center justify-between text-xs bg-slate-900 border border-white/10 rounded px-3 py-2"
                  >
                    <span className="text-slate-300">{name}</span>
                    <span className="text-slate-500">
                      {typeof content === 'string' ? content.length : 0} chars
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Prompt Preview */}
          <div className="space-y-2">
            <h5 className="text-xs font-medium text-slate-400">Full Prompt</h5>
            <div className="border border-white/10 rounded-md p-4 max-h-96 overflow-y-auto bg-slate-900 font-mono text-xs whitespace-pre-wrap">
              {promptData.prompt || 'No prompt generated'}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
