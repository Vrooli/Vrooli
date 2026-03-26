import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Sun, Moon } from "lucide-react";
import { fetchThemePreview } from "../lib/api";
import { Button } from "./ui/button";
import { Section } from "./ui/section";

// [REQ:BM-REQ-UI-THEME]

interface ThemePreviewProps {
  brandId: string;
}

export function ThemePreview({ brandId }: ThemePreviewProps) {
  const [mode, setMode] = useState<"light" | "dark">("light");

  const { data: preview, isLoading } = useQuery({
    queryKey: ["theme-preview", brandId, mode],
    queryFn: () => fetchThemePreview(brandId, mode),
  });

  return (
    <Section testId="theme-preview-section">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-medium text-slate-400">Theme Preview</h2>
        <div className="flex items-center gap-1 rounded-lg border border-white/10 p-0.5" data-testid="theme-mode-toggle">
          <Button
            variant={mode === "light" ? "default" : "outline"}
            size="sm"
            onClick={() => setMode("light")}
            data-testid="theme-light-btn"
          >
            <Sun className="h-3 w-3 mr-1" /> Light
          </Button>
          <Button
            variant={mode === "dark" ? "default" : "outline"}
            size="sm"
            onClick={() => setMode("dark")}
            data-testid="theme-dark-btn"
          >
            <Moon className="h-3 w-3 mr-1" /> Dark
          </Button>
        </div>
      </div>

      {isLoading && <div className="text-slate-500 text-sm py-4 text-center">Loading preview...</div>}

      {preview && (
        <div
          className="rounded-lg border border-white/10 overflow-hidden"
          data-testid="theme-preview-card"
        >
          {/* Preview frame using CSS tokens */}
          <div
            className="p-6 transition-colors duration-300"
            style={{
              backgroundColor: preview.tokens.background || (mode === "dark" ? "#1a1a2e" : "#ffffff"),
              color: preview.tokens.text || (mode === "dark" ? "#eaeaea" : "#1a202c"),
            }}
          >
            <div className="space-y-3">
              <h3
                className="text-lg font-bold"
                style={{
                  color: preview.tokens.primary || "#1a365d",
                  fontFamily: preview.tokens["heading-font"] || "inherit",
                }}
              >
                Heading Preview
              </h3>
              <p
                className="text-sm"
                style={{
                  fontFamily: preview.tokens["body-font"] || "inherit",
                }}
              >
                Body text using your brand typography and colors. This shows how content will appear in {mode} mode.
              </p>
              <div className="flex gap-2">
                <span
                  className="rounded px-3 py-1 text-xs font-medium text-white"
                  style={{ backgroundColor: preview.tokens.primary || "#1a365d" }}
                >
                  Primary
                </span>
                <span
                  className="rounded px-3 py-1 text-xs font-medium text-white"
                  style={{ backgroundColor: preview.tokens.secondary || "#2d3748" }}
                >
                  Secondary
                </span>
                <span
                  className="rounded px-3 py-1 text-xs font-medium text-white"
                  style={{ backgroundColor: preview.tokens.accent || "#e53e3e" }}
                >
                  Accent
                </span>
              </div>
              {preview.tokens.surface && (
                <div
                  className="rounded p-3 text-xs"
                  style={{
                    backgroundColor: preview.tokens.surface,
                    color: preview.tokens.text || "inherit",
                  }}
                >
                  Surface card preview
                </div>
              )}
            </div>
          </div>

          {/* Token list */}
          <div className="border-t border-white/10 p-3 bg-slate-900/50">
            <p className="text-xs text-slate-500 mb-2">CSS Tokens ({mode} mode)</p>
            <div className="grid grid-cols-2 gap-1">
              {Object.entries(preview.tokens).map(([key, value]) => (
                <div key={key} className="flex items-center gap-2 text-xs">
                  <code className="text-slate-400">--brand-{key}</code>
                  <span className="text-slate-500">{value}</span>
                  {value.startsWith("#") && (
                    <span
                      className="inline-block h-3 w-3 rounded-sm border border-white/20"
                      style={{ backgroundColor: value }}
                    />
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </Section>
  );
}
