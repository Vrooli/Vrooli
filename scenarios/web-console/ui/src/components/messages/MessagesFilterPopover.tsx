import type { ReactNode } from "react";
import { SlidersHorizontal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Chip } from "@vrooli/react-component-library/Chip";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Popover, PopoverContent, PopoverTrigger } from "@vrooli/react-component-library/Popover";
import { strings } from "../../consts/strings";
import {
  type ContentFilter,
  type GroupMode,
  type MessageFilterState,
  type RoleFilter,
  type SortMode,
  type SourceId,
  type StatusFilter,
} from "../MessageJumpList.helpers";
import type { SearchOptionsState } from "../MessageJumpList";

const STATUS_OPTIONS: StatusFilter[] = ["all", "unheard", "played", "failed", "summarized"];
const STATUS_LABEL_KEY = {
  all: strings.messageJumpList.statusAll,
  unheard: strings.messageJumpList.statusUnheard,
  played: strings.messageJumpList.statusPlayed,
  failed: strings.messageJumpList.statusFailed,
  summarized: strings.messageJumpList.statusSummarized,
} satisfies Record<StatusFilter, string>;

const CONTENT_OPTIONS: ContentFilter[] = ["all", "code", "fileReference", "long"];
const CONTENT_LABEL_KEY = {
  all: strings.messageJumpList.contentAll,
  code: strings.messageJumpList.contentCode,
  fileReference: strings.messageJumpList.contentFile,
  long: strings.messageJumpList.contentLong,
} satisfies Record<ContentFilter, string>;

const SORT_OPTIONS: SortMode[] = ["oldest", "newest"];
const SORT_LABEL_KEY = {
  oldest: strings.messageJumpList.sortOldest,
  newest: strings.messageJumpList.sortNewest,
} satisfies Record<SortMode, string>;

const GROUP_OPTIONS: GroupMode[] = ["turn", "flat", "role"];
const GROUP_LABEL_KEY = {
  turn: strings.messageJumpList.groupTurn,
  flat: strings.messageJumpList.groupFlat,
  role: strings.messageJumpList.groupRole,
} satisfies Record<GroupMode, string>;

const SOURCE_LABEL_KEY = {
  claude: strings.messageJumpList.roleClaude,
  codex: strings.messageJumpList.roleCodex,
  opencode: strings.messageJumpList.roleOpenCode,
  grok: strings.messageJumpList.roleGrok,
} satisfies Record<SourceId, string>;

const SEARCH_MODES = ["text", "regex", "fuzzy"] as const;
const SEARCH_MODE_LABEL_KEY = {
  text: strings.messageJumpList.modeText,
  regex: strings.messageJumpList.modeRegex,
  fuzzy: strings.messageJumpList.modeFuzzy,
} satisfies Record<(typeof SEARCH_MODES)[number], string>;

interface MessagesFilterPopoverProps {
  filters: MessageFilterState;
  onFiltersChange: (filters: MessageFilterState) => void;
  searchOptions: SearchOptionsState;
  onSearchOptionsChange: (options: SearchOptionsState) => void;
  sourceOptions: readonly SourceId[];
  activeCount: number;
  onReset: () => void;
}

function Section({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="mt-3 first:mt-0">
      <div className="mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">{label}</div>
      <div role="group" aria-label={label} className="flex flex-wrap gap-1">
        {children}
      </div>
    </div>
  );
}

/**
 * Always-available messages controls. Search mode, role, content, status, sort
 * and grouping all live here so the reader never has to open the jump
 * navigator just to narrow the list.
 */
export function MessagesFilterPopover({
  filters,
  onFiltersChange,
  searchOptions,
  onSearchOptionsChange,
  sourceOptions,
  activeCount,
  onReset,
}: MessagesFilterPopoverProps) {
  const { t } = useTranslation();
  const set = <K extends keyof MessageFilterState>(key: K, value: MessageFilterState[K]) => {
    onFiltersChange({ ...filters, [key]: value });
  };

  return (
    <Popover placement="bottom-end" responsive="auto">
      <PopoverTrigger asChild>
        <IconButton
          data-testid="messages-filter-toggle"
          data-active={activeCount > 0 || undefined}
          aria-label={activeCount > 0
            ? `${t(strings.messageJumpList.filtersToggle)} (${String(activeCount)} active)`
            : t(strings.messageJumpList.filtersToggle)}
          title={t(strings.messageJumpList.filtersToggle)}
          surface="soft"
          size="xs"
          shape="circle"
          selected={activeCount > 0}
        >
          <SlidersHorizontal />
        </IconButton>
      </PopoverTrigger>
      <PopoverContent
        className="max-h-[min(34rem,calc(100vh-5rem))] w-[min(22rem,calc(100vw-2rem))] overflow-y-auto p-3"
        aria-label={t(strings.messageJumpList.filterAriaLabel)}
      >
        <div className="flex items-center justify-between gap-3">
          <div className="text-xs font-semibold text-wc-text-primary">{t(strings.messageJumpList.filtersToggle)}</div>
          <button
            type="button"
            data-testid="messages-filter-reset"
            onClick={onReset}
            className="min-h-9 rounded-full px-2 text-[11px] font-medium text-wc-text-faint transition hover:bg-wc-surface-input hover:text-wc-text-primary"
          >
            {t(strings.messageJumpList.filterAll)}
          </button>
        </div>

        <Section label={t(strings.messageJumpList.modeLabel)}>
          {SEARCH_MODES.map((mode) => (
            <Chip
              key={mode}
              selected={searchOptions.mode === mode}
              onClick={() => { onSearchOptionsChange({ ...searchOptions, mode }); }}
              aria-label={t(SEARCH_MODE_LABEL_KEY[mode])}
            >
              {t(SEARCH_MODE_LABEL_KEY[mode])}
            </Chip>
          ))}
          <Chip
            selected={searchOptions.caseSensitive}
            onClick={() => { onSearchOptionsChange({ ...searchOptions, caseSensitive: !searchOptions.caseSensitive }); }}
            aria-label={t(strings.messageJumpList.matchCase)}
          >
            Aa
          </Chip>
          <Chip
            selected={searchOptions.wholeWord}
            onClick={() => { onSearchOptionsChange({ ...searchOptions, wholeWord: !searchOptions.wholeWord }); }}
          >
            {t(strings.messageJumpList.wholeWord)}
          </Chip>
          <Chip
            data-testid="messages-search-include-work"
            selected={searchOptions.includeWork === true}
            onClick={() => { onSearchOptionsChange({ ...searchOptions, includeWork: !searchOptions.includeWork }); }}
          >
            {t(strings.messageJumpList.includeWork)}
          </Chip>
        </Section>

        <Section label={t(strings.messageJumpList.sortHeading)}>
          {SORT_OPTIONS.map((option) => (
            <Chip
              key={option}
              selected={filters.sort === option}
              onClick={() => { set("sort", option); }}
            >
              {t(SORT_LABEL_KEY[option])}
            </Chip>
          ))}
        </Section>

        <Section label={t(strings.messageJumpList.groupHeading)}>
          {GROUP_OPTIONS.map((option) => (
            <Chip
              key={option}
              selected={filters.group === option}
              onClick={() => { set("group", option); }}
            >
              {t(GROUP_LABEL_KEY[option])}
            </Chip>
          ))}
        </Section>

        <Section label={t(strings.messageJumpList.filterAriaLabel)}>
          <Chip
            selected={filters.role === "all"}
            onClick={() => { set("role", "all"); }}
          >
            {t(strings.messageJumpList.filterAll)}
          </Chip>
          <Chip
            selected={filters.role === "user"}
            onClick={() => { set("role", "user"); }}
          >
            {t(strings.messageJumpList.filterUser)}
          </Chip>
          <Chip
            selected={filters.role === "assistant"}
            onClick={() => { set("role", "assistant"); }}
          >
            {t(strings.messageJumpList.filterAssistant)}
          </Chip>
        </Section>

        {sourceOptions.length > 0 && (
          <Section label={t(strings.messageJumpList.sourceHeading)}>
            {sourceOptions.map((source) => {
              const value: RoleFilter = `source:${source}`;
              return (
                <Chip
                  key={source}
                  selected={filters.role === value}
                  onClick={() => { set("role", value); }}
                >
                  {t(SOURCE_LABEL_KEY[source])}
                </Chip>
              );
            })}
          </Section>
        )}

        <Section label={t(strings.messageJumpList.statusHeading)}>
          {STATUS_OPTIONS.map((option) => (
            <Chip
              key={option}
              selected={filters.status === option}
              onClick={() => { set("status", option); }}
            >
              {t(STATUS_LABEL_KEY[option])}
            </Chip>
          ))}
        </Section>

        <Section label={t(strings.messageJumpList.contentHeading)}>
          {CONTENT_OPTIONS.map((option) => (
            <Chip
              key={option}
              selected={filters.content === option}
              onClick={() => { set("content", option); }}
            >
              {t(CONTENT_LABEL_KEY[option])}
            </Chip>
          ))}
        </Section>

      </PopoverContent>
    </Popover>
  );
}
