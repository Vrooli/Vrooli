import { useState, useMemo } from "react";
import { X } from "lucide-react";
import { buildCaptureScreenshotUrl, presetSuffix, presetKey, SIZE_PRESETS } from "../lib/api";
import type { CapturePreset, CaptureTheme } from "../lib/api";
import { sanitizePagePath } from "./ScenarioReviewPanelShared";

export function PresetConfigPanel({ config, onChange }: { config: CapturePreset[]; onChange: (presets: CapturePreset[]) => void }) {
  const [addName, setAddName] = useState("");
  const [addSize, setAddSize] = useState("Desktop");
  const [addCustomW, setAddCustomW] = useState("");
  const [addCustomH, setAddCustomH] = useState("");
  const [addTheme, setAddTheme] = useState<CaptureTheme>("light");

  const sizeEntries = Object.entries(SIZE_PRESETS);
  const isCustomSize = addSize === "Custom";

  // Auto-populate name from size + theme
  const autoName = useMemo(() => {
    const sizeName = isCustomSize ? `${addCustomW || "?"}x${addCustomH || "?"}` : addSize;
    return `${sizeName} ${addTheme === "light" ? "Light" : "Dark"}`;
  }, [addSize, addTheme, addCustomW, addCustomH, isCustomSize]);

  const effectiveName = addName || autoName;

  const addPreset = () => {
    let w: number, h: number;
    if (isCustomSize) {
      w = parseInt(addCustomW, 10);
      h = parseInt(addCustomH, 10);
      if (!w || !h || w <= 0 || h <= 0 || w > 7680 || h > 4320) return;
    } else {
      const preset = SIZE_PRESETS[addSize];
      if (!preset) return;
      w = preset.width;
      h = preset.height;
    }
    const newPreset: CapturePreset = { name: effectiveName, width: w, height: h, theme: addTheme };
    // Duplicate check by key
    if (config.some(c => presetKey(c) === presetKey(newPreset))) return;
    onChange([...config, newPreset]);
    setAddName("");
    setAddCustomW("");
    setAddCustomH("");
  };

  const removePreset = (p: CapturePreset) => {
    if (config.length <= 1) return;
    onChange(config.filter(c => presetKey(c) !== presetKey(p)));
  };

  const isDuplicate = (() => {
    let w: number, h: number;
    if (isCustomSize) {
      w = parseInt(addCustomW, 10);
      h = parseInt(addCustomH, 10);
      if (!w || !h) return false;
    } else {
      const preset = SIZE_PRESETS[addSize];
      if (!preset) return false;
      w = preset.width;
      h = preset.height;
    }
    return config.some(c => presetKey(c) === `${w}x${h}_${addTheme}`);
  })();

  return (
    <div className="p-3 space-y-3 min-w-[260px]">
      <p className="text-xs font-medium text-slate-400">Capture presets</p>

      {/* Current preset list */}
      {config.map(p => (
        <div key={presetKey(p)} className="flex items-center gap-2 text-xs text-slate-300">
          <span className="flex-1 truncate">{p.name}</span>
          <span className="text-slate-500">{p.width}&times;{p.height}</span>
          <span className={`px-1.5 py-0.5 rounded text-[10px] ${p.theme === "dark" ? "bg-slate-700 text-slate-300" : "bg-slate-800 text-slate-400"}`}>
            {p.theme === "dark" ? "Dark" : "Light"}
          </span>
          <button
            type="button"
            onClick={() => removePreset(p)}
            disabled={config.length <= 1}
            className="text-slate-600 hover:text-red-400 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ))}

      {/* Add form */}
      <div className="pt-2 border-t border-slate-800 space-y-2">
        <p className="text-[10px] font-medium text-slate-500 uppercase tracking-wider">Add preset</p>
        <input
          type="text"
          placeholder={autoName}
          value={addName}
          onChange={e => setAddName(e.target.value)}
          className="w-full px-2 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
        <div className="flex items-center gap-1.5">
          <select
            value={addSize}
            onChange={e => setAddSize(e.target.value)}
            className="flex-1 px-2 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            {sizeEntries.map(([name, s]) => (
              <option key={name} value={name}>{name} ({s.width}&times;{s.height})</option>
            ))}
            <option value="Custom">Custom</option>
          </select>
          <select
            value={addTheme}
            onChange={e => setAddTheme(e.target.value as CaptureTheme)}
            className="px-2 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </div>
        {isCustomSize && (
          <div className="flex items-center gap-1.5">
            <input
              type="number"
              placeholder="W"
              value={addCustomW}
              onChange={e => setAddCustomW(e.target.value)}
              className="w-20 px-1.5 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <span className="text-xs text-slate-600">&times;</span>
            <input
              type="number"
              placeholder="H"
              value={addCustomH}
              onChange={e => setAddCustomH(e.target.value)}
              className="w-20 px-1.5 py-1 rounded bg-slate-800 border border-slate-700 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
        )}
        <button
          type="button"
          onClick={addPreset}
          disabled={isDuplicate || (isCustomSize && (!addCustomW || !addCustomH))}
          className="w-full px-2 py-1.5 rounded bg-blue-600 text-white text-xs hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isDuplicate ? "Already exists" : "Add"}
        </button>
      </div>
    </div>
  );
}

export function ScreenshotImage({
  captureId,
  scenarioSlug,
  pagePath,
  preset,
  onClick,
}: {
  captureId: string;
  scenarioSlug: string;
  pagePath: string;
  preset: CapturePreset;
  onClick: () => void;
}) {
  const filename = sanitizePagePath(pagePath) + presetSuffix(preset) + ".png";
  const url = buildCaptureScreenshotUrl(captureId, scenarioSlug, filename);

  return (
    <div
      className="rounded-lg border border-slate-800 overflow-hidden bg-slate-900 cursor-pointer hover:ring-2 hover:ring-blue-500/50 transition-shadow"
      onClick={onClick}
    >
      <img
        src={url}
        alt={`Screenshot of ${pagePath}`}
        className="max-w-full object-contain"
        loading="lazy"
      />
    </div>
  );
}
