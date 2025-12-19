import { useState, useRef, useEffect, useCallback } from 'react';
import { ChevronDown, Check, Database } from 'lucide-react';
import clsx from 'clsx';
import {
  useExecutionStore,
  ARTIFACT_PROFILE_DESCRIPTIONS,
  type ArtifactProfile,
} from '../store';

const PROFILE_ORDER: ArtifactProfile[] = ['full', 'standard', 'minimal', 'debug', 'none'];

interface ArtifactProfileSelectorProps {
  /** Compact mode shows only an icon with tooltip */
  compact?: boolean;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Dropdown selector for artifact collection profiles.
 * Controls what artifacts are collected during workflow execution.
 */
export function ArtifactProfileSelector({
  compact = false,
  className,
}: ArtifactProfileSelectorProps) {
  const artifactProfile = useExecutionStore((state) => state.artifactProfile);
  const setArtifactProfile = useExecutionStore((state) => state.setArtifactProfile);

  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  // Close on escape
  useEffect(() => {
    if (!isOpen) return;

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen]);

  const handleSelect = useCallback(
    (profile: ArtifactProfile) => {
      setArtifactProfile(profile);
      setIsOpen(false);
    },
    [setArtifactProfile],
  );

  const selectedProfile = ARTIFACT_PROFILE_DESCRIPTIONS[artifactProfile];

  if (compact) {
    return (
      <div ref={containerRef} className={clsx('relative', className)}>
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          className={clsx(
            'toolbar-button p-1.5 flex items-center gap-1',
            'text-gray-400 hover:text-gray-200',
          )}
          title={`Artifact collection: ${selectedProfile.label}`}
          aria-haspopup="menu"
          aria-expanded={isOpen}
        >
          <Database size={14} />
          <span className="text-[10px] font-medium">{selectedProfile.label}</span>
          <ChevronDown
            size={12}
            className={clsx('transition-transform duration-150', isOpen && 'rotate-180')}
          />
        </button>

        {isOpen && (
          <div
            role="menu"
            className="absolute right-0 z-50 mt-1 w-64 rounded-lg border border-gray-700 bg-gray-900 p-1 shadow-xl"
          >
            <div className="px-2 py-1.5 text-[10px] uppercase tracking-wider text-gray-500">
              Artifact Collection
            </div>
            {PROFILE_ORDER.map((profile) => {
              const info = ARTIFACT_PROFILE_DESCRIPTIONS[profile];
              const isActive = artifactProfile === profile;
              return (
                <button
                  key={profile}
                  type="button"
                  role="menuitemradio"
                  aria-checked={isActive}
                  onClick={() => handleSelect(profile)}
                  className={clsx(
                    'flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm transition-colors',
                    'focus:outline-none focus:ring-2 focus:ring-flow-accent/60',
                    isActive
                      ? 'bg-flow-accent/20 text-white'
                      : 'text-gray-300 hover:bg-gray-800 hover:text-white',
                  )}
                >
                  <div className="flex-1">
                    <div className="flex items-center justify-between">
                      <span className="font-medium">{info.label}</span>
                      {isActive && <Check size={14} className="text-flow-accent" />}
                    </div>
                    <span className="text-[11px] text-gray-500">{info.description}</span>
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>
    );
  }

  // Full-size selector (for settings or dialogs)
  return (
    <div ref={containerRef} className={clsx('relative', className)}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={clsx(
          'flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-sm transition-all',
          'focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-gray-900',
          isOpen
            ? 'border-flow-accent/70 bg-gray-800 text-white'
            : 'border-gray-700 bg-gray-900 text-gray-200 hover:border-gray-600 hover:text-white',
        )}
        aria-haspopup="menu"
        aria-expanded={isOpen}
      >
        <span className="flex items-center gap-2">
          <Database size={16} className="text-gray-400" />
          <span className="flex flex-col">
            <span className="font-medium">{selectedProfile.label}</span>
            <span className="text-[11px] text-gray-500">{selectedProfile.description}</span>
          </span>
        </span>
        <ChevronDown
          size={14}
          className={clsx(
            'ml-2 flex-shrink-0 text-gray-400 transition-transform duration-150',
            isOpen && 'rotate-180 text-white',
          )}
        />
      </button>

      {isOpen && (
        <div
          role="menu"
          className="absolute right-0 z-50 mt-2 w-full min-w-[280px] rounded-xl border border-gray-700 bg-gray-900 p-2 shadow-2xl"
        >
          <div className="px-2 py-1.5 text-[10px] uppercase tracking-wider text-gray-500">
            Select artifact collection profile
          </div>
          <div className="space-y-1">
            {PROFILE_ORDER.map((profile) => {
              const info = ARTIFACT_PROFILE_DESCRIPTIONS[profile];
              const isActive = artifactProfile === profile;
              return (
                <button
                  key={profile}
                  type="button"
                  role="menuitemradio"
                  aria-checked={isActive}
                  onClick={() => handleSelect(profile)}
                  className={clsx(
                    'flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors',
                    'focus:outline-none focus:ring-2 focus:ring-flow-accent/60',
                    isActive
                      ? 'border-flow-accent/60 bg-flow-accent/15 text-white'
                      : 'border-transparent bg-gray-800/50 text-gray-300 hover:border-gray-600 hover:text-white',
                  )}
                >
                  <div className="flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-sm font-medium">{info.label}</span>
                      {isActive && <Check size={14} className="text-flow-accent" />}
                    </div>
                    <span className="text-xs text-gray-500">{info.description}</span>
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

export default ArtifactProfileSelector;
