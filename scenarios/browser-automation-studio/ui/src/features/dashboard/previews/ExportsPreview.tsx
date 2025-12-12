import React, { useCallback, useEffect, useRef, useState } from 'react';
import { FileJson2, Film, Image, Share2, CheckCircle2, Sparkles, Palette } from 'lucide-react';
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
    accentClass: 'text-blue-200 bg-blue-500/15 border-blue-500/30',
  },
  {
    label: 'GIF highlight',
    icon: <Image size={14} />,
    accentClass: 'text-amber-200 bg-amber-500/15 border-amber-500/30',
  },
];

const DESKTOP_BACKGROUNDS = [
  'bg-gradient-to-br from-slate-950 via-slate-900 to-black',
  'bg-gradient-to-br from-sky-900/70 via-teal-900/60 to-indigo-900/70',
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
  const [activeFrameIndex, setActiveFrameIndex] = useState(0);
  const [brandingVisible, setBrandingVisible] = useState(false);
  const [emailValue, setEmailValue] = useState('');
  const [companyValue, setCompanyValue] = useState('');
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
    setActiveFrameIndex(0);
    setBrandingVisible(false);
    setEmailValue('');
    setCompanyValue('');
  }, []);

  useEffect(() => {
    reset();
    if (!isActive) return;

    const schedule = (fn: () => void, delay: number) => {
      const timeoutId = setTimeout(fn, delay);
      timeoutsRef.current.push(timeoutId);
    };

    const typeIntoField = (text: string, setter: React.Dispatch<React.SetStateAction<string>>, delayOffset = 0) => {
      text.split('').forEach((_, index) => {
        schedule(() => setter(text.slice(0, index + 1)), delayOffset + index * 55);
      });
    };

    // Apply styling steps: theme swap -> autofill -> branding drop
    schedule(() => {
      setAppliedStyleIndex(0);
      setActiveFrameIndex(1);
    }, 1000);
    schedule(() => {
      setAppliedStyleIndex(1);
      typeIntoField('mila@orbit.dev', setEmailValue, 120);
      typeIntoField('Orbit Labs', setCompanyValue, 300);
    }, 2800);
    schedule(() => {
      setAppliedStyleIndex(2);
      setBrandingVisible(true);
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
        return format;
      }));
    }, 3800);

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
        <div className="space-y-2.5 bg-flow-surface/70 rounded-2xl border border-flow-border/50 p-3 lg:max-w-[420px] w-full min-w-0 shadow-lg shadow-black/20">
          <div className="flex items-center gap-2 text-xs font-medium text-flow-text-secondary">
            <Palette size={14} className="text-amber-300" />
            <span>Marketing-ready styling</span>
          </div>
          {[
            { title: 'Theme swap', description: 'Gradient + frame swap live', icon: <Sparkles size={14} /> },
            { title: 'Autofill', description: 'Email + company typed live', icon: <FileJson2 size={14} /> },
            { title: 'Branding lockup', description: 'Corner badge drops in', icon: <Image size={14} /> },
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
          <div className="relative rounded-2xl border border-flow-border/50 bg-flow-surface/80 overflow-hidden shadow-xl shadow-black/30 min-w-0">
            <div className="absolute inset-0 pointer-events-none">
              {DESKTOP_BACKGROUNDS.map((bg, index) => {
                const isActive = index === activeFrameIndex;
                return (
                  <div
                    key={bg}
                    className={`absolute inset-0 transition-all duration-700 ${bg} ${
                      isActive ? 'opacity-100 scale-100' : 'opacity-0 scale-98'
                    }`}
                  />
                );
              })}
            </div>
            <div className="relative m-3 rounded-xl overflow-hidden border border-white/10 bg-black/40 shadow-lg shadow-black/30">
              <div className="flex items-center px-3 py-2 text-[11px] text-flow-text-muted bg-black/60 justify-between">
                <div className="flex items-center gap-1.5">
                  <span className="w-2.5 h-2.5 rounded-full bg-red-500" />
                  <span className="w-2.5 h-2.5 rounded-full bg-amber-400" />
                  <span className="w-2.5 h-2.5 rounded-full bg-green-400" />
                </div>
              </div>
              <div className="relative h-36 bg-gradient-to-br from-slate-950 via-slate-900 to-slate-800 flex items-center justify-center px-4 overflow-hidden">
                {brandingVisible && (
                  <div className="absolute bottom-2 right-2 flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-black/35 border border-white/15 backdrop-blur-sm text-[10px] font-semibold text-white/80 shadow-lg shadow-black/30">
                    <div className="h-5 w-5 rounded-full bg-gradient-to-br from-amber-300/50 via-white/30 to-purple-400/50 border border-white/30" />
                    <span className="tracking-tight">Orbit Studio</span>
                  </div>
                )}
                <div className="rounded-xl bg-black/60 border border-white/10 px-3 py-2 text-left w-full max-w-xs shadow-lg shadow-black/40 space-y-2">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2 text-[11px] text-flow-text-muted">
                      <span className="w-12 text-flow-text-secondary">Email</span>
                      <div className="flex-1 h-6 rounded bg-slate-800/60 border border-white/10 px-2 flex items-center overflow-hidden">
                        <span className={`text-[11px] font-medium text-amber-50/90 tracking-tight transition-opacity duration-200 ${emailValue ? 'opacity-100' : 'opacity-50'}`}>
                          {emailValue || 'typing...'}
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 text-[11px] text-flow-text-muted">
                      <span className="w-12 text-flow-text-secondary">Company</span>
                      <div className="flex-1 h-6 rounded bg-slate-800/60 border border-white/10 px-2 flex items-center overflow-hidden">
                        <span className={`text-[11px] font-medium text-amber-50/90 tracking-tight transition-opacity duration-200 ${companyValue ? 'opacity-100' : 'opacity-50'}`}>
                          {companyValue || 'waiting...'}
                        </span>
                      </div>
                    </div>
                  </div>
                  <div className="text-[11px] text-flow-text-muted">
                    Auto-captioning the replay...
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="mt-3 rounded-2xl border border-flow-border/50 bg-flow-surface/80 p-3 shadow-lg shadow-black/20">
        <div className="text-xs text-flow-text-muted flex items-center gap-2 mb-2">
          <Sparkles size={12} className="text-amber-300" />
          <span>Render to share-ready formats</span>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
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
        </div>
        <div className="text-xs text-flow-text-secondary flex items-center gap-2 mt-2">
          <CheckCircle2 size={12} className="text-emerald-300" />
          <span>Logs, screenshots, and timeline stay embedded.</span>
        </div>
      </div>
    </PreviewContainer>
  );
};

export default ExportsPreview;
