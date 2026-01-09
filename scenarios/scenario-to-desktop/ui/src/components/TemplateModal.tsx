import { createPortal } from "react-dom";
import { Info, X } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Button } from "./ui/button";
import { TemplateGrid } from "./TemplateGrid";

interface TemplateModalProps {
  open: boolean;
  selectedTemplate: string;
  onSelect: (template: string) => void;
  onClose: () => void;
}

export function TemplateModal({ open, selectedTemplate, onSelect, onClose }: TemplateModalProps) {
  if (!open) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <Card className="w-full max-w-5xl border-slate-800 bg-slate-950/90 shadow-xl">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle className="text-lg text-slate-100">Choose a template</CardTitle>
            <p className="text-sm text-slate-400">
              Pick the wrapper that matches today&apos;s needs. You can change this anytime.
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={onClose} className="h-8 w-8 p-0">
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-start gap-3 rounded-lg border border-slate-800/80 bg-slate-950/60 p-3 text-sm text-slate-200">
            <Info className="mt-0.5 h-5 w-5 text-blue-300" />
            <div className="space-y-1">
              <p className="font-semibold text-slate-100">All templates share the same Electron base.</p>
              <p className="text-slate-300">
                Switching templates only changes the wrapper. Your scenario logic stays intact.
              </p>
            </div>
          </div>
          <div className="max-h-[70vh] overflow-y-auto pr-1">
            <TemplateGrid
              selectedTemplate={selectedTemplate}
              onSelect={(template) => {
                onSelect(template);
                onClose();
              }}
            />
          </div>
        </CardContent>
      </Card>
    </div>,
    document.body
  );
}
