import { useState } from "react";
import {
  BookOpen,
  Blocks,
  FileJson,
  X,
  ArrowLeft,
} from "lucide-react";
import { GettingStarted } from "./GettingStarted";
import { NodeReference } from "./NodeReference";
import { SchemaReference } from "./SchemaReference";

type DocsTab = "getting-started" | "node-reference" | "schema-reference";

interface DocsHubProps {
  /** Initial tab to display */
  initialTab?: DocsTab;
  /** Initial node to show in node reference */
  initialNodeType?: string;
  /** Called when the docs hub should close */
  onClose?: () => void;
  /** Whether to show as a full page or a modal */
  mode?: "page" | "modal";
  /** API base URL for fetching live schema */
  apiBaseUrl?: string;
}

const TABS: { id: DocsTab; label: string; icon: React.ElementType }[] = [
  { id: "getting-started", label: "Getting Started", icon: BookOpen },
  { id: "node-reference", label: "Node Reference", icon: Blocks },
  { id: "schema-reference", label: "Schema Reference", icon: FileJson },
];

export function DocsHub({
  initialTab = "getting-started",
  initialNodeType,
  onClose,
  mode = "page",
  apiBaseUrl,
}: DocsHubProps) {
  const [activeTab, setActiveTab] = useState<DocsTab>(
    initialNodeType ? "node-reference" : initialTab
  );

  const content = (
    <div className="flex flex-col h-full bg-flow-bg">
      {/* Header */}
      <header className="flex items-center justify-between px-6 py-4 border-b border-gray-800">
        <div className="flex items-center gap-4">
          {onClose && (
            <button
              type="button"
              onClick={onClose}
              className="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
              aria-label="Back to dashboard"
            >
              <ArrowLeft size={20} />
            </button>
          )}
          <div>
            <h1 className="text-xl font-bold text-white">Documentation</h1>
            <p className="text-sm text-gray-400">
              Learn how to use Browser Automation Studio
            </p>
          </div>
        </div>
        {mode === "modal" && onClose && (
          <button
            type="button"
            onClick={onClose}
            className="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close documentation"
          >
            <X size={20} />
          </button>
        )}
      </header>

      {/* Tab Navigation */}
      <nav className="flex border-b border-gray-800 px-6">
        {TABS.map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                isActive
                  ? "border-flow-accent text-white"
                  : "border-transparent text-gray-400 hover:text-gray-300"
              }`}
            >
              <Icon size={16} />
              {tab.label}
            </button>
          );
        })}
      </nav>

      {/* Tab Content */}
      <div className="flex-1 min-h-0">
        {activeTab === "getting-started" && <GettingStarted />}
        {activeTab === "node-reference" && (
          <NodeReference initialNodeType={initialNodeType} />
        )}
        {activeTab === "schema-reference" && (
          <SchemaReference apiBaseUrl={apiBaseUrl} />
        )}
      </div>
    </div>
  );

  if (mode === "modal") {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
        <div className="w-full h-full max-w-6xl max-h-[90vh] m-4 rounded-xl overflow-hidden shadow-2xl border border-gray-700">
          {content}
        </div>
      </div>
    );
  }

  return content;
}

/**
 * Standalone modal wrapper for DocsHub
 */
interface DocsModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialTab?: DocsTab;
  initialNodeType?: string;
  apiBaseUrl?: string;
}

export function DocsModal({
  isOpen,
  onClose,
  initialTab,
  initialNodeType,
  apiBaseUrl,
}: DocsModalProps) {
  if (!isOpen) return null;

  return (
    <DocsHub
      mode="modal"
      onClose={onClose}
      initialTab={initialTab}
      initialNodeType={initialNodeType}
      apiBaseUrl={apiBaseUrl}
    />
  );
}
