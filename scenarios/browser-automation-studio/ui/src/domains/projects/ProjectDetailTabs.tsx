import { useRef, useEffect } from "react";
import {
  FileCode,
  History,
  Search,
  X,
  LayoutGrid,
  FolderTree,
} from "lucide-react";
import { usePopoverPosition } from "@hooks/usePopoverPosition";
import { selectors } from "@constants/selectors";
import {
  useProjectDetailStore,
  type ViewMode,
  type ActiveTab,
} from "./hooks/useProjectDetailStore";

interface ProjectDetailTabsProps {
  workflowCount: number;
}

/**
 * Tab navigation and search/view mode controls for ProjectDetail
 */
export function ProjectDetailTabs({ workflowCount }: ProjectDetailTabsProps) {
  // Store state
  const activeTab = useProjectDetailStore((s) => s.activeTab);
  const viewMode = useProjectDetailStore((s) => s.viewMode);
  const searchTerm = useProjectDetailStore((s) => s.searchTerm);
  const showViewModeDropdown = useProjectDetailStore((s) => s.showViewModeDropdown);

  // Store actions
  const setActiveTab = useProjectDetailStore((s) => s.setActiveTab);
  const setViewMode = useProjectDetailStore((s) => s.setViewMode);
  const setSearchTerm = useProjectDetailStore((s) => s.setSearchTerm);
  const setShowViewModeDropdown = useProjectDetailStore((s) => s.setShowViewModeDropdown);

  // Refs for view mode dropdown
  const viewModeButtonRef = useRef<HTMLButtonElement | null>(null);
  const viewModeDropdownRef = useRef<HTMLDivElement | null>(null);

  const { floatingStyles: viewModeDropdownStyles } = usePopoverPosition(
    viewModeButtonRef,
    viewModeDropdownRef,
    {
      isOpen: showViewModeDropdown,
      placementPriority: ["bottom-end", "bottom-start", "top-end", "top-start"],
      matchReferenceWidth: true,
    },
  );

  // Click outside handler for view mode dropdown
  useEffect(() => {
    if (!showViewModeDropdown) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        viewModeDropdownRef.current &&
        !viewModeDropdownRef.current.contains(target) &&
        viewModeButtonRef.current &&
        !viewModeButtonRef.current.contains(target)
      ) {
        setShowViewModeDropdown(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setShowViewModeDropdown(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [showViewModeDropdown, setShowViewModeDropdown]);

  const handleTabChange = (tab: ActiveTab) => {
    setActiveTab(tab);
  };

  const handleViewModeChange = (mode: ViewMode) => {
    setViewMode(mode);
    setShowViewModeDropdown(false);
  };

  return (
    <div>
      {/* Tab buttons */}
      <div className="flex items-center gap-3 border-b border-gray-700 pb-2">
        <button
          data-testid={selectors.workflows.tab}
          onClick={() => handleTabChange("workflows")}
          role="tab"
          aria-selected={activeTab === "workflows"}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === "workflows"
              ? "border-flow-accent text-surface"
              : "border-transparent text-subtle hover:text-surface"
          }`}
        >
          <div className="flex items-center gap-2">
            <FileCode size={16} />
            <span className="whitespace-nowrap">
              Workflows ({workflowCount})
            </span>
          </div>
        </button>
        <button
          data-testid={selectors.projects.tabs.executions}
          onClick={() => handleTabChange("executions")}
          role="tab"
          aria-selected={activeTab === "executions"}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === "executions"
              ? "border-flow-accent text-surface"
              : "border-transparent text-subtle hover:text-surface"
          }`}
        >
          <div className="flex items-center gap-2">
            <History size={16} />
            <span className="whitespace-nowrap">Execution History</span>
          </div>
        </button>
      </div>

      {/* Search and View Mode Toggle - only show for workflows tab */}
      {activeTab === "workflows" && (
        <div className="mt-0 md:mt-4 flex items-center gap-3">
          {/* Search input */}
          <div className="relative flex-1">
            <Search
              size={16}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
            />
            <input
              type="text"
              data-testid={selectors.workflowBuilder.search.input}
              placeholder="Search workflows..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-10 py-2 bg-flow-node border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
            {searchTerm && (
              <button
                type="button"
                data-testid={selectors.workflowBuilder.search.clearButton}
                onClick={() => setSearchTerm("")}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label="Clear workflow search"
              >
                <X size={16} />
              </button>
            )}
          </div>

          {/* Desktop: Toggle buttons */}
          <div className="hidden md:flex items-center gap-2 bg-flow-node border border-gray-700 rounded-lg p-1">
            <button
              onClick={() => handleViewModeChange("card")}
              className={`px-3 py-1.5 rounded flex items-center gap-2 transition-colors ${
                viewMode === "card"
                  ? "bg-flow-accent text-white"
                  : "text-subtle hover:text-surface"
              }`}
              title="Card View"
            >
              <LayoutGrid size={16} />
              <span className="text-sm">Cards</span>
            </button>
            <button
              onClick={() => handleViewModeChange("tree")}
              data-testid={selectors.projects.fileTree.viewModeToggle}
              className={`px-3 py-1.5 rounded flex items-center gap-2 transition-colors ${
                viewMode === "tree"
                  ? "bg-flow-accent text-white"
                  : "text-subtle hover:text-surface"
              }`}
              title="File Tree View"
            >
              <FolderTree size={16} />
              <span className="text-sm">Files</span>
            </button>
          </div>

          {/* Mobile: Icon button with popover */}
          <div className="md:hidden relative">
            <button
              ref={viewModeButtonRef}
              onClick={() => setShowViewModeDropdown(!showViewModeDropdown)}
              className="p-2 bg-flow-node border border-gray-700 rounded-lg text-subtle hover:border-flow-accent hover:text-surface transition-colors"
              title={viewMode === "card" ? "Card View" : "File Tree"}
            >
              {viewMode === "card" ? (
                <LayoutGrid size={20} />
              ) : (
                <FolderTree size={20} />
              )}
            </button>
            {showViewModeDropdown && (
              <div
                ref={viewModeDropdownRef}
                style={viewModeDropdownStyles}
                className="z-30 w-48 rounded-lg border border-gray-700 bg-flow-node shadow-lg overflow-hidden"
              >
                <button
                  onClick={() => handleViewModeChange("card")}
                  className={`w-full flex items-center gap-3 px-4 py-3 transition-colors ${
                    viewMode === "card"
                      ? "bg-flow-accent text-white"
                      : "text-subtle hover:bg-flow-node-hover hover:text-surface"
                  }`}
                >
                  <LayoutGrid size={16} />
                  <span className="text-sm">Card View</span>
                </button>
                <button
                  onClick={() => handleViewModeChange("tree")}
                  className={`w-full flex items-center gap-3 px-4 py-3 transition-colors ${
                    viewMode === "tree"
                      ? "bg-flow-accent text-white"
                      : "text-subtle hover:bg-flow-node-hover hover:text-surface"
                  }`}
                >
                  <FolderTree size={16} />
                  <span className="text-sm">File Tree</span>
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
