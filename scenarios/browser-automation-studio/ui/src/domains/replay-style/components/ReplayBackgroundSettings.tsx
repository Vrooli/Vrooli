import clsx from 'clsx';
import { Image as ImageIcon, X } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useAssetStore } from '@stores/assetStore';
import { AssetPicker } from '@/domains/exports/replay/AssetPicker';
import { RangeSlider } from '@shared/ui';
import type {
  ReplayBackgroundImageFit,
  ReplayBackgroundSource,
  ReplayGradientSpec,
  ReplayGradientStop,
} from '../model';
import { getReplayBackgroundThemeId } from '../model';
import { buildGradientCss, parseGradientCss, DEFAULT_REPLAY_GRADIENT_SPEC, normalizeGradientStops } from '../gradient';
import { REPLAY_BACKGROUND_OPTIONS } from '../catalog';

type BackgroundTab = 'presets' | 'solid' | 'gradient' | 'pattern' | 'image';
type BackgroundSection = 'solid' | 'gradient' | 'pattern' | 'image';
type BackgroundSettingsVariant = 'settings' | 'compact';

interface ReplayBackgroundSettingsProps {
  background: ReplayBackgroundSource;
  onBackgroundChange: (value: ReplayBackgroundSource) => void;
  variant?: BackgroundSettingsVariant;
}

const TAB_OPTIONS: Array<{ id: BackgroundTab; label: string }> = [
  { id: 'presets', label: 'Presets' },
  { id: 'solid', label: 'Solid' },
  { id: 'gradient', label: 'Gradient' },
  { id: 'pattern', label: 'Pattern' },
  { id: 'image', label: 'Image' },
];

const SECTION_LABELS: Record<BackgroundSection, string> = {
  solid: 'Solid',
  gradient: 'Gradient',
  pattern: 'Pattern',
  image: 'Image',
};

const getTabFromSection = (section: BackgroundSection): BackgroundTab => section;

const buildSectionOptions = (section: Exclude<BackgroundSection, 'image'>) =>
  REPLAY_BACKGROUND_OPTIONS.filter((option) => option.kind === section);

const getStyles = (variant: BackgroundSettingsVariant) => {
  const isCompact = variant === 'compact';
  return {
    isCompact,
    tabButton: clsx(
      'rounded-full px-3 py-1.5 font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60',
      isCompact ? 'text-[11px]' : 'text-xs',
    ),
    tabButtonActive: 'bg-flow-accent text-white',
    tabButtonIdle: isCompact
      ? 'bg-slate-900/60 text-slate-300 hover:bg-slate-900/80'
      : 'bg-gray-800 text-gray-300 hover:bg-gray-700',
    sectionTitle: clsx(
      'uppercase tracking-[0.24em]',
      isCompact ? 'text-[11px] text-slate-400' : 'text-xs text-gray-500',
    ),
    sectionSubtitle: isCompact ? 'text-[11px] text-slate-500' : 'text-xs text-gray-500',
    panel: clsx('rounded-xl border p-4', isCompact ? 'border-white/10 bg-slate-900/60' : 'border-gray-800 bg-gray-900/50'),
    cardBase: clsx(
      'flex flex-col items-center rounded-lg border transition-all text-center',
      isCompact ? 'border-white/10 bg-slate-900/60' : 'border-gray-700 bg-gray-800/30',
    ),
    cardActive: isCompact
      ? 'border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]'
      : 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50',
    cardIdle: isCompact ? 'text-slate-300 hover:border-flow-accent/40 hover:text-white' : 'text-gray-300 hover:border-gray-600',
    cardLabel: isCompact ? 'text-[11px] font-medium text-white' : 'text-sm font-medium text-surface',
    cardSubtitle: isCompact ? 'text-[10px] text-slate-400' : 'text-xs text-gray-500',
    previewSize: isCompact ? 'h-10 w-10' : 'h-12 w-12',
    imagePreviewSize: isCompact ? 'h-12 w-12' : 'h-12 w-12',
    input: clsx(
      'w-full rounded-lg border px-3 py-2 focus:outline-none focus:ring-2 focus:ring-flow-accent/50',
      isCompact ? 'border-white/10 bg-slate-900/60 text-xs text-slate-100' : 'border-gray-700 bg-gray-800 text-sm text-surface',
    ),
    textMuted: isCompact ? 'text-slate-400' : 'text-gray-400',
    textSubtle: isCompact ? 'text-slate-500' : 'text-gray-500',
    ghostButton: isCompact
      ? 'rounded-lg px-3 py-1.5 text-[11px] text-flow-accent transition-colors hover:bg-blue-900/30'
      : 'rounded-lg px-3 py-1.5 text-xs text-flow-accent transition-colors hover:bg-blue-900/30',
    dangerButton: isCompact
      ? 'rounded-lg p-1.5 text-slate-500 transition-colors hover:bg-red-900/20 hover:text-red-400'
      : 'rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-900/20 hover:text-red-400',
    dashedButton: clsx(
      'flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed transition-colors',
      isCompact
        ? 'border-white/10 p-4 text-slate-400 hover:border-flow-accent/40 hover:text-white'
        : 'border-gray-700 p-4 text-gray-400 hover:border-gray-600 hover:text-white',
    ),
  };
};

export function ReplayBackgroundSettings({
  background,
  onBackgroundChange,
  variant = 'settings',
}: ReplayBackgroundSettingsProps) {
  const styles = useMemo(() => getStyles(variant), [variant]);
  const [activeTab, setActiveTab] = useState<BackgroundTab>('presets');
  const [gradientMode, setGradientMode] = useState<'simple' | 'advanced'>('simple');
  const [gradientCssDraft, setGradientCssDraft] = useState('');
  const [gradientCssError, setGradientCssError] = useState<string | null>(null);
  const [showBackgroundAssetPicker, setShowBackgroundAssetPicker] = useState(false);
  const [imageDraft, setImageDraft] = useState<{ assetId: string; fit: ReplayBackgroundImageFit }>({
    assetId: '',
    fit: 'cover',
  });
  const { assets, isInitialized, initialize } = useAssetStore();

  useEffect(() => {
    if (!isInitialized) {
      void initialize();
    }
  }, [initialize, isInitialized]);

  useEffect(() => {
    if (background.type !== 'image') {
      return;
    }
    setImageDraft({
      assetId: background.assetId ?? '',
      fit: background.fit ?? 'cover',
    });
  }, [background]);

  const gradientSpec =
    background.type === 'gradient' ? background.value : DEFAULT_REPLAY_GRADIENT_SPEC;
  const gradientStops = useMemo<ReplayGradientStop[]>(
    () => normalizeGradientStops(gradientSpec.stops),
    [gradientSpec.stops],
  );
  const gradientType = gradientSpec.type ?? 'linear';
  const gradientCenter = gradientSpec.center ?? { x: 50, y: 50 };
  const gradientAngle =
    gradientSpec.type === 'linear' && typeof gradientSpec.angle === 'number'
      ? gradientSpec.angle
      : DEFAULT_REPLAY_GRADIENT_SPEC.angle ?? 135;
  const baseGradientSpec: ReplayGradientSpec = {
    type: gradientType,
    angle: gradientType === 'linear' ? gradientAngle : undefined,
    center: gradientType === 'radial' ? gradientCenter : undefined,
    stops: gradientStops,
  };

  const backgroundThemeId = getReplayBackgroundThemeId(background);
  const selectedBackgroundAsset = useMemo(() => {
    if (background.type !== 'image' || !background.assetId) {
      return null;
    }
    return assets.find((asset) => asset.id === background.assetId) ?? null;
  }, [assets, background]);
  const imagePreviewUrl = selectedBackgroundAsset?.thumbnail ?? (background.type === 'image' ? background.url : undefined);

  useEffect(() => {
    if (background.type !== 'gradient' || gradientMode !== 'advanced') {
      return;
    }
    setGradientCssDraft(buildGradientCss(baseGradientSpec));
    setGradientCssError(null);
  }, [background.type, baseGradientSpec, gradientMode]);

  const applyGradientSpec = useCallback(
    (next: ReplayGradientSpec) => {
      onBackgroundChange({
        type: 'gradient',
        value: {
          ...next,
          type: next.type ?? 'linear',
          stops: normalizeGradientStops(next.stops),
        },
      });
    },
    [onBackgroundChange],
  );

  const applyImageDraft = useCallback(
    (draft: { assetId: string; fit: ReplayBackgroundImageFit }) => {
      onBackgroundChange({
        type: 'image',
        assetId: draft.assetId.trim() || undefined,
        fit: draft.fit,
      });
    },
    [onBackgroundChange],
  );

  const handleGradientTypeChange = (value: 'linear' | 'radial') => {
    applyGradientSpec({
      ...baseGradientSpec,
      type: value,
      angle: value === 'linear' ? gradientAngle : undefined,
      center: value === 'radial' ? gradientCenter : undefined,
    });
  };

  const handleGradientStopChange = (
    index: number,
    update: Partial<(typeof gradientStops)[number]>,
  ) => {
    const nextStops = gradientStops.map((stop: ReplayGradientStop, i: number) =>
      i === index ? { ...stop, ...update } : stop,
    );
    applyGradientSpec({ ...baseGradientSpec, stops: nextStops });
  };

  const handleRemoveGradientStop = (index: number) => {
    const nextStops = gradientStops.filter((_: ReplayGradientStop, i: number) => i !== index);
    applyGradientSpec({ ...baseGradientSpec, stops: nextStops });
  };

  const handleAddGradientStop = () => {
    if (gradientStops.length >= 4) {
      return;
    }
    const lastStop = gradientStops[gradientStops.length - 1];
    const nextStops = [
      ...gradientStops,
      {
        color: lastStop?.color ?? '#ffffff',
        position: typeof lastStop?.position === 'number' ? Math.min(100, lastStop.position + 20) : 100,
      },
    ];
    applyGradientSpec({ ...baseGradientSpec, stops: nextStops });
  };

  const handleApplyGradientCss = () => {
    const parsed = parseGradientCss(gradientCssDraft.trim());
    if (!parsed) {
      setGradientCssError('Invalid gradient syntax.');
      return;
    }
    applyGradientSpec(parsed);
    setGradientCssError(null);
  };

  const renderOptionGrid = (options: typeof REPLAY_BACKGROUND_OPTIONS, selectedId?: string) => {
    const gridCols = styles.isCompact ? 'grid-cols-2' : 'grid-cols-2 sm:grid-cols-4';
    return (
      <div className={clsx('grid gap-2', gridCols)}>
        {options.map((option) => {
          const isActive = selectedId === option.id;
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => onBackgroundChange({ type: 'theme', id: option.id })}
              className={clsx(styles.cardBase, isActive ? styles.cardActive : styles.cardIdle)}
            >
              <span
                className={clsx('relative mb-2 overflow-hidden rounded-lg border border-white/10 shadow-inner', styles.previewSize)}
                style={option.previewStyle}
              >
                {option.previewNode}
              </span>
              <span className={styles.cardLabel}>{option.label}</span>
              <span className={styles.cardSubtitle}>{option.subtitle}</span>
            </button>
          );
        })}
      </div>
    );
  };

  const renderImageCard = () => {
    const isActive = background.type === 'image';
    return (
      <button
        type="button"
        onClick={() => setActiveTab('image')}
        className={clsx(styles.cardBase, isActive ? styles.cardActive : styles.cardIdle)}
      >
        <span
          className={clsx(
            'relative mb-2 overflow-hidden rounded-lg border border-white/10 shadow-inner',
            styles.imagePreviewSize,
          )}
          style={imagePreviewUrl ? { backgroundImage: `url(${imagePreviewUrl})`, backgroundSize: imageDraft.fit, backgroundPosition: 'center' } : undefined}
        >
          {!imagePreviewUrl && (
            <span className="flex h-full w-full items-center justify-center">
              <ImageIcon size={18} className={styles.textSubtle} />
            </span>
          )}
        </span>
        <span className={styles.cardLabel}>{selectedBackgroundAsset?.name ?? 'Background Image'}</span>
        <span className={styles.cardSubtitle}>
          {selectedBackgroundAsset ? `${selectedBackgroundAsset.width}x${selectedBackgroundAsset.height}` : 'Customize image'}
        </span>
      </button>
    );
  };

  const renderPresetSection = (section: BackgroundSection, options?: typeof REPLAY_BACKGROUND_OPTIONS) => {
    const tabId = getTabFromSection(section);
    return (
      <div className="space-y-3">
        <div className="flex items-center justify-between gap-2">
          <span className={styles.sectionTitle}>{SECTION_LABELS[section]}</span>
          <button
            type="button"
            onClick={() => setActiveTab(tabId)}
            className={clsx(
              'rounded-full px-3 py-1 text-[10px] font-medium transition-colors',
              styles.isCompact ? 'bg-slate-900/60 text-slate-300 hover:bg-slate-900/80' : 'bg-gray-800 text-gray-300 hover:bg-gray-700',
            )}
          >
            Customize
          </button>
        </div>
        {section === 'image' ? renderImageCard() : renderOptionGrid(options ?? [], background.type === 'theme' ? backgroundThemeId : undefined)}
      </div>
    );
  };

  const gradientOptions = useMemo(() => buildSectionOptions('gradient'), []);
  const solidOptions = useMemo(() => buildSectionOptions('solid'), []);
  const patternOptions = useMemo(() => buildSectionOptions('pattern'), []);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        {TAB_OPTIONS.map((tab) => {
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={clsx(styles.tabButton, isActive ? styles.tabButtonActive : styles.tabButtonIdle)}
            >
              {tab.label}
            </button>
          );
        })}
      </div>

      {activeTab === 'presets' && (
        <div className="space-y-4">
          {renderPresetSection('solid', solidOptions)}
          {renderPresetSection('gradient', gradientOptions)}
          {renderPresetSection('pattern', patternOptions)}
          {renderPresetSection('image')}
        </div>
      )}

      {activeTab === 'solid' && renderOptionGrid(solidOptions, background.type === 'theme' ? backgroundThemeId : undefined)}

      {activeTab === 'pattern' && renderOptionGrid(patternOptions, background.type === 'theme' ? backgroundThemeId : undefined)}

      {activeTab === 'gradient' && (
        <div className={clsx('space-y-3', styles.panel)}>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className={styles.sectionTitle}>Gradient editor</span>
            <div className={clsx('flex items-center gap-2', styles.isCompact ? 'text-[11px]' : 'text-xs')}>
              <button
                type="button"
                onClick={() => setGradientMode('simple')}
                className={clsx(
                  'rounded-full px-3 py-1.5 font-medium transition-colors',
                  gradientMode === 'simple' ? 'bg-flow-accent text-white' : styles.tabButtonIdle,
                )}
              >
                Simple
              </button>
              <button
                type="button"
                onClick={() => setGradientMode('advanced')}
                className={clsx(
                  'rounded-full px-3 py-1.5 font-medium transition-colors',
                  gradientMode === 'advanced' ? 'bg-flow-accent text-white' : styles.tabButtonIdle,
                )}
              >
                Advanced
              </button>
            </div>
          </div>

          {gradientMode === 'simple' ? (
            <div className="space-y-3">
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => handleGradientTypeChange('linear')}
                  className={clsx(
                    'rounded-full px-3 py-1.5 font-medium transition-colors',
                    gradientType === 'linear' ? 'bg-flow-accent text-white' : styles.tabButtonIdle,
                  )}
                >
                  Linear
                </button>
                <button
                  type="button"
                  onClick={() => handleGradientTypeChange('radial')}
                  className={clsx(
                    'rounded-full px-3 py-1.5 font-medium transition-colors',
                    gradientType === 'radial' ? 'bg-flow-accent text-white' : styles.tabButtonIdle,
                  )}
                >
                  Radial
                </button>
              </div>

              {gradientType === 'linear' ? (
                <div>
                  <div className={clsx('flex items-center justify-between', styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
                    <span>Gradient angle</span>
                    <span>{Math.round(gradientAngle)}°</span>
                  </div>
                  <RangeSlider
                    min={0}
                    max={360}
                    step={5}
                    value={gradientAngle}
                    onChange={(value) =>
                      applyGradientSpec({
                        ...baseGradientSpec,
                        type: 'linear',
                        angle: value,
                        center: undefined,
                      })
                    }
                    ariaLabel="Gradient angle"
                  />
                </div>
              ) : (
                <div className="space-y-2">
                  <div className={clsx('flex items-center justify-between', styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
                    <span>Gradient center</span>
                    <span>
                      {Math.round(gradientCenter.x)}% / {Math.round(gradientCenter.y)}%
                    </span>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <label className={clsx('flex flex-col gap-1', styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
                      X position
                      <RangeSlider
                        min={0}
                        max={100}
                        step={1}
                        value={gradientCenter.x}
                        onChange={(value) =>
                          applyGradientSpec({
                            ...baseGradientSpec,
                            type: 'radial',
                            center: { ...gradientCenter, x: value },
                          })
                        }
                        ariaLabel="Radial center X"
                      />
                    </label>
                    <label className={clsx('flex flex-col gap-1', styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
                      Y position
                      <RangeSlider
                        min={0}
                        max={100}
                        step={1}
                        value={gradientCenter.y}
                        onChange={(value) =>
                          applyGradientSpec({
                            ...baseGradientSpec,
                            type: 'radial',
                            center: { ...gradientCenter, y: value },
                          })
                        }
                        ariaLabel="Radial center Y"
                      />
                    </label>
                  </div>
                </div>
              )}

              <div className="space-y-2">
                <div className={clsx('flex items-center justify-between', styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
                  <span>Stops</span>
                  <span>{gradientStops.length} / 4</span>
                </div>
                <div className="space-y-2">
                  {gradientStops.map((stop: ReplayGradientStop, index: number) => (
                    <div key={`gradient-stop-${index}`} className="flex items-center gap-2">
                      <input
                        type="color"
                        value={stop.color}
                        onChange={(event) => handleGradientStopChange(index, { color: event.target.value })}
                        className={clsx(
                          'h-9 w-10 rounded-lg border p-1',
                          styles.isCompact ? 'border-white/10 bg-slate-900/60' : 'border-gray-700 bg-gray-800',
                        )}
                      />
                      <RangeSlider
                        min={0}
                        max={100}
                        step={1}
                        value={stop.position ?? 0}
                        onChange={(value) => handleGradientStopChange(index, { position: value })}
                        ariaLabel={`Gradient stop ${index + 1} position`}
                        className="flex-1"
                      />
                      <span className={clsx('w-10 text-right', styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
                        {Math.round(stop.position ?? 0)}%
                      </span>
                        {gradientStops.length > 2 && (
                          <button
                            type="button"
                            onClick={() => handleRemoveGradientStop(index)}
                            className={clsx(
                              'inline-flex h-7 w-7 items-center justify-center rounded-full border',
                              styles.isCompact ? 'border-white/10 text-slate-400 hover:text-white' : 'border-gray-700 text-gray-400 hover:text-white',
                            )}
                            aria-label="Remove gradient stop"
                          >
                            <X size={12} />
                          </button>
                        )}
                    </div>
                  ))}
                </div>
                <button
                  type="button"
                  onClick={handleAddGradientStop}
                  disabled={gradientStops.length >= 4}
                  className={clsx(
                    'w-full rounded-lg border px-3 py-2 transition-colors disabled:cursor-not-allowed disabled:opacity-50',
                    styles.isCompact
                      ? 'border-white/10 bg-slate-900/60 text-[11px] text-slate-300 hover:bg-slate-900/80'
                      : 'border-gray-700 bg-gray-800 text-xs text-gray-300 hover:bg-gray-700',
                  )}
                >
                  Add stop
                </button>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              <label className={clsx(styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>CSS gradient</label>
              <textarea
                value={gradientCssDraft}
                onChange={(event) => setGradientCssDraft(event.target.value)}
                rows={3}
                className={clsx(styles.input, styles.isCompact ? 'text-[11px]' : 'text-xs')}
                placeholder="linear-gradient(135deg, #38bdf8 0%, #818cf8 100%)"
              />
              {gradientCssError && <p className={clsx(styles.isCompact ? 'text-[11px]' : 'text-xs', 'text-rose-300')}>{gradientCssError}</p>}
              <button
                type="button"
                onClick={handleApplyGradientCss}
                className={clsx(
                  'w-full rounded-lg bg-flow-accent px-3 py-2 font-medium text-white shadow-[0_12px_35px_rgba(59,130,246,0.35)]',
                  styles.isCompact ? 'text-[11px]' : 'text-xs',
                )}
              >
                Apply CSS
              </button>
              <p className={clsx(styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textSubtle)}>
                Supports linear-gradient and radial-gradient syntax.
              </p>
            </div>
          )}
        </div>
      )}

      {activeTab === 'image' && (
        <div className={clsx('space-y-3', styles.panel)}>
          <div className="space-y-2">
            <label className={clsx(styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
              Background image asset
            </label>
            {selectedBackgroundAsset ? (
              <div className={clsx('flex items-center gap-3 rounded-lg border p-3', styles.isCompact ? 'border-white/10 bg-slate-900/80' : 'border-gray-800 bg-gray-900/80')}>
                <div className={clsx('overflow-hidden rounded-lg', styles.imagePreviewSize, styles.isCompact ? 'bg-slate-900' : 'bg-gray-900')}>
                  {selectedBackgroundAsset.thumbnail ? (
                    <img
                      src={selectedBackgroundAsset.thumbnail}
                      alt={selectedBackgroundAsset.name}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center">
                      <ImageIcon size={18} className={styles.textSubtle} />
                    </div>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <span className={clsx('block truncate', styles.isCompact ? 'text-xs text-white' : 'text-sm text-white')}>
                    {selectedBackgroundAsset.name}
                  </span>
                  <span className={clsx(styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textSubtle)}>
                    {selectedBackgroundAsset.width}x{selectedBackgroundAsset.height}
                  </span>
                </div>
                <div className="flex gap-2">
                  <button type="button" onClick={() => setShowBackgroundAssetPicker(true)} className={styles.ghostButton}>
                    Change
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      const next = { ...imageDraft, assetId: '' };
                      setImageDraft(next);
                      applyImageDraft(next);
                    }}
                    className={styles.dangerButton}
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>
            ) : (
              <button type="button" onClick={() => setShowBackgroundAssetPicker(true)} className={styles.dashedButton}>
                <ImageIcon size={18} />
                <span className={styles.isCompact ? 'text-[11px]' : 'text-sm'}>Select background asset</span>
              </button>
            )}
          </div>
          <label className={clsx('flex flex-col gap-1', styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textMuted)}>
            Image fit
            <select
              value={imageDraft.fit}
              onChange={(event) => {
                const next = { ...imageDraft, fit: event.target.value as ReplayBackgroundImageFit };
                setImageDraft(next);
                applyImageDraft(next);
              }}
              className={styles.input}
            >
              <option value="cover">Cover</option>
              <option value="contain">Contain</option>
            </select>
          </label>
          <p className={clsx(styles.isCompact ? 'text-[11px]' : 'text-xs', styles.textSubtle)}>
            Choose a background asset from your brand library.
          </p>
        </div>
      )}

      <AssetPicker
        isOpen={showBackgroundAssetPicker}
        onClose={() => setShowBackgroundAssetPicker(false)}
        onSelect={(assetId) => {
          const next = { ...imageDraft, assetId: assetId ?? '' };
          setImageDraft(next);
          applyImageDraft(next);
        }}
        selectedId={imageDraft.assetId || null}
        title="Select Background Image"
        filterType="background"
      />
    </div>
  );
}

export default ReplayBackgroundSettings;
