import { useEffect, useRef } from "react";
import { X, Monitor, Laptop, Tablet, Smartphone } from "lucide-react";
import type { EmulatorFrame as EmulatorFrameType } from "../types";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";

interface EmulatorFrameProps {
  frame: EmulatorFrameType;
  componentCode: string;
  onRemove: () => void;
  onElementSelect?: (selector: string) => void;
}

const iconMap = {
  Monitor,
  Laptop,
  Tablet,
  Smartphone,
};

export function EmulatorFrame({
  frame,
  componentCode,
  onRemove,
  onElementSelect,
}: EmulatorFrameProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const Icon = iconMap[frame.preset.icon as keyof typeof iconMap] || Monitor;

  useEffect(() => {
    if (!iframeRef.current) return;

    const iframeDoc = iframeRef.current.contentDocument;
    if (!iframeDoc) return;

    // Create a simple preview environment
    const html = `
      <!DOCTYPE html>
      <html>
        <head>
          <meta charset="UTF-8">
          <meta name="viewport" content="width=device-width, initial-scale=1.0">
          <script src="https://cdn.tailwindcss.com"></script>
          <style>
            body {
              margin: 0;
              padding: 16px;
              background: #0f172a;
              color: #f1f5f9;
              font-family: system-ui, -apple-system, sans-serif;
            }
            .preview-container {
              min-height: 100vh;
              display: flex;
              align-items: center;
              justify-content: center;
            }
          </style>
        </head>
        <body>
          <div class="preview-container">
            <div class="preview-content">
              <h3 class="text-lg font-semibold mb-4">Component Preview</h3>
              <p class="text-sm text-slate-400">Preview will render here when component is valid React code.</p>
              <pre class="mt-4 p-4 bg-slate-900 rounded text-xs overflow-auto max-w-full"><code>${escapeHtml(
                componentCode.slice(0, 200)
              )}${componentCode.length > 200 ? "..." : ""}</code></pre>
            </div>
          </div>
        </body>
      </html>
    `;

    iframeDoc.open();
    iframeDoc.write(html);
    iframeDoc.close();

    // Setup element selection if enabled
    if (onElementSelect) {
      iframeDoc.addEventListener("click", (e) => {
        const target = e.target as HTMLElement;
        if (target) {
          e.preventDefault();
          const selector = generateSelector(target);
          onElementSelect(selector);
        }
      });
    }
  }, [componentCode, onElementSelect]);

  return (
    <div className="flex flex-col rounded-lg border border-white/10 bg-slate-900/50 shadow-lg">
      {/* Frame Header */}
      <div className="flex items-center justify-between border-b border-white/10 px-4 py-2">
        <div className="flex items-center space-x-2">
          <Icon className="h-4 w-4 text-slate-400" />
          <span className="text-sm font-medium">{frame.preset.name}</span>
          <Badge variant="outline" className="text-xs">
            {frame.preset.width} × {frame.preset.height}
          </Badge>
        </div>
        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onRemove}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Preview Area */}
      <div className="relative flex-1 overflow-auto bg-slate-950 p-4">
        <div
          className="mx-auto origin-top-left"
          style={{
            width: `${frame.preset.width}px`,
            height: `${frame.preset.height}px`,
            transform: `scale(${Math.min(1, 800 / frame.preset.width)})`,
          }}
        >
          <iframe
            ref={iframeRef}
            title={`Preview: ${frame.preset.name}`}
            className="h-full w-full rounded border border-white/5 bg-white"
            sandbox="allow-scripts allow-same-origin"
          />
        </div>
      </div>
    </div>
  );
}

function escapeHtml(text: string): string {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

function generateSelector(element: HTMLElement): string {
  if (element.id) return `#${element.id}`;
  if (element.className) {
    const classes = element.className.split(" ").filter((c) => c.trim());
    if (classes.length) return `.${classes.join(".")}`;
  }
  return element.tagName.toLowerCase();
}
