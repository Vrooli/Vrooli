/**
 * AttachmentMenuItems - Dropdown menu items for the AttachmentButton.
 *
 * Extracted from AttachmentButton.tsx. Contains the image/PDF upload, web search toggle,
 * force tool selection, templates, and skills menu items.
 */
import { useRef, useLayoutEffect, useState } from "react";
import {
  Image,
  FileText,
  Globe,
  Check,
  Wrench,
  ChevronRight,
  ChevronLeft,
  Sparkles,
  BookOpen,
} from "lucide-react";
import type { EffectiveTool } from "../../lib/api";
import type { ForcedTool } from "./AttachmentButton";
import type { Template } from "@/lib/types/templates";

interface ForceToolMenuProps {
  scenariosWithTools: [string, EffectiveTool[]][];
  expandedScenario: string | null;
  onScenarioClick: (scenario: string) => void;
  onToolSelect: (scenario: string, toolName: string) => void;
  forcedTool?: ForcedTool | null;
}

export function ForceToolMenu({
  scenariosWithTools,
  expandedScenario,
  onScenarioClick,
  onToolSelect,
  forcedTool,
}: ForceToolMenuProps) {
  const scenarioButtonRefs = useRef<Map<string, HTMLButtonElement>>(new Map());
  const [flyoutDirection, setFlyoutDirection] = useState<"right" | "left">("right");
  const [flyoutVertical, setFlyoutVertical] = useState<"top" | "bottom">("top");

  useLayoutEffect(() => {
    if (!expandedScenario) return;
    const scenarioButton = scenarioButtonRefs.current.get(expandedScenario);
    if (!scenarioButton) return;
    const rect = scenarioButton.getBoundingClientRect();
    const flyoutWidth = 256;
    const flyoutMaxHeight = 320;
    const margin = 8;

    const spaceOnRight = window.innerWidth - rect.right;
    const spaceOnLeft = rect.left;
    if (spaceOnRight >= flyoutWidth + margin) setFlyoutDirection("right");
    else if (spaceOnLeft >= flyoutWidth + margin) setFlyoutDirection("left");
    else setFlyoutDirection("right");

    const spaceBelow = window.innerHeight - rect.top;
    if (spaceBelow >= flyoutMaxHeight) setFlyoutVertical("top");
    else setFlyoutVertical("bottom");
  }, [expandedScenario]);

  return (
    <div data-testid="force-tool-section">
      <div className="px-3 py-1.5 text-xs font-medium text-slate-500 uppercase tracking-wide">Force Tool</div>
      {scenariosWithTools.map(([scenario, tools]) => (
        <div key={scenario} className="relative">
          <button
            ref={(el) => { if (el) scenarioButtonRefs.current.set(scenario, el); else scenarioButtonRefs.current.delete(scenario); }}
            onClick={() => onScenarioClick(scenario)}
            className={`w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 text-left ${expandedScenario === scenario ? "bg-white/5" : ""}`}
            data-testid={`scenario-${scenario}`}
          >
            {flyoutDirection === "left" && expandedScenario === scenario && <ChevronLeft className="h-4 w-4 text-slate-400" />}
            <Wrench className="h-4 w-4 text-violet-400" />
            <div className="flex-1">
              <div className="text-white">{scenario}</div>
              <div className="text-xs text-slate-500">{tools.length} tool{tools.length !== 1 ? "s" : ""} available</div>
            </div>
            {(flyoutDirection === "right" || expandedScenario !== scenario) && <ChevronRight className="h-4 w-4 text-slate-400" />}
          </button>

          {expandedScenario === scenario && (
            <div className={`absolute w-64 max-h-80 overflow-y-auto rounded-lg border border-white/10 bg-slate-900 shadow-xl z-50 ${flyoutDirection === "right" ? "left-full ml-1" : "right-full mr-1"} ${flyoutVertical === "top" ? "top-0" : "bottom-0"}`}>
              <div className="p-1">
                {tools.map((tool) => (
                  <button key={tool.tool.name} onClick={() => onToolSelect(scenario, tool.tool.name)} className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md hover:bg-white/10 text-left" data-testid={`tool-${tool.tool.name}`}>
                    <div className="flex-1 min-w-0">
                      <div className="text-white truncate">{tool.tool.name}</div>
                      {tool.tool.description && <div className="text-xs text-slate-500 line-clamp-2">{tool.tool.description}</div>}
                    </div>
                    {forcedTool?.scenario === scenario && forcedTool.toolName === tool.tool.name && <Check className="h-4 w-4 text-violet-400 shrink-0" />}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

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
