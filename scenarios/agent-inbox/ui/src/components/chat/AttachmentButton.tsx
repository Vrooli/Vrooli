/**
 * AttachmentButton - ChatGPT-style "+" button with dropdown menu.
 *
 * Opens a dropdown with options for:
 * - Image upload
 * - PDF upload
 * - Web search toggle
 */
import { useState, useRef, useCallback, useEffect } from "react";
import { Plus, Image, FileText, Globe, Check } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";

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
}: AttachmentButtonProps) {
  const showWebSearch = !!onWebSearchToggle;
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const pdfInputRef = useRef<HTMLInputElement>(null);

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
  }, []);

  const handlePDFClick = useCallback(() => {
    pdfInputRef.current?.click();
    setIsOpen(false);
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
  }, [webSearchEnabled, onWebSearchToggle]);

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
          </div>
        </div>
      )}
    </div>
  );
}
