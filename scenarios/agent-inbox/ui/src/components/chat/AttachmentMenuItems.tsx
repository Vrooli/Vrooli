/**
 * AttachmentMenuItems - Dropdown menu items for the AttachmentButton.
 *
 * Extracted from AttachmentButton.tsx. Contains the image/PDF upload,
 * web search toggle, templates, and skills menu items.
 */
import {
  Image,
  FileText,
  Globe,
  Check,
  Sparkles,
  BookOpen,
} from "lucide-react";
import type { Template } from "@/lib/types/templates";

interface UploadMenuItemsProps {
  modelSupportsImages: boolean;
  modelSupportsPDFs: boolean;
  onImageClick: () => void;
  onPDFClick: () => void;
}

export function UploadMenuItems({ modelSupportsImages, modelSupportsPDFs, onImageClick, onPDFClick }: UploadMenuItemsProps) {
  return (
    <>
      <button onClick={onImageClick} disabled={!modelSupportsImages} className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed text-left" data-testid="attachment-image-option">
        <Image className="h-4 w-4 text-blue-400" />
        <div className="flex-1"><div className="text-white">Upload image</div>{!modelSupportsImages && <div className="text-xs text-slate-500">Model doesn't support images</div>}</div>
      </button>
      <button onClick={onPDFClick} disabled={!modelSupportsPDFs} className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed text-left" data-testid="attachment-pdf-option">
        <FileText className="h-4 w-4 text-orange-400" />
        <div className="flex-1"><div className="text-white">Upload PDF</div>{!modelSupportsPDFs && <div className="text-xs text-slate-500">Model doesn't support PDFs</div>}</div>
      </button>
    </>
  );
}

interface WebSearchMenuItemProps {
  webSearchEnabled: boolean;
  modelSupportsWebSearch: boolean;
  onClick: () => void;
}

export function WebSearchMenuItem({ webSearchEnabled, modelSupportsWebSearch, onClick }: WebSearchMenuItemProps) {
  return (
    <>
      <div className="my-1 border-t border-white/10" />
      <button onClick={onClick} disabled={!modelSupportsWebSearch} className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed text-left" data-testid="attachment-search-option">
        <Globe className="h-4 w-4 text-green-400" />
        <div className="flex-1">
          <div className="text-white">Web search</div>
          <div className={`text-xs ${!modelSupportsWebSearch ? "text-amber-400" : "text-slate-500"}`}>
            {!modelSupportsWebSearch ? "Model doesn't support tool use" : webSearchEnabled ? "Enabled for this message" : "Search the web for answers"}
          </div>
        </div>
        {webSearchEnabled && modelSupportsWebSearch && <Check className="h-4 w-4 text-green-400" />}
      </button>
    </>
  );
}

interface TemplateSkillMenuItemsProps {
  showTemplates: boolean;
  showSkills: boolean;
  activeTemplate?: Template | null;
  selectedSkillCount: number;
  onTemplateClick: () => void;
  onSkillClick: () => void;
}

export function TemplateSkillMenuItems({ showTemplates, showSkills, activeTemplate, selectedSkillCount, onTemplateClick, onSkillClick }: TemplateSkillMenuItemsProps) {
  return (
    <>
      {showTemplates && (
        <>
          <div className="my-1 border-t border-white/10" />
          <button onClick={onTemplateClick} className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 text-left" data-testid="attachment-template-option">
            <Sparkles className="h-4 w-4 text-blue-400" />
            <div className="flex-1">
              <div className="text-white">Templates</div>
              <div className="text-xs text-slate-500">{activeTemplate ? `Using: ${activeTemplate.name}` : "Browse message templates"}</div>
            </div>
            {activeTemplate && <Check className="h-4 w-4 text-blue-400" />}
          </button>
        </>
      )}
      {showSkills && (
        <>
          {!showTemplates && <div className="my-1 border-t border-white/10" />}
          <button onClick={onSkillClick} className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 text-left" data-testid="attachment-skill-option">
            <BookOpen className="h-4 w-4 text-amber-400" />
            <div className="flex-1">
              <div className="text-white">Skills</div>
              <div className="text-xs text-slate-500">{selectedSkillCount > 0 ? `${selectedSkillCount} skill${selectedSkillCount !== 1 ? "s" : ""} attached` : "Attach knowledge skills"}</div>
            </div>
            {selectedSkillCount > 0 && <span className="text-xs px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-400">{selectedSkillCount}</span>}
          </button>
        </>
      )}
    </>
  );
}
