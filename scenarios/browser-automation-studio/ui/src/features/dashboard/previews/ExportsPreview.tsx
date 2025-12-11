import React, { useCallback, useEffect, useRef, useState } from 'react';
import { FileJson2, Film, Image, Share2, CheckCircle2, Sparkles, Palette, Wand2 } from 'lucide-react';
import PreviewContainer from './PreviewContainer';

type TimeoutRef = ReturnType<typeof setTimeout> | null;
type ExportStatus = 'queued' | 'rendering' | 'ready';

interface ExportFormatState {
  label: string;
  icon: React.ReactNode;
  accentClass: string;
  status: ExportStatus;
  progress: number;
}

const EXPORT_FORMATS: Omit<ExportFormatState, 'status' | 'progress'>[] = [
  {
    label: 'MP4 replay',
    icon: <Film size={14} />,
    accentClass: 'text-purple-200 bg-purple-500/20 border-purple-500/40',
  },
  {
    label: 'GIF highlight',
    icon: <Image size={14} />,
    accentClass: 'text-amber-200 bg-amber-500/20 border-amber-500/40',
  },
  {
    label: 'JSON package',
    icon: <FileJson2 size={14} />,
    accentClass: 'text-emerald-200 bg-emerald-500/20 border-emerald-500/40',
  },
];

export const ExportsPreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
  const [formatStates, setFormatStates] = useState<ExportFormatState[]>(
    EXPORT_FORMATS.map((format) => ({
      ...format,
      status: 'queued',
      progress: 0,
    }))
  );
  const [appliedStyleIndex, setAppliedStyleIndex] = useState(-1);
  const [selectedTheme, setSelectedTheme] = useState('Aurora');
  const [ctaEnabled, setCtaEnabled] = useState(false);
  const [caption, setCaption] = useState('');
  const timeoutsRef = useRef<TimeoutRef[]>([]);

  const reset = useCallback(() => {
    timeoutsRef.current.forEach((timeoutId) => {
      if (timeoutId) clearTimeout(timeoutId);
    });
    timeoutsRef.current = [];
    setFormatStates(EXPORT_FORMATS.map((format) => ({
      ...format,
      status: 'queued',
      progress: 0,
    })));
    setAppliedStyleIndex(-1);
    setSelectedTheme('Aurora');
    setCtaEnabled(false);
    setCaption('');
  }, []);

  useEffect(() => {
    reset();
    if (!isActive) return;

    const schedule = (fn: () => void, delay: number) => {
      const timeoutId = setTimeout(fn, delay);
      timeoutsRef.current.push(timeoutId);
    };

    // Apply styling steps at a calmer cadence: theme -> cursor trail -> watermark -> caption -> CTA
    schedule(() => setAppliedStyleIndex(0), 500);
    schedule(() => setSelectedTheme('Nebula'), 1200);
    schedule(() => setAppliedStyleIndex(1), 2200);
    schedule(() => setAppliedStyleIndex(2), 3200);
    schedule(() => {
      setCaption('Checkout completes in under a minute. Zero flakes.');
      setAppliedStyleIndex(3);
    }, 4200);
    schedule(() => {
      setCtaEnabled(true);
      setAppliedStyleIndex(4);
    }, 5200);

    // Render formats after styling is set
    setFormatStates(EXPORT_FORMATS.map((format, index) => ({
      ...format,
      status: index === 0 ? 'rendering' : 'queued',
      progress: index === 0 ? 25 : 0,
    })));

    schedule(() => {
      setFormatStates((prev) => prev.map((format, index) => {
        if (index === 0) {
          return { ...format, status: 'ready', progress: 100 };
        }
        if (index === 1) {
          return { ...format, status: 'rendering', progress: 40 };
        }
        return format;
      }));
    }, 2400);

    schedule(() => {
      setFormatStates((prev) => prev.map((format, index) => {
        if (index === 1) {
          return { ...format, status: 'ready', progress: 100 };
        }
        if (index === 2) {
          return { ...format, status: 'rendering', progress: 55 };
        }
        return format;
      }));
    }, 3800);

    schedule(() => {
      setFormatStates((prev) => prev.map((format, index) => index === 2
        ? { ...format, status: 'ready', progress: 100 }
        : format
      ));
    }, 5200);

    return reset;
  }, [isActive, reset]);

  const renderStatusBadge = (status: ExportStatus) => {
    const statusStyles: Record<ExportStatus, { bg: string; text: string; label: string }> = {
      queued: { bg: 'bg-flow-surface', text: 'text-flow-text-muted', label: 'Queued' },
      rendering: { bg: 'bg-amber-500/15', text: 'text-amber-200', label: 'Rendering' },
      ready: { bg: 'bg-emerald-500/15', text: 'text-emerald-200', label: 'Ready' },
    };
    const styles = statusStyles[status];
    return (
      <span className={`text-[11px] px-2 py-0.5 rounded-full border border-white/5 ${styles.bg} ${styles.text}`}>
        {styles.label}
      </span>
    );
  };

  return (
    <PreviewContainer
      headerText="exports/replay-studio"
      footerContent={
        <>
          <Share2 size={14} className="text-amber-300" />
          <span className="text-xs text-flow-text-muted">Style, caption, and export in one pass</span>
        </>
      }
    >
      <div className="grid grid-cols-1 lg:grid-cols-[2fr_3fr] gap-3 items-stretch">
        <div className="space-y-2.5 bg-gradient-to-br from-amber-500/5 via-purple-500/5 to-blue-500/5 rounded-2xl border border-flow-border/50 p-3 lg:max-w-[420px] w-full min-w-0">
          <div className="flex items-center gap-2 text-xs font-medium text-flow-text-secondary">
            <Palette size={14} className="text-amber-300" />
            <span>Marketing-ready styling</span>
          </div>
          {[
            { title: `Theme · ${selectedTheme}`, description: 'Browser chrome, background, gradients', icon: <Sparkles size={14} /> },
            { title: 'Cursor trail · Pulse', description: 'Smooth path & click glow', icon: <Wand2 size={14} /> },
            { title: 'Watermark · Orbit Labs', description: 'Branded corner lockup + CTA', icon: <Image size={14} /> },
            { title: 'Caption', description: caption || 'AI copy loads automatically', icon: <FileJson2 size={14} /> },
            { title: 'CTA', description: ctaEnabled ? '“Start free trial” live' : 'CTA preparing...', icon: <Share2 size={14} /> },
          ].map((item, index) => {
            const isActiveStep = index <= appliedStyleIndex;
            return (
              <div
                key={item.title}
                className={`flex items-center gap-2.5 px-2.5 py-2 rounded-lg border transition-all shadow-sm ${
                  isActiveStep
                    ? 'border-amber-400/60 bg-amber-500/15 text-surface shadow-amber-500/10'
                    : 'border-flow-border/60 bg-flow-surface/60 text-flow-text-secondary'
                }`}
                style={{ animation: 'fade-in-up 0.4s ease-out both', animationDelay: `${index * 180}ms` }}
              >
                <div className="w-8 h-8 rounded-md bg-black/30 border border-flow-border/50 flex items-center justify-center text-amber-200">
                  {item.icon}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-semibold truncate">{item.title}</div>
                  <div className="text-[11px] text-flow-text-muted truncate">{item.description}</div>
                </div>
                {renderStatusBadge(isActiveStep ? 'ready' : 'rendering')}
              </div>
            );
          })}
        </div>

        <div className="relative space-y-3">
          <div className="relative rounded-2xl border border-flow-border/50 bg-gradient-to-br from-slate-900/80 via-indigo-900/60 to-slate-800/80 overflow-hidden shadow-xl shadow-black/30 min-w-0">
            <div className="absolute inset-0 bg-gradient-to-br from-amber-500/10 via-transparent to-blue-500/10 pointer-events-none" />
            <div className="relative rounded-xl overflow-hidden border border-white/5 m-3 bg-black/50">
              <div className="flex items-center justify-between px-3 py-2 text-[11px] text-flow-text-muted bg-black/60">
                <span>execution-player.mov</span>
                <span className="text-amber-200">{selectedTheme} theme</span>
              </div>
              <div className="relative h-36 bg-gradient-to-br from-slate-950 via-slate-900 to-slate-800 flex items-center justify-center px-4">
                <div className="absolute top-3 right-3 px-2 py-1 rounded-full text-[11px] bg-black/70 text-amber-100 border border-white/10 shadow-sm">
                  Orbit Labs
                </div>
                <div className="rounded-xl bg-black/60 border border-white/10 px-3 py-2 text-left w-full max-w-xs shadow-lg shadow-black/40">
                  <div className="text-sm text-white font-semibold mb-1">Checkout smoke test</div>
                  <div className="text-xs text-flow-text-muted line-clamp-2">
                    {caption || 'Auto-captioning the replay...'}
                  </div>
                  {ctaEnabled && (
                    <button className="mt-2 inline-flex items-center justify-center w-full rounded-lg bg-amber-400 text-black text-xs font-semibold py-1.5 hover:bg-amber-300 transition-colors shadow-sm shadow-amber-500/30">
                      Start free trial
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-2xl border border-flow-border/50 bg-gradient-to-br from-indigo-900/60 via-slate-900/60 to-slate-800/70 p-3 shadow-lg shadow-black/20 min-w-0">
            <div className="text-xs text-flow-text-muted flex items-center gap-2 mb-2">
              <Sparkles size={12} className="text-amber-300" />
              <span>Render to share-ready formats</span>
            </div>
            <div className="space-y-2">
              {formatStates.map((format, index) => (
                <div
                  key={format.label}
                  className="flex items-center gap-3 rounded-lg border border-flow-border/60 bg-flow-surface/70 px-3 py-2"
                  style={{ animation: 'fade-in-up 0.35s ease-out both', animationDelay: `${index * 180}ms` }}
                >
                  <div className={`flex items-center justify-center w-9 h-9 rounded-lg border ${format.accentClass}`}>
                    {format.icon}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-surface">{format.label}</span>
                      {renderStatusBadge(format.status)}
                    </div>
                    <div className="mt-1 h-1 bg-flow-surface rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all duration-500 ease-out ${
                          format.status === 'ready'
                            ? 'bg-emerald-400'
                            : 'bg-amber-400'
                        }`}
                        style={{ width: `${format.progress}%` }}
                      />
                    </div>
                  </div>
                </div>
              ))}
              <div className="text-xs text-flow-text-secondary flex items-center gap-2">
                <CheckCircle2 size={12} className="text-emerald-300" />
                <span>Logs, screenshots, and timeline stay embedded.</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </PreviewContainer>
  );
};

export default ExportsPreview;
