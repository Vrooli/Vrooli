import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocation, useSearchParams } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import type { DocEntry } from '../../../shared/api';
import { useDocsViewer } from '../hooks/useDocsViewer';
import { Book, ChevronRight, ChevronDown, FileText, Folder, FolderOpen, RefreshCw, PanelLeftClose, PanelLeft, GripVertical } from 'lucide-react';

interface TreeNodeProps {
  entry: DocEntry;
  level: number;
  selectedPath: string | null;
  expandedPaths: Set<string>;
  onSelect: (path: string) => void;
  onToggle: (path: string) => void;
}

function TreeNode({ entry, level, selectedPath, expandedPaths, onSelect, onToggle }: TreeNodeProps) {
  const isExpanded = expandedPaths.has(entry.path);
  const isSelected = selectedPath === entry.path;
  const paddingLeft = level * 16 + 8;

  if (entry.isDir) {
    return (
      <div>
        <button
          type="button"
          onClick={() => onToggle(entry.path)}
          className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-white/5 transition-colors"
          style={{ paddingLeft }}
        >
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-slate-400 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-slate-400 flex-shrink-0" />
          )}
          {isExpanded ? (
            <FolderOpen className="h-4 w-4 text-amber-400 flex-shrink-0" />
          ) : (
            <Folder className="h-4 w-4 text-amber-400 flex-shrink-0" />
          )}
          <span className="text-slate-200 truncate">{entry.name}</span>
        </button>
        {isExpanded && entry.children && (
          <div>
            {entry.children.map((child) => (
              <TreeNode
                key={child.path}
                entry={child}
                level={level + 1}
                selectedPath={selectedPath}
                expandedPaths={expandedPaths}
                onSelect={onSelect}
                onToggle={onToggle}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => onSelect(entry.path)}
      className={`flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm transition-colors ${
        isSelected ? 'bg-blue-500/20 text-blue-300' : 'hover:bg-white/5 text-slate-300'
      }`}
      style={{ paddingLeft: paddingLeft + 20 }}
    >
      <FileText className={`h-4 w-4 flex-shrink-0 ${isSelected ? 'text-blue-400' : 'text-slate-500'}`} />
      <span className="truncate">{entry.name.replace(/\.md$/i, '')}</span>
    </button>
  );
}

function buildHeadingId(text: string, counts: Map<string, number>): string {
  const base = text
    .toLowerCase()
    .trim()
    .replace(/[`*_~]/g, '')
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-');

  if (!base) {
    const fallback = `section-${counts.size + 1}`;
    counts.set(fallback, 1);
    return fallback;
  }

  const count = counts.get(base) ?? 0;
  counts.set(base, count + 1);
  if (count === 0) {
    return base;
  }
  return `${base}-${count + 1}`;
}

function MarkdownRenderer({ content }: { content: string }) {
  // Simple markdown rendering with basic formatting
  const lines = content.split('\n');
  const elements: JSX.Element[] = [];
  let i = 0;
  let inCodeBlock = false;
  let codeBlockContent: string[] = [];
  const headingCounts = new Map<string, number>();

  while (i < lines.length) {
    const line = lines[i];
    // Skip undefined lines (shouldn't happen, but TypeScript's noUncheckedIndexedAccess requires this)
    if (line === undefined) {
      i++;
      continue;
    }

    // Code blocks
    if (line.startsWith('```')) {
      if (!inCodeBlock) {
        inCodeBlock = true;
        codeBlockContent = [];
      } else {
        elements.push(
          <pre key={i} className="my-4 rounded-lg bg-slate-800/80 border border-white/10 p-4 overflow-x-auto">
            <code className="text-sm text-slate-200 font-mono">{codeBlockContent.join('\n')}</code>
          </pre>
        );
        inCodeBlock = false;
        codeBlockContent = [];
      }
      i++;
      continue;
    }

    if (inCodeBlock) {
      codeBlockContent.push(line);
      i++;
      continue;
    }

    // Headings
    if (line.startsWith('# ')) {
      const headingText = line.slice(2);
      const headingId = buildHeadingId(headingText, headingCounts);
      elements.push(
        <h1
          key={i}
          id={headingId}
          className="text-3xl font-bold text-white mt-8 mb-4 first:mt-0 scroll-mt-24"
        >
          {formatInlineMarkdown(headingText)}
        </h1>
      );
    } else if (line.startsWith('## ')) {
      const headingText = line.slice(3);
      const headingId = buildHeadingId(headingText, headingCounts);
      elements.push(
        <h2
          key={i}
          id={headingId}
          className="text-2xl font-semibold text-white mt-6 mb-3 border-b border-white/10 pb-2 scroll-mt-24"
        >
          {formatInlineMarkdown(headingText)}
        </h2>
      );
    } else if (line.startsWith('### ')) {
      const headingText = line.slice(4);
      const headingId = buildHeadingId(headingText, headingCounts);
      elements.push(
        <h3 key={i} id={headingId} className="text-xl font-semibold text-white mt-5 mb-2 scroll-mt-24">
          {formatInlineMarkdown(headingText)}
        </h3>
      );
    } else if (line.startsWith('#### ')) {
      const headingText = line.slice(5);
      const headingId = buildHeadingId(headingText, headingCounts);
      elements.push(
        <h4 key={i} id={headingId} className="text-lg font-medium text-white mt-4 mb-2 scroll-mt-24">
          {formatInlineMarkdown(headingText)}
        </h4>
      );
    }
    // Horizontal rule
    else if (line.match(/^---+$/)) {
      elements.push(<hr key={i} className="my-6 border-white/10" />);
    }
    // Blockquotes
    else if (line.startsWith('> ')) {
      elements.push(
        <blockquote key={i} className="my-4 border-l-4 border-blue-500/50 pl-4 italic text-slate-300">
          {formatInlineMarkdown(line.slice(2))}
        </blockquote>
      );
    }
    // Unordered lists
    else if (line.match(/^[-*]\s/)) {
      const listItems: string[] = [line.replace(/^[-*]\s/, '')];
      let nextLine = lines[i + 1];
      while (i + 1 < lines.length && nextLine?.match(/^[-*]\s/)) {
        i++;
        const currentLine = lines[i];
        if (currentLine) listItems.push(currentLine.replace(/^[-*]\s/, ''));
        nextLine = lines[i + 1];
      }
      elements.push(
        <ul key={i} className="my-4 ml-6 list-disc space-y-1">
          {listItems.map((item, idx) => (
            <li key={idx} className="text-slate-300">{formatInlineMarkdown(item)}</li>
          ))}
        </ul>
      );
    }
    // Ordered lists
    else if (line.match(/^\d+\.\s/)) {
      const listItems: string[] = [line.replace(/^\d+\.\s/, '')];
      let nextLine = lines[i + 1];
      while (i + 1 < lines.length && nextLine?.match(/^\d+\.\s/)) {
        i++;
        const currentLine = lines[i];
        if (currentLine) listItems.push(currentLine.replace(/^\d+\.\s/, ''));
        nextLine = lines[i + 1];
      }
      elements.push(
        <ol key={i} className="my-4 ml-6 list-decimal space-y-1">
          {listItems.map((item, idx) => (
            <li key={idx} className="text-slate-300">{formatInlineMarkdown(item)}</li>
          ))}
        </ol>
      );
    }
    // Tables
    else if (line.includes('|') && line.trim().startsWith('|')) {
      const tableLines: string[] = [line];
      let nextLine = lines[i + 1];
      while (i + 1 < lines.length && nextLine?.includes('|')) {
        i++;
        const currentLine = lines[i];
        if (currentLine) tableLines.push(currentLine);
        nextLine = lines[i + 1];
      }
      elements.push(renderTable(tableLines, i));
    }
    // Paragraphs
    else if (line.trim()) {
      elements.push(
        <p key={i} className="my-3 text-slate-300 leading-relaxed">{formatInlineMarkdown(line)}</p>
      );
    }

    i++;
  }

  return <div className="prose prose-invert max-w-none">{elements}</div>;
}

function renderTable(lines: string[], key: number): JSX.Element {
  const rows = lines.filter(line => !line.match(/^[\s|:-]+$/));
  const firstRow = rows[0];
  if (rows.length === 0 || !firstRow) return <></>;

  const parseCells = (row: string) =>
    row.split('|').filter(cell => cell.trim()).map(cell => cell.trim());

  const headers = parseCells(firstRow);
  const bodyRows = rows.slice(1).map(parseCells);

  return (
    <div key={key} className="my-4 overflow-x-auto">
      <table className="w-full border-collapse border border-white/10">
        <thead>
          <tr className="bg-slate-800/50">
            {headers.map((header, idx) => (
              <th key={idx} className="border border-white/10 px-4 py-2 text-left text-sm font-semibold text-white">
                {formatInlineMarkdown(header)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {bodyRows.map((row, rowIdx) => (
            <tr key={rowIdx} className="hover:bg-white/5">
              {row.map((cell, cellIdx) => (
                <td key={cellIdx} className="border border-white/10 px-4 py-2 text-sm text-slate-300">
                  {formatInlineMarkdown(cell)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatInlineMarkdown(text: string): (string | JSX.Element)[] {
  const parts: (string | JSX.Element)[] = [];
  let remaining = text;
  let keyIdx = 0;

  while (remaining.length > 0) {
    // Bold
    let match = remaining.match(/\*\*(.+?)\*\*/);
    if (match && match.index !== undefined) {
      if (match.index > 0) {
        parts.push(...formatInlineMarkdown(remaining.slice(0, match.index)));
      }
      parts.push(<strong key={keyIdx++} className="font-semibold text-white">{match[1]}</strong>);
      remaining = remaining.slice(match.index + match[0].length);
      continue;
    }

    // Inline code
    match = remaining.match(/`([^`]+)`/);
    if (match && match.index !== undefined) {
      if (match.index > 0) {
        parts.push(remaining.slice(0, match.index));
      }
      parts.push(
        <code key={keyIdx++} className="rounded bg-slate-700/50 px-1.5 py-0.5 font-mono text-sm text-cyan-300">
          {match[1]}
        </code>
      );
      remaining = remaining.slice(match.index + match[0].length);
      continue;
    }

    // Links
    match = remaining.match(/\[([^\]]+)\]\(([^)]+)\)/);
    if (match && match.index !== undefined) {
      const linkText = match[1] ?? '';
      const linkUrl = match[2] ?? '#';
      if (match.index > 0) {
        parts.push(remaining.slice(0, match.index));
      }
      parts.push(
        <a
          key={keyIdx++}
          href={linkUrl}
          className="text-blue-400 hover:text-blue-300 underline"
          target={linkUrl.startsWith('http') ? '_blank' : undefined}
          rel={linkUrl.startsWith('http') ? 'noopener noreferrer' : undefined}
        >
          {linkText}
        </a>
      );
      remaining = remaining.slice(match.index + match[0].length);
      continue;
    }

    // Italic
    match = remaining.match(/\*([^*]+)\*/);
    if (match && match.index !== undefined) {
      if (match.index > 0) {
        parts.push(remaining.slice(0, match.index));
      }
      parts.push(<em key={keyIdx++} className="italic">{match[1]}</em>);
      remaining = remaining.slice(match.index + match[0].length);
      continue;
    }

    // No more patterns, add remaining text
    parts.push(remaining);
    break;
  }

  return parts;
}

const MIN_SIDEBAR_WIDTH = 200;
const MAX_SIDEBAR_WIDTH = 500;
const DEFAULT_SIDEBAR_WIDTH = 280;

export function DocsViewer() {
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const requestedDoc = searchParams.get('doc');
  const requestedAnchor = location.hash ? location.hash.slice(1) : null;
  const {
    tree,
    selectedPath,
    selectedDoc,
    expandedPaths,
    loading,
    loadingDoc,
    error,
    loadTree,
    loadDoc,
    handleToggle,
  } = useDocsViewer({ requestedPath: requestedDoc });

  // Sidebar state
  const [sidebarWidth, setSidebarWidth] = useState(DEFAULT_SIDEBAR_WIDTH);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  // Handle resize drag
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  useEffect(() => {
    if (!isDragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!containerRef.current) return;
      const containerRect = containerRef.current.getBoundingClientRect();
      const newWidth = e.clientX - containerRect.left;
      setSidebarWidth(Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, newWidth)));
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging]);

  // Scroll to anchor when doc loads
  useEffect(() => {
    if (!requestedAnchor || loadingDoc || !selectedDoc) {
      return;
    }

    const schedule = window.requestAnimationFrame ?? ((cb: FrameRequestCallback) => window.setTimeout(cb, 0));
    const cancel = window.cancelAnimationFrame ?? window.clearTimeout;
    const handle = schedule(() => {
      const element = document.getElementById(requestedAnchor);
      if (element && contentRef.current) {
        const elementTop = element.offsetTop;
        contentRef.current.scrollTo({ top: elementTop - 24, behavior: 'smooth' });
      }
    });

    return () => cancel(handle);
  }, [loadingDoc, requestedAnchor, selectedDoc]);

  const toggleSidebar = useCallback(() => {
    setIsCollapsed(prev => !prev);
  }, []);

  return (
    <AdminLayout maxWidth="extraWide">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Template Documentation"
          description="Browse the documentation files for this landing page template. Learn about configuration, customization, and deployment."
          icon={Book}
          iconBgClass="bg-amber-500/10"
          iconColorClass="text-amber-400"
          testId="docs-header"
          actions={
            <Button variant="ghost" size="sm" onClick={loadTree} className="gap-2" data-testid="docs-refresh">
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
          }
        />

        {loading ? (
          <div className="text-slate-400" data-testid="docs-loading">Loading documentation...</div>
        ) : error ? (
          <div className="text-rose-400" data-testid="docs-error">{error}</div>
        ) : tree.length === 0 ? (
          <Card className={LAYOUT.card.base} data-testid="docs-empty">
            <CardContent className="py-12 text-center">
              <Book className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-white mb-2">No Documentation Found</h3>
              <p className="text-slate-400">
                No markdown files were found in the docs/ directory.
              </p>
            </CardContent>
          </Card>
        ) : (
          <Card className={LAYOUT.card.base} data-testid="docs-content">
            <CardHeader className="border-b border-white/10 py-3">
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <Book className="h-5 w-5 text-amber-400" />
                  Documentation Browser
                </CardTitle>
                <div className="flex items-center gap-2">
                    <Button
                    variant="ghost"
                    size="sm"
                    onClick={toggleSidebar}
                    className="gap-2"
                    title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
                    data-testid="docs-toggle-sidebar"
                  >
                    {isCollapsed ? (
                      <PanelLeft className="h-4 w-4" />
                    ) : (
                      <PanelLeftClose className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>
            </CardHeader>
            <div
              ref={containerRef}
              className="flex relative"
              style={{ height: 'calc(100vh - 280px)', minHeight: '400px' }}
            >
              {/* Sidebar - File Tree */}
              <div
                className={`flex-shrink-0 border-r border-white/10 overflow-hidden transition-all duration-200 ${
                  isCollapsed ? 'w-0' : ''
                }`}
                style={{ width: isCollapsed ? 0 : sidebarWidth }}
              >
                <div className="h-full flex flex-col">
                  <div className="px-4 py-3 border-b border-white/5">
                    <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
                      <Folder className="h-4 w-4 text-amber-400" />
                      Files
                    </div>
                  </div>
                  <div className="flex-1 overflow-y-auto p-2" data-testid="docs-tree">
                    <div className="space-y-0.5">
                      {tree.map((entry) => (
                        <TreeNode
                          key={entry.path}
                          entry={entry}
                          level={0}
                          selectedPath={selectedPath}
                          expandedPaths={expandedPaths}
                          onSelect={loadDoc}
                          onToggle={handleToggle}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </div>

              {/* Resizable Divider */}
              {!isCollapsed && (
                <div
                  className={`w-1 flex-shrink-0 cursor-col-resize group relative ${
                    isDragging ? 'bg-blue-500/50' : 'hover:bg-blue-500/30'
                  } transition-colors`}
                  onMouseDown={handleMouseDown}
                  data-testid="docs-resize-handle"
                >
                  <div className="absolute inset-y-0 -left-1 -right-1" />
                  <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <GripVertical className="h-6 w-4 text-slate-500" />
                  </div>
                </div>
              )}

              {/* Main Content */}
              <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
                {/* Document Header */}
                <div className="px-6 py-3 border-b border-white/5 flex-shrink-0">
                  <div className="flex items-center gap-2">
                    <FileText className={`h-4 w-4 flex-shrink-0 ${selectedDoc ? 'text-blue-400' : 'text-slate-500'}`} />
                    <span className="text-sm font-medium text-slate-200 truncate">
                      {selectedDoc?.title || 'Select a document'}
                    </span>
                    {selectedPath && (
                      <span className="text-xs text-slate-500 font-mono ml-2 truncate">
                        docs/{selectedPath}
                      </span>
                    )}
                  </div>
                </div>

                {/* Document Content */}
                <div
                  ref={contentRef}
                  className="flex-1 overflow-y-auto px-6 py-4"
                  data-testid="docs-content-area"
                >
                  {loadingDoc ? (
                    <div className="text-slate-400" data-testid="docs-loading-doc">Loading document...</div>
                  ) : selectedDoc ? (
                    <div className="max-w-4xl">
                      <MarkdownRenderer content={selectedDoc.content} />
                    </div>
                  ) : (
                    <div className="text-center py-12" data-testid="docs-no-selection">
                      <Book className="h-12 w-12 text-slate-500 mx-auto mb-4" />
                      <p className="text-slate-400">Select a document from the sidebar to view its contents.</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </Card>
        )}
      </div>
    </AdminLayout>
  );
}
