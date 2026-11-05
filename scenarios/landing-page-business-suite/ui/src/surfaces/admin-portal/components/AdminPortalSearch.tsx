import { useEffect, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowRight, Command, Search } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
  DialogTrigger,
} from '../../../shared/ui/dialog';
import { Input } from '../../../shared/ui/input';
import { NAVIGATION_CONFIG } from '../config/navigation';
import type { NavItem } from '../config/navigation.types';

interface SearchEntry extends NavItem {
  groupLabel: string;
}

interface AdminPortalSearchProps {
  compact?: boolean;
}

function buildSearchEntries(): SearchEntry[] {
  const directEntries: SearchEntry[] = NAVIGATION_CONFIG.directLinks.map((link) => ({
    ...link,
    description: link.id === 'home' ? 'Admin overview, quick flows, and system health' : 'Browse the portal documentation',
    groupLabel: 'Quick access',
  }));
  const groupedEntries = NAVIGATION_CONFIG.groups.flatMap((group) =>
    group.items.map((item) => ({ ...item, groupLabel: group.label })),
  );

  return [...directEntries, ...groupedEntries].filter((entry, index, entries) =>
    entries.findIndex((candidate) => candidate.path === entry.path) === index,
  );
}

function normalize(value: string) {
  return value.trim().toLocaleLowerCase();
}

function scoreEntry(entry: SearchEntry, query: string) {
  const title = normalize(entry.name);
  const description = normalize(entry.description);
  const section = normalize(entry.groupLabel);
  if (!query) return 0;
  if (title === query) return 100;
  if (title.startsWith(query)) return 80;
  if (title.includes(query)) return 60;
  if (section.includes(query)) return 35;
  if (description.includes(query)) return 25;
  return -1;
}

export function AdminPortalSearch({ compact = false }: AdminPortalSearchProps) {
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const entries = useMemo(buildSearchEntries, []);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);

  const results = useMemo(() => {
    const normalizedQuery = normalize(query);
    return entries
      .map((entry, index) => ({ entry, score: scoreEntry(entry, normalizedQuery), index }))
      .filter(({ score }) => !normalizedQuery || score >= 0)
      .sort((left, right) => right.score - left.score || left.index - right.index)
      .slice(0, 8)
      .map(({ entry }) => entry);
  }, [entries, query]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setOpen(true);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => { window.removeEventListener('keydown', onKeyDown); };
  }, []);

  useEffect(() => {
    if (open) {
      window.requestAnimationFrame(() => { inputRef.current?.focus(); });
    } else {
      setQuery('');
      setActiveIndex(0);
    }
  }, [open]);

  useEffect(() => {
    setActiveIndex((current) => Math.min(current, Math.max(results.length - 1, 0)));
  }, [results.length]);

  const goTo = (entry: SearchEntry) => {
    setOpen(false);
    navigate(entry.path);
  };

  const handleKeyDown = (event: ReactKeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setActiveIndex((current) => results.length ? (current + 1) % results.length : 0);
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      setActiveIndex((current) => results.length ? (current - 1 + results.length) % results.length : 0);
    } else if (event.key === 'Enter' && results[activeIndex]) {
      event.preventDefault();
      goTo(results[activeIndex]);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <button
          type="button"
          className={compact
            ? 'inline-flex h-9 w-9 items-center justify-center rounded-md text-slate-300 transition-colors hover:bg-white/10 hover:text-white'
            : 'flex h-9 w-9 items-center justify-center gap-3 rounded-md border border-white/10 bg-white/[0.04] px-0 text-left text-sm text-slate-400 transition-colors hover:border-white/20 hover:bg-white/[0.08] hover:text-slate-200 md:w-56 md:justify-start md:px-3'}
          aria-label="Search admin portal"
          data-testid="admin-search-trigger"
        >
          <Search className="h-4 w-4 shrink-0" />
          {!compact && <span className="hidden flex-1 md:inline">Search admin portal</span>}
          {!compact && <kbd className="hidden rounded border border-white/10 px-1.5 py-0.5 text-[10px] font-medium text-slate-500 sm:inline">⌘K</kbd>}
        </button>
      </DialogTrigger>
      <DialogContent className="max-w-xl gap-0 overflow-hidden border-white/10 bg-slate-900 p-0 shadow-2xl shadow-black/40">
        <DialogTitle className="sr-only">Search admin portal</DialogTitle>
        <DialogDescription className="sr-only">Find a page in the administrator portal by name, section, or description.</DialogDescription>
        <div className="flex items-center gap-3 border-b border-white/10 px-5 py-4">
          <Search className="h-5 w-5 shrink-0 text-sky-400" />
          <Input
            ref={inputRef}
            value={query}
            onChange={(event) => { setQuery(event.target.value); }}
            onKeyDown={handleKeyDown}
            placeholder="Search pages, settings, and tools..."
            aria-label="Search admin pages"
            className="h-10 border-0 bg-transparent px-0 text-base shadow-none focus-visible:ring-0"
            data-testid="admin-search-input"
          />
          <kbd className="hidden rounded border border-white/10 px-1.5 py-1 text-[10px] text-slate-500 sm:inline">ESC</kbd>
        </div>
        <div className="max-h-[min(28rem,60vh)] overflow-y-auto p-2" role="listbox" aria-label="Admin pages">
          {results.length > 0 ? results.map((entry, index) => {
            const Icon = entry.icon;
            const active = index === activeIndex;
            return (
              <button
                key={entry.path}
                type="button"
                role="option"
                aria-selected={active}
                className={`group flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left transition-colors ${active ? 'bg-sky-500/10 text-white' : 'text-slate-300 hover:bg-white/[0.06]'}`}
                onMouseEnter={() => { setActiveIndex(index); }}
                onClick={() => { goTo(entry); }}
                data-testid={`admin-search-result-${entry.id}`}
              >
                <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${active ? 'bg-sky-400/15 text-sky-300' : 'bg-white/[0.06] text-slate-400'}`}>
                  <Icon className="h-4 w-4" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="flex items-center gap-2 text-sm font-medium">
                    <span>{entry.name}</span>
                    {entry.isStub && <span className="rounded-full bg-amber-400/10 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-300">Soon</span>}
                  </span>
                  <span className="mt-0.5 block truncate text-xs text-slate-500">{entry.groupLabel} · {entry.description}</span>
                </span>
                <ArrowRight className={`h-4 w-4 shrink-0 transition-opacity ${active ? 'text-sky-300 opacity-100' : 'opacity-0 group-hover:opacity-60'}`} />
              </button>
            );
          }) : (
            <div className="px-4 py-12 text-center">
              <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-white/[0.06] text-slate-500"><Search className="h-5 w-5" /></div>
              <p className="mt-3 text-sm font-medium text-slate-300">No admin pages found</p>
              <p className="mt-1 text-xs text-slate-500">Try a page name, section, or setting.</p>
            </div>
          )}
        </div>
        <div className="flex items-center gap-4 border-t border-white/10 px-5 py-3 text-xs text-slate-500">
          <span className="inline-flex items-center gap-1.5"><kbd className="rounded border border-white/10 px-1">↑</kbd><kbd className="rounded border border-white/10 px-1">↓</kbd> to navigate</span>
          <span className="inline-flex items-center gap-1.5"><kbd className="rounded border border-white/10 px-1">↵</kbd> to open</span>
          <span className="ml-auto hidden items-center gap-1.5 sm:inline-flex"><Command className="h-3 w-3" /> K to search anytime</span>
        </div>
      </DialogContent>
    </Dialog>
  );
}
