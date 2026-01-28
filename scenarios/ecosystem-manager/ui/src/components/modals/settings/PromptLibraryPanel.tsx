import { useEffect, useMemo, useState } from 'react';
import { FileText, RefreshCw, Save, FolderOpen, Eye, Copy, Cloud, CloudOff, Loader2 } from 'lucide-react';
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
import { useSteerSkills, useSyncSkills } from '@/hooks/useSkills';
import { markdownToHtml } from '@/lib/markdown';
import type { PromptFileInfo, SkillResponse } from '@/types/api';

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

function toTitleCase(value: string) {
  return value
    .split(/[-_]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function describePrompt(file: PromptFileInfo) {
  const base = file.display_name || file.id;
  const clean = base.replace(/\.md$/, '');

  if (file.type === 'template') {
    return `Prompt template: ${toTitleCase(clean)}`;
  }

  return `Prompt file: ${toTitleCase(clean)}`;
}

// Unified item type for the dropdown
type DropdownItem = {
  id: string;
  displayName: string;
  description: string;
  type: 'local' | 'skill';
  // For local files
  fileInfo?: PromptFileInfo;
  // For skills
  skill?: SkillResponse;
};

type GroupedItems = {
  key: string;
  label: string;
  items: DropdownItem[];
  isSkillGroup?: boolean;
};

export function PromptLibraryPanel() {
  const { data: files = [], isLoading: filesLoading } = usePromptFiles();
  const { data: steerSkills = [], isLoading: skillsLoading } = useSteerSkills();
  const syncSkills = useSyncSkills();
  const [selectedId, setSelectedId] = useState<string>();
  const { data: file, isFetching: fileLoading, refetch } = usePromptFile(
    // Only fetch if it's a local file (not a skill)
    selectedId?.startsWith('skill:') ? undefined : selectedId
  );
  const savePrompt = useSavePromptFile();
  const [draft, setDraft] = useState('');

  // Determine prompt-manager availability based on whether we have skills
  const promptManagerAvailable = steerSkills.length > 0;

  // Find the selected skill if a skill is selected
  const selectedSkill = useMemo(() => {
    if (!selectedId?.startsWith('skill:')) return null;
    const skillId = selectedId.replace('skill:', '');
    return steerSkills.find((s) => s.id === skillId);
  }, [selectedId, steerSkills]);

  // Check if the selected item is a skill (read-only)
  const isSkillSelected = selectedId?.startsWith('skill:');

  // Build grouped sections with local files and skills
  const groupedSections: GroupedItems[] = useMemo(() => {
    const groups: Record<string, DropdownItem[]> = {
      skill: [],
      template: [],
      other: [],
    };

    // Add local files (excluding phases - those now come from prompt-manager)
    files.forEach((f) => {
      const key = f.type && groups[f.type] ? f.type : 'other';
      groups[key].push({
        id: f.id,
        displayName: f.display_name || f.id,
        description: describePrompt(f),
        type: 'local',
        fileInfo: f,
      });
    });

    // Add skills from prompt-manager
    steerSkills.forEach((skill) => {
      groups.skill.push({
        id: `skill:${skill.id}`,
        displayName: skill.name,
        description: skill.description || `Steer skill: ${skill.name}`,
        type: 'skill',
        skill,
      });
    });

    const sections: GroupedItems[] = [];

    // Add skill group first if there are skills
    if (groups.skill.length > 0) {
      sections.push({
        key: 'skill',
        label: 'Steer Phases (from prompt-manager)',
        items: groups.skill,
        isSkillGroup: true,
      });
    }

    // Add local file groups
    if (groups.template.length > 0) {
      sections.push({ key: 'template', label: 'Prompt Templates', items: groups.template });
    }
    if (groups.other.length > 0) {
      sections.push({ key: 'other', label: 'Other Prompts', items: groups.other });
    }

    return sections;
  }, [files, steerSkills]);

  // Auto-select first item
  useEffect(() => {
    if (!selectedId && groupedSections.length > 0) {
      const first = groupedSections[0].items[0];
      if (first) {
        setSelectedId(first.id);
      }
    }
  }, [groupedSections, selectedId]);

  // Update draft when file or skill changes
  useEffect(() => {
    if (selectedSkill) {
      setDraft(selectedSkill.content);
    } else if (file) {
      setDraft(file.content);
    } else {
      setDraft('');
    }
  }, [file, selectedSkill]);

  // Find current info for the metadata panel
  const currentLocalFile: PromptFileInfo | undefined = files.find((f) => f.id === selectedId);
  const isDirty = !isSkillSelected && draft !== (file?.content ?? '');
  const renderedPreview = useMemo(() => markdownToHtml(draft), [draft]);

  const handleSave = () => {
    if (!selectedId || isSkillSelected) return;
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

  const modifiedLabel = currentLocalFile?.modified_at
    ? new Date(currentLocalFile.modified_at).toLocaleString()
    : '—';

  const isLoading = filesLoading || skillsLoading;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <FileText className="h-4 w-4" />
          <span>Browse, edit, and preview raw prompt files</span>
        </div>
        <div className="flex items-center gap-2">
          {/* prompt-manager status and sync */}
          <div className="flex items-center gap-1.5 text-xs text-slate-400 mr-2">
            {promptManagerAvailable ? (
              <Cloud className="h-3.5 w-3.5 text-green-400" />
            ) : (
              <CloudOff className="h-3.5 w-3.5 text-slate-500" />
            )}
            <span className={promptManagerAvailable ? 'text-green-400' : ''}>
              {promptManagerAvailable ? 'Connected' : 'Unavailable'}
            </span>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => syncSkills.mutate()}
            disabled={syncSkills.isPending}
          >
            {syncSkills.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-[280px_1fr]">
        <div className="space-y-3">
          <div className="space-y-2">
            <Label>Prompt File</Label>
            <Select
              value={selectedId || ''}
              onValueChange={setSelectedId}
              disabled={isLoading || (files.length === 0 && steerSkills.length === 0)}
            >
              <SelectTrigger>
                <SelectValue placeholder={isLoading ? 'Loading...' : 'Select file'} />
              </SelectTrigger>
              <SelectContent>
                {groupedSections.map((section) => (
                  <div key={section.key} className="px-2 py-1">
                    <div className="flex items-center gap-1.5 text-xs uppercase tracking-wide text-slate-400 px-2 pb-1">
                      {section.isSkillGroup && (
                        <Cloud className="h-3 w-3 text-green-400" />
                      )}
                      {section.label}
                    </div>
                    {section.items.map((item) => (
                      <SelectItem key={item.id} value={item.id} className="py-2">
                        <div className="flex flex-col">
                          <div className="flex items-center gap-1.5">
                            <span className="font-medium">{item.displayName}</span>
                            {item.type === 'skill' && item.skill?.icon && (
                              <span className="text-sm">{item.skill.icon}</span>
                            )}
                          </div>
                          <span className="text-xs text-slate-400">{item.description}</span>
                        </div>
                      </SelectItem>
                    ))}
                  </div>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2 rounded-md border border-white/10 bg-slate-900 p-3 text-sm">
            {isSkillSelected && selectedSkill ? (
              <>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Source</span>
                  <span className="text-green-400 flex items-center gap-1">
                    <Cloud className="h-3 w-3" />
                    prompt-manager
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Name</span>
                  <span className="text-slate-100">{selectedSkill.name}</span>
                </div>
                {selectedSkill.modes && selectedSkill.modes.length > 0 && (
                  <div className="flex items-center justify-between">
                    <span className="text-slate-400">Modes</span>
                    <span className="text-slate-100">{selectedSkill.modes.join(', ')}</span>
                  </div>
                )}
                {selectedSkill.tags && selectedSkill.tags.length > 0 && (
                  <div className="flex items-center justify-between">
                    <span className="text-slate-400">Tags</span>
                    <span className="text-slate-100">{selectedSkill.tags.join(', ')}</span>
                  </div>
                )}
                <div className="mt-2 p-2 bg-amber-500/10 border border-amber-500/20 rounded text-xs text-amber-200">
                  This is a read-only skill from prompt-manager. Edit in the Prompt Manager UI.
                </div>
              </>
            ) : (
              <>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Path</span>
                  <code className="text-xs bg-white/5 px-2 py-1 rounded border border-white/10">
                    {currentLocalFile?.path || '—'}
                  </code>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Type</span>
                  <span className="text-slate-100">{currentLocalFile?.type || '—'}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Size</span>
                  <span className="text-slate-100">{formatBytes(currentLocalFile?.size)}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Last Modified</span>
                  <span className="text-slate-100">{modifiedLabel}</span>
                </div>
              </>
            )}
          </div>

          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigator.clipboard.writeText(isSkillSelected ? (selectedSkill?.name || '') : (currentLocalFile?.path || ''))}
              disabled={!currentLocalFile && !selectedSkill}
            >
              <FolderOpen className="h-4 w-4 mr-2" />
              {isSkillSelected ? 'Copy Name' : 'Copy Path'}
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
              <Label className="text-sm">{isSkillSelected ? 'Content (Read-only)' : 'Editor'}</Label>
              <p className="text-xs text-slate-400">
                {isSkillSelected
                  ? 'Skills from prompt-manager are read-only. Edit them in the Prompt Manager UI.'
                  : 'Changes are saved directly to the prompt file. Keep markdown formatting intact.'}
              </p>
            </div>
            {!isSkillSelected && (
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
            )}
          </div>

          <Textarea
            value={draft}
            onChange={(e) => !isSkillSelected && setDraft(e.target.value)}
            className="font-mono text-sm min-h-[320px] bg-slate-900"
            spellCheck={false}
            readOnly={isSkillSelected}
            placeholder={fileLoading || skillsLoading ? 'Loading prompt...' : 'Select a prompt file to edit'}
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
                  {fileLoading || skillsLoading ? 'Loading preview...' : 'No content to preview'}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
