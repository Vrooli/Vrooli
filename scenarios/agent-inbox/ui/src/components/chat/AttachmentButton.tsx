/**
 * AttachmentButton - ChatGPT-style "+" button with dropdown menu.
 *
 * Opens a dropdown with options for:
 * - Image upload
 * - PDF upload
 * - Web search toggle
 * - Force tool selection (cascading menu)
 */
import { useState, useRef, useCallback, useEffect, useLayoutEffect } from "react";
import { Plus, Image, FileText, Globe, Check, Wrench, ChevronRight, ChevronLeft } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import type { EffectiveTool } from "../../lib/api";

/** Forced tool selection state */
export interface ForcedTool {
  scenario: string;
  toolName: string;
}

interface AttachmentButtonProps {
  onImageSelect: (file: File) => void;
  onPDFSelect: (file: File) => void;
  /** Whether web search is enabled. Only used when showWebSearch is true. */
  webSearchEnabled?: boolean;
  /** Callback when web search is toggled. If not provided, web search option is hidden. */
  onWebSearchToggle?: (enabled: boolean) => void;
  disabled?: boolean;
  modelSupportsImages: boolean;
  modelSupportsPDFs: boolean;
  /** Whether the current model supports web search (requires tool calling). Default: true */
  modelSupportsWebSearch?: boolean;
  /** Enabled tools grouped by scenario for force tool selection */
  enabledToolsByScenario?: Map<string, EffectiveTool[]>;
  /** Currently forced tool, if any */
  forcedTool?: ForcedTool | null;
  /** Callback when a tool is force-selected */
  onForceTool?: (scenario: string, toolName: string) => void;
  /** Whether the current model supports tool calling. Default: true */
  modelSupportsTools?: boolean;
}

export function AttachmentButton({
  onImageSelect,
  onPDFSelect,
  webSearchEnabled = false,
  onWebSearchToggle,
  disabled = false,
  modelSupportsImages,
  modelSupportsPDFs,
  modelSupportsWebSearch = true,
  enabledToolsByScenario,
  forcedTool,
  onForceTool,
  modelSupportsTools = true,
}: AttachmentButtonProps) {
  const showWebSearch = !!onWebSearchToggle;
  const showForceTools = !!onForceTool && modelSupportsTools;
  const [isOpen, setIsOpen] = useState(false);
  const [expandedScenario, setExpandedScenario] = useState<string | null>(null);
  const [flyoutDirection, setFlyoutDirection] = useState<"right" | "left">("right");
  const [flyoutVertical, setFlyoutVertical] = useState<"top" | "bottom">("top");
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const pdfInputRef = useRef<HTMLInputElement>(null);
  const scenarioButtonRefs = useRef<Map<string, HTMLButtonElement>>(new Map());

  // Get scenarios that have enabled tools
  const scenariosWithTools = enabledToolsByScenario
    ? Array.from(enabledToolsByScenario.entries()).filter(([, tools]) => tools.length > 0)
    : [];
  const hasEnabledTools = scenariosWithTools.length > 0;

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
        setExpandedScenario(null);
      }
    }

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [isOpen]);

  const handleImageClick = useCallback(() => {
    imageInputRef.current?.click();
    setIsOpen(false);
    setExpandedScenario(null);
  }, []);

  const handlePDFClick = useCallback(() => {
    pdfInputRef.current?.click();
    setIsOpen(false);
    setExpandedScenario(null);
  }, []);

  const handleImageChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) {
        onImageSelect(file);
        // Reset input so same file can be selected again
        e.target.value = "";
      }
    },
    [onImageSelect]
  );

  const handlePDFChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) {
        onPDFSelect(file);
        e.target.value = "";
      }
    },
    [onPDFSelect]
  );

  const handleWebSearchClick = useCallback(() => {
    onWebSearchToggle?.(!webSearchEnabled);
    setIsOpen(false);
    setExpandedScenario(null);
  }, [webSearchEnabled, onWebSearchToggle]);

  const handleScenarioClick = useCallback((scenario: string) => {
    setExpandedScenario((prev) => (prev === scenario ? null : scenario));
  }, []);

  // Calculate flyout direction based on available viewport space
  useLayoutEffect(() => {
    if (!expandedScenario) return;

    const scenarioButton = scenarioButtonRefs.current.get(expandedScenario);
    if (!scenarioButton) return;

    const rect = scenarioButton.getBoundingClientRect();
    const flyoutWidth = 256; // w-64 = 16rem = 256px
    const flyoutMaxHeight = 320; // max-h-80 = 20rem = 320px
    const margin = 8;

    // Horizontal positioning
    const spaceOnRight = window.innerWidth - rect.right;
    const spaceOnLeft = rect.left;

    if (spaceOnRight >= flyoutWidth + margin) {
      setFlyoutDirection("right");
    } else if (spaceOnLeft >= flyoutWidth + margin) {
      setFlyoutDirection("left");
    } else {
      // Default to right if neither side has enough space
      setFlyoutDirection("right");
    }

    // Vertical positioning - align to top or bottom of trigger
    const spaceBelow = window.innerHeight - rect.top;
    const spaceAbove = rect.bottom;

    if (spaceBelow >= flyoutMaxHeight) {
      setFlyoutVertical("top"); // Align flyout top with trigger top
    } else if (spaceAbove >= flyoutMaxHeight) {
      setFlyoutVertical("bottom"); // Align flyout bottom with trigger bottom
    } else {
      // Default to top alignment
      setFlyoutVertical("top");
    }
  }, [expandedScenario]);

  const handleToolSelect = useCallback(
    (scenario: string, toolName: string) => {
      onForceTool?.(scenario, toolName);
      setIsOpen(false);
      setExpandedScenario(null);
    },
    [onForceTool]
  );

  return (
    <div className="relative">
      {/* Hidden file inputs */}
      <input
        ref={imageInputRef}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp"
        onChange={handleImageChange}
        className="hidden"
      />
      <input
        ref={pdfInputRef}
        type="file"
        accept="application/pdf"
        onChange={handlePDFChange}
        className="hidden"
      />

      {/* Main button */}
      <Tooltip content="Add attachment or enable features">
        <Button
          ref={buttonRef}
          variant="ghost"
          size="icon"
          onClick={() => setIsOpen(!isOpen)}
          disabled={disabled}
          className="h-9 w-9 text-slate-400 hover:text-white hover:bg-white/10"
          data-testid="attachment-button"
        >
          <Plus className="h-5 w-5" />
        </Button>
      </Tooltip>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          ref={dropdownRef}
          className="absolute bottom-full left-0 mb-2 w-56 rounded-lg border border-white/10 bg-slate-900 shadow-xl z-50"
          data-testid="attachment-dropdown"
        >
          <div className="p-1">
            {/* Image upload option */}
            <button
              onClick={handleImageClick}
              disabled={!modelSupportsImages}
              className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed text-left"
              data-testid="attachment-image-option"
            >
              <Image className="h-4 w-4 text-blue-400" />
              <div className="flex-1">
                <div className="text-white">Upload image</div>
                {!modelSupportsImages && (
                  <div className="text-xs text-slate-500">Model doesn't support images</div>
                )}
              </div>
            </button>

            {/* PDF upload option */}
            <button
              onClick={handlePDFClick}
              disabled={!modelSupportsPDFs}
              className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed text-left"
              data-testid="attachment-pdf-option"
            >
              <FileText className="h-4 w-4 text-orange-400" />
              <div className="flex-1">
                <div className="text-white">Upload PDF</div>
                {!modelSupportsPDFs && (
                  <div className="text-xs text-slate-500">Model doesn't support PDFs</div>
                )}
              </div>
            </button>

            {/* Web search toggle (only shown when enabled) */}
            {showWebSearch && (
              <>
                {/* Divider */}
                <div className="my-1 border-t border-white/10" />

                <button
                  onClick={handleWebSearchClick}
                  disabled={!modelSupportsWebSearch}
                  className="w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed text-left"
                  data-testid="attachment-search-option"
                >
                  <Globe className="h-4 w-4 text-green-400" />
                  <div className="flex-1">
                    <div className="text-white">Web search</div>
                    <div className={`text-xs ${!modelSupportsWebSearch ? "text-amber-400" : "text-slate-500"}`}>
                      {!modelSupportsWebSearch
                        ? "Model doesn't support tool use"
                        : webSearchEnabled
                          ? "Enabled for this message"
                          : "Search the web for answers"}
                    </div>
                  </div>
                  {webSearchEnabled && modelSupportsWebSearch && (
                    <Check className="h-4 w-4 text-green-400" />
                  )}
                </button>
              </>
            )}

            {/* Force tool selection (cascading flyout menu) */}
            {showForceTools && hasEnabledTools && (
              <>
                {/* Divider (only if web search isn't shown, otherwise already have one) */}
                {!showWebSearch && <div className="my-1 border-t border-white/10" />}

                <div data-testid="force-tool-section">
                  {/* Force tool header */}
                  <div className="px-3 py-1.5 text-xs font-medium text-slate-500 uppercase tracking-wide">
                    Force Tool
                  </div>

                  {/* Scenario list with flyout submenus */}
                  {scenariosWithTools.map(([scenario, tools]) => (
                    <div key={scenario} className="relative">
                      {/* Scenario header (clickable to show flyout) */}
                      <button
                        ref={(el) => {
                          if (el) scenarioButtonRefs.current.set(scenario, el);
                          else scenarioButtonRefs.current.delete(scenario);
                        }}
                        onClick={() => handleScenarioClick(scenario)}
                        className={`w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md hover:bg-white/10 text-left ${
                          expandedScenario === scenario ? "bg-white/5" : ""
                        }`}
                        data-testid={`scenario-${scenario}`}
                      >
                        {flyoutDirection === "left" && expandedScenario === scenario && (
                          <ChevronLeft className="h-4 w-4 text-slate-400" />
                        )}
                        <Wrench className="h-4 w-4 text-violet-400" />
                        <div className="flex-1">
                          <div className="text-white">{scenario}</div>
                          <div className="text-xs text-slate-500">
                            {tools.length} tool{tools.length !== 1 ? "s" : ""} available
                          </div>
                        </div>
                        {(flyoutDirection === "right" || expandedScenario !== scenario) && (
                          <ChevronRight className="h-4 w-4 text-slate-400" />
                        )}
                      </button>

                      {/* Flyout submenu (positioned based on available space) */}
                      {expandedScenario === scenario && (
                        <div
                          className={`absolute w-64 max-h-80 overflow-y-auto rounded-lg border border-white/10 bg-slate-900 shadow-xl z-50 ${
                            flyoutDirection === "right" ? "left-full ml-1" : "right-full mr-1"
                          } ${flyoutVertical === "top" ? "top-0" : "bottom-0"}`}
                        >
                          <div className="p-1">
                            {tools.map((tool) => (
                              <button
                                key={tool.tool.name}
                                onClick={() => handleToolSelect(scenario, tool.tool.name)}
                                className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md hover:bg-white/10 text-left"
                                data-testid={`tool-${tool.tool.name}`}
                              >
                                <div className="flex-1 min-w-0">
                                  <div className="text-white truncate">{tool.tool.name}</div>
                                  {tool.tool.description && (
                                    <div className="text-xs text-slate-500 line-clamp-2">
                                      {tool.tool.description}
                                    </div>
                                  )}
                                </div>
                                {/* Show check if this tool is currently forced */}
                                {forcedTool?.scenario === scenario &&
                                  forcedTool?.toolName === tool.tool.name && (
                                    <Check className="h-4 w-4 text-violet-400 shrink-0" />
                                  )}
                              </button>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
