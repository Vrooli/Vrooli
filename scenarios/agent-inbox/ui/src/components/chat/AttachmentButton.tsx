/**
 * AttachmentButton - ChatGPT-style "+" button with dropdown menu.
 *
 * Opens a dropdown with options for image/PDF upload, web search,
 * templates, and skills. Sub-menu components extracted to AttachmentMenuItems.tsx.
 */
import { useState, useRef, useCallback, useEffect } from "react";
import { Plus } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import type { Template } from "@/lib/types/templates";
import {
  UploadMenuItems,
  WebSearchMenuItem,
  TemplateSkillMenuItems,
} from "./AttachmentMenuItems";

interface AttachmentButtonProps {
  onImageSelect: (file: File) => void;
  onPDFSelect: (file: File) => void;
  webSearchEnabled?: boolean;
  onWebSearchToggle?: (enabled: boolean) => void;
  disabled?: boolean;
  modelSupportsImages: boolean;
  modelSupportsPDFs: boolean;
  modelSupportsWebSearch?: boolean;
  onOpenTemplateSelector?: () => void;
  onOpenSkillSelector?: () => void;
  activeTemplate?: Template | null;
  selectedSkillCount?: number;
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
  onOpenTemplateSelector,
  onOpenSkillSelector,
  activeTemplate,
  selectedSkillCount = 0,
}: AttachmentButtonProps) {
  const showWebSearch = !!onWebSearchToggle;
  const showTemplates = !!onOpenTemplateSelector;
  const showSkills = !!onOpenSkillSelector;
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const pdfInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node) && buttonRef.current && !buttonRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [isOpen]);

  const closeMenu = useCallback(() => { setIsOpen(false); }, []);

  const handleImageClick = useCallback(() => { imageInputRef.current?.click(); closeMenu(); }, [closeMenu]);
  const handlePDFClick = useCallback(() => { pdfInputRef.current?.click(); closeMenu(); }, [closeMenu]);

  const handleImageChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) { onImageSelect(file); e.target.value = ""; }
  }, [onImageSelect]);

  const handlePDFChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) { onPDFSelect(file); e.target.value = ""; }
  }, [onPDFSelect]);

  const handleWebSearchClick = useCallback(() => { onWebSearchToggle?.(!webSearchEnabled); closeMenu(); }, [webSearchEnabled, onWebSearchToggle, closeMenu]);
  const handleTemplateClick = useCallback(() => { onOpenTemplateSelector?.(); closeMenu(); }, [onOpenTemplateSelector, closeMenu]);
  const handleSkillClick = useCallback(() => { onOpenSkillSelector?.(); closeMenu(); }, [onOpenSkillSelector, closeMenu]);

  return (
    <div className="relative">
      <input ref={imageInputRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleImageChange} className="hidden" />
      <input ref={pdfInputRef} type="file" accept="application/pdf" onChange={handlePDFChange} className="hidden" />

      <Tooltip content="Add attachment or enable features">
        <Button ref={buttonRef} variant="ghost" size="icon" onClick={() => setIsOpen(!isOpen)} disabled={disabled} className="h-9 w-9 text-slate-400 hover:text-white hover:bg-white/10" data-testid="attachment-button" aria-label="Add attachment or tools">
          <Plus className="h-5 w-5" />
        </Button>
      </Tooltip>

      {isOpen && (
        <div ref={dropdownRef} className="absolute bottom-full left-0 mb-2 w-56 rounded-lg border border-white/10 bg-slate-900 shadow-xl z-50" data-testid="attachment-dropdown">
          <div className="p-1">
            <UploadMenuItems modelSupportsImages={modelSupportsImages} modelSupportsPDFs={modelSupportsPDFs} onImageClick={handleImageClick} onPDFClick={handlePDFClick} />

            {showWebSearch && <WebSearchMenuItem webSearchEnabled={webSearchEnabled} modelSupportsWebSearch={modelSupportsWebSearch} onClick={handleWebSearchClick} />}

            <TemplateSkillMenuItems
              showTemplates={showTemplates}
              showSkills={showSkills}
              activeTemplate={activeTemplate}
              selectedSkillCount={selectedSkillCount}
              onTemplateClick={handleTemplateClick}
              onSkillClick={handleSkillClick}
            />
          </div>
        </div>
      )}
    </div>
  );
}
