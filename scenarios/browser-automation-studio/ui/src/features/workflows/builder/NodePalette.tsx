import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { ChevronDown, Code, History, Search, Star } from "lucide-react";
import {
  NODE_CATEGORIES,
  WORKFLOW_NODE_DEFINITIONS,
  type NodeCategory,
  type WorkflowNodeDefinition,
} from "@constants/nodeCategories";
import { selectors } from "@constants/selectors";

const FAVORITES_KEY = "bas.palette.favorites";
const RECENTS_KEY = "bas.palette.recents";
const EXPANDED_KEY = "bas.palette.categories";
const MAX_FAVORITES = 5;
const MAX_RECENTS = 5;
const DEFAULT_EXPANDED_IDS = NODE_CATEGORIES.filter(
  (category) => category.defaultExpanded,
).map((c) => c.id);

interface CategoryWithNodes extends NodeCategory {
  resolvedNodes: WorkflowNodeDefinition[];
}

const normalizeQuery = (value: string) => value.trim().toLowerCase();

const escapeRegExp = (value: string) =>
  value.replace(/[-/\\^$*+?.()|[\]{}]/g, "\\$&");

const fuzzyMatch = (value: string, query: string) => {
  if (!query) {
    return true;
  }
  let queryIndex = 0;
  const lowerValue = value.toLowerCase();
  const lowerQuery = query.toLowerCase();
  for (
    let i = 0;
    i < lowerValue.length && queryIndex < lowerQuery.length;
    i += 1
  ) {
    if (lowerValue[i] === lowerQuery[queryIndex]) {
      queryIndex += 1;
    }
  }
  return queryIndex === lowerQuery.length;
};

const matchesQuery = (node: WorkflowNodeDefinition, query: string) => {
  if (!query) {
    return true;
  }
  const haystacks = [node.label, node.description, ...(node.keywords ?? [])];
  return haystacks.some((value) => {
    const lower = value.toLowerCase();
    return lower.includes(query) || fuzzyMatch(lower, query);
  });
};

const highlightText = (text: string, query: string) => {
  if (!query) {
    return text;
  }
  const safeQuery = escapeRegExp(query);
  const splitRegex = new RegExp(`(${safeQuery})`, "ig");
  const matchRegex = new RegExp(`^${safeQuery}$`, "i");
  const parts = text.split(splitRegex);
  return parts.map((part, index) =>
    matchRegex.test(part) ? (
      <mark
        key={`${part}-${index}`}
        className="bg-flow-accent/20 text-flow-accent rounded px-0.5"
      >
        {part}
      </mark>
    ) : (
      <Fragment key={`${part}-${index}`}>{part}</Fragment>
    ),
  );
};

interface NodeCardProps {
  node: WorkflowNodeDefinition;
  onDragStart: (event: React.DragEvent<HTMLDivElement>, type: string) => void;
  onToggleFavorite: (type: string) => void;
  isFavorite: boolean;
  searchTerm: string;
}

function NodeCard({
  node,
  onDragStart,
  onToggleFavorite,
  isFavorite,
  searchTerm,
}: NodeCardProps) {
  const Icon = node.icon;

  return (
    <div
      data-node-type={node.type}
      draggable
      onDragStart={(event) => onDragStart(event, node.type)}
      className="relative bg-flow-bg border border-gray-700 rounded-lg p-3 cursor-move hover:border-flow-accent transition-colors group"
      data-testid={selectors.nodePalette.card({ type: node.type })}
    >
      <button
        type="button"
        aria-label={`${isFavorite ? "Remove" : "Add"} ${node.label} ${isFavorite ? "from" : "to"} favorites`}
        aria-pressed={isFavorite}
        className="absolute right-2 top-2 text-gray-500 hover:text-yellow-300"
        onClick={(event) => {
          event.stopPropagation();
          event.preventDefault();
          onToggleFavorite(node.type);
        }}
        data-testid={selectors.nodePalette.favoriteButton({
          type: node.type,
        })}
      >
        <Star
          size={14}
          className={isFavorite ? "text-yellow-300" : "text-gray-500"}
          fill={isFavorite ? "currentColor" : "none"}
        />
      </button>
      <div className="flex items-start gap-3">
        <div
          className={`mt-0.5 ${node.color} group-hover:scale-110 transition-transform`}
        >
          <Icon size={18} />
        </div>
        <div className="flex-1">
          <div className="font-medium text-sm text-white mb-0.5">
            {highlightText(node.label, searchTerm)}
          </div>
          <div className="text-xs text-gray-500">
            {highlightText(node.description, searchTerm)}
          </div>
        </div>
      </div>
    </div>
  );
}

function NodePalette() {
  const [searchTerm, setSearchTerm] = useState("");
  const [favorites, setFavorites] = useState<Set<string>>(new Set());
  const [recents, setRecents] = useState<string[]>([]);
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    () => new Set(DEFAULT_EXPANDED_IDS),
  );
  const [quickAccessExpanded, setQuickAccessExpanded] = useState(true);
  const [hydrated, setHydrated] = useState(false);
  const searchInputRef = useRef<HTMLInputElement | null>(null);

  const normalizedQuery = normalizeQuery(searchTerm);

  const isValidNodeType = useCallback(
    (nodeType: string) => Boolean(WORKFLOW_NODE_DEFINITIONS[nodeType]),
    [],
  );

  useEffect(() => {
    if (typeof window === "undefined") {
      setHydrated(true);
      return;
    }

    try {
      const storedFavorites = window.localStorage.getItem(FAVORITES_KEY);
      if (storedFavorites) {
        const parsed: unknown = JSON.parse(storedFavorites);
        if (Array.isArray(parsed)) {
          setFavorites(
            new Set(
              parsed.filter(
                (value): value is string =>
                  typeof value === "string" && isValidNodeType(value),
              ),
            ),
          );
        }
      }

      const storedRecents = window.localStorage.getItem(RECENTS_KEY);
      if (storedRecents) {
        const parsed: unknown = JSON.parse(storedRecents);
        if (Array.isArray(parsed)) {
          setRecents(
            parsed
              .filter(
                (value): value is string =>
                  typeof value === "string" && isValidNodeType(value),
              )
              .slice(0, MAX_RECENTS),
          );
        }
      }

      const storedExpanded = window.localStorage.getItem(EXPANDED_KEY);
      if (storedExpanded) {
        const parsed: unknown = JSON.parse(storedExpanded);
        if (Array.isArray(parsed)) {
          setExpandedCategories(
            new Set(
              parsed.filter(
                (value): value is string => typeof value === "string",
              ),
            ),
          );
        }
      }
    } catch {
      // Ignore malformed localStorage payloads and fall back to defaults.
    } finally {
      setHydrated(true);
    }
  }, [isValidNodeType]);

  useEffect(() => {
    if (!hydrated || typeof window === "undefined") {
      return;
    }
    window.localStorage.setItem(
      FAVORITES_KEY,
      JSON.stringify(Array.from(favorites)),
    );
  }, [favorites, hydrated]);

  useEffect(() => {
    if (!hydrated || typeof window === "undefined") {
      return;
    }
    window.localStorage.setItem(RECENTS_KEY, JSON.stringify(recents));
  }, [recents, hydrated]);

  useEffect(() => {
    if (!hydrated || typeof window === "undefined") {
      return;
    }
    window.localStorage.setItem(
      EXPANDED_KEY,
      JSON.stringify(Array.from(expandedCategories)),
    );
  }, [expandedCategories, hydrated]);

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        searchInputRef.current?.focus();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  const toggleFavorite = (nodeType: string) => {
    setFavorites((prev) => {
      const next = new Set(prev);
      if (next.has(nodeType)) {
        next.delete(nodeType);
        return next;
      }
      if (next.size >= MAX_FAVORITES) {
        const first = next.values().next();
        if (!first.done) {
          next.delete(first.value);
        }
      }
      next.add(nodeType);
      return next;
    });
  };

  const recordRecent = (nodeType: string) => {
    setRecents((prev) => {
      const filtered = prev.filter((value) => value !== nodeType);
      filtered.unshift(nodeType);
      return filtered.slice(0, MAX_RECENTS);
    });
  };

  const handleDragStart = (
    event: React.DragEvent<HTMLDivElement>,
    nodeType: string,
  ) => {
    event.dataTransfer.setData("nodeType", nodeType);
    event.dataTransfer.effectAllowed = "move";
    recordRecent(nodeType);
    if (normalizedQuery) {
      setSearchTerm("");
    }
  };

  const toggleCategory = (categoryId: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(categoryId)) {
        next.delete(categoryId);
      } else {
        next.add(categoryId);
      }
      return next;
    });
  };

  const filteredCategories = useMemo<CategoryWithNodes[]>(() => {
    return NODE_CATEGORIES.map((category) => {
      const resolvedNodes = category.nodes
        .map((nodeType) => WORKFLOW_NODE_DEFINITIONS[nodeType])
        .filter((node): node is WorkflowNodeDefinition => Boolean(node))
        .filter((node) => matchesQuery(node, normalizedQuery));

      if (normalizedQuery && resolvedNodes.length === 0) {
        return null;
      }

      return { ...category, resolvedNodes } satisfies CategoryWithNodes;
    }).filter((category): category is CategoryWithNodes => Boolean(category));
  }, [normalizedQuery]);

  const favoriteNodes = useMemo(
    () =>
      Array.from(favorites)
        .map((nodeType) => WORKFLOW_NODE_DEFINITIONS[nodeType])
        .filter((node): node is WorkflowNodeDefinition => Boolean(node)),
    [favorites],
  );

  const recentNodes = useMemo(
    () =>
      recents
        .filter((nodeType) => !favorites.has(nodeType))
        .map((nodeType) => WORKFLOW_NODE_DEFINITIONS[nodeType])
        .filter((node): node is WorkflowNodeDefinition => Boolean(node)),
    [recents, favorites],
  );

  const totalVisibleNodes = filteredCategories.reduce(
    (sum, category) => sum + category.resolvedNodes.length,
    0,
  );
  const showQuickAccess = favoriteNodes.length > 0 || recentNodes.length > 0;

  return (
    <div
      className="flex-1 overflow-y-auto p-3"
      data-testid={selectors.nodePalette.container}
    >
      <div className="mb-3">
        <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
          Node Library
        </h3>
        <div className="relative">
          <Search
            size={14}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
          />
          <input
            ref={searchInputRef}
            type="search"
            value={searchTerm}
            onChange={(event) => setSearchTerm(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape" && searchTerm) {
                setSearchTerm("");
              }
            }}
            placeholder="Search nodes (Cmd/Ctrl + K)"
            className="w-full bg-flow-bg border border-gray-700 rounded-lg py-2 pl-9 pr-3 text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-flow-accent"
            data-testid={selectors.nodePalette.searchInput}
          />
        </div>
      </div>

      {showQuickAccess && (
        <div
          className="mb-4 border border-gray-700 rounded-lg"
          data-testid={selectors.nodePalette.quickAccess}
        >
          <button
            type="button"
            className="w-full flex items-center justify-between px-3 py-2 text-xs font-semibold text-gray-400 uppercase tracking-wide"
            onClick={() => setQuickAccessExpanded((value) => !value)}
            aria-expanded={quickAccessExpanded}
            data-testid={selectors.nodePalette.quickAccessToggle}
          >
            <span className="flex items-center gap-2">
              <History size={14} className="text-flow-accent" />
              Quick Access
            </span>
            <ChevronDown
              size={14}
              className={`transition-transform ${quickAccessExpanded ? "rotate-0" : "-rotate-90"}`}
            />
          </button>
          <div
            className="overflow-hidden transition-[max-height] duration-300"
            style={{ maxHeight: quickAccessExpanded ? "600px" : "0px" }}
          >
            <div className="p-3 space-y-3">
              {favoriteNodes.length > 0 && (
                <div data-testid={selectors.nodePalette.favorites.section}>
                  <div className="text-xs text-gray-400 uppercase font-semibold mb-2">
                    Favorites
                  </div>
                  <div className="space-y-2">
                    {favoriteNodes.map((node) => (
                      <NodeCard
                        key={`favorite-${node.type}`}
                        node={node}
                        onDragStart={handleDragStart}
                        onToggleFavorite={toggleFavorite}
                        isFavorite
                        searchTerm={normalizedQuery}
                      />
                    ))}
                  </div>
                </div>
              )}

              {recentNodes.length > 0 && (
                <div data-testid={selectors.nodePalette.recents.section}>
                  <div className="flex items-center justify-between mb-2">
                    <div className="text-xs text-gray-400 uppercase font-semibold">
                      Recent
                    </div>
                    <button
                      type="button"
                      className="text-[11px] text-gray-500 hover:text-gray-300"
                      onClick={() => setRecents([])}
                      data-testid={selectors.nodePalette.recents.clearButton}
                    >
                      Clear
                    </button>
                  </div>
                  <div className="space-y-2">
                    {recentNodes.map((node) => (
                      <NodeCard
                        key={`recent-${node.type}`}
                        node={node}
                        onDragStart={handleDragStart}
                        onToggleFavorite={toggleFavorite}
                        isFavorite={favorites.has(node.type)}
                        searchTerm={normalizedQuery}
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      <div className="space-y-3">
        {filteredCategories.map((category) => {
          const categoryExpanded = normalizedQuery
            ? true
            : expandedCategories.has(category.id);
          const CategoryIcon = category.icon;

          return (
            <div
              key={category.id}
              className="border border-gray-800 rounded-lg"
              data-testid={selectors.nodePalette.category({
                category: category.id,
              })}
            >
              <button
                type="button"
                className="w-full flex items-center gap-3 px-3 py-2 text-left"
                onClick={() => toggleCategory(category.id)}
                aria-expanded={categoryExpanded}
                aria-controls={`${category.id}-nodes`}
                disabled={Boolean(normalizedQuery)}
                data-testid={selectors.nodePalette.categoryToggle({
                  category: category.id,
                })}
              >
                <div className="flex items-center gap-2">
                  <CategoryIcon size={16} className="text-flow-accent" />
                  <div>
                    <div className="text-sm font-semibold text-white">
                      {category.label}
                    </div>
                    <div className="text-[11px] text-gray-500">
                      {category.description}
                    </div>
                  </div>
                </div>
                <ChevronDown
                  size={14}
                  className={`ml-auto text-gray-500 transition-transform ${categoryExpanded ? "rotate-0" : "-rotate-90"}`}
                />
              </button>
              <div
                id={`${category.id}-nodes`}
                className="overflow-hidden transition-[max-height] duration-300"
                style={{ maxHeight: categoryExpanded ? "999px" : "0px" }}
              >
                <div className="p-3 space-y-2">
                  {category.resolvedNodes.map((node) => (
                    <NodeCard
                      key={node.type}
                      node={node}
                      onDragStart={handleDragStart}
                      onToggleFavorite={toggleFavorite}
                      isFavorite={favorites.has(node.type)}
                      searchTerm={normalizedQuery}
                    />
                  ))}
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {normalizedQuery && totalVisibleNodes === 0 && (
        <div
          className="mt-4 text-center text-sm text-gray-500"
          data-testid={selectors.nodePalette.noResults}
        >
          No nodes match "{searchTerm}". Try a different keyword.
        </div>
      )}

      <div className="mt-6 p-3 bg-flow-bg rounded-lg border border-gray-700">
        <div className="flex items-center gap-2 mb-2">
          <Code size={16} className="text-flow-accent" />
          <span className="text-xs font-semibold text-gray-400">PRO TIP</span>
        </div>
        <p className="text-xs text-gray-500">
          Drag nodes from any category or quick access list onto the canvas.
          Favorites pin essential actions, and Cmd/Ctrl + K jumps to search
          instantly.
        </p>
      </div>
    </div>
  );
}

export default NodePalette;
