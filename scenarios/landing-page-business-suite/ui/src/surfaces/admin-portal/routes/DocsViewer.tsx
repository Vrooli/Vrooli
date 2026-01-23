import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import type { DocEntry } from '../../../shared/api';
import { useDocsViewer } from '../hooks/useDocsViewer';
import { Book, ChevronRight, ChevronDown, FileText, Folder, FolderOpen, RefreshCw, ExternalLink } from 'lucide-react';

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

function MarkdownRenderer({ content }: { content: string }) {
  // Simple markdown rendering with basic formatting
  const lines = content.split('\n');
  const elements: JSX.Element[] = [];
  let i = 0;
  let inCodeBlock = false;
  let codeBlockContent: string[] = [];

  while (i < lines.length) {
    const line = lines[i];

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
      elements.push(
        <h1 key={i} className="text-3xl font-bold text-white mt-8 mb-4 first:mt-0">{formatInlineMarkdown(line.slice(2))}</h1>
      );
    } else if (line.startsWith('## ')) {
      elements.push(
        <h2 key={i} className="text-2xl font-semibold text-white mt-6 mb-3 border-b border-white/10 pb-2">{formatInlineMarkdown(line.slice(3))}</h2>
      );
    } else if (line.startsWith('### ')) {
      elements.push(
        <h3 key={i} className="text-xl font-semibold text-white mt-5 mb-2">{formatInlineMarkdown(line.slice(4))}</h3>
      );
    } else if (line.startsWith('#### ')) {
      elements.push(
        <h4 key={i} className="text-lg font-medium text-white mt-4 mb-2">{formatInlineMarkdown(line.slice(5))}</h4>
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
      while (i + 1 < lines.length && lines[i + 1].match(/^[-*]\s/)) {
        i++;
        listItems.push(lines[i].replace(/^[-*]\s/, ''));
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
      while (i + 1 < lines.length && lines[i + 1].match(/^\d+\.\s/)) {
        i++;
        listItems.push(lines[i].replace(/^\d+\.\s/, ''));
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
      while (i + 1 < lines.length && lines[i + 1].includes('|')) {
        i++;
        tableLines.push(lines[i]);
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
  if (rows.length === 0) return <></>;

  const parseCells = (row: string) =>
    row.split('|').filter(cell => cell.trim()).map(cell => cell.trim());

  const headers = parseCells(rows[0]);
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
      if (match.index > 0) {
        parts.push(remaining.slice(0, match.index));
      }
      parts.push(
        <a
          key={keyIdx++}
          href={match[2]}
          className="text-blue-400 hover:text-blue-300 underline"
          target={match[2].startsWith('http') ? '_blank' : undefined}
          rel={match[2].startsWith('http') ? 'noopener noreferrer' : undefined}
        >
          {match[1]}
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

export function DocsViewer() {
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
  } = useDocsViewer();

  return (
    <AdminLayout maxWidth="wide">
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
          <Card className="${LAYOUT.card.base}" data-testid="docs-empty">
            <CardContent className="py-12 text-center">
              <Book className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-white mb-2">No Documentation Found</h3>
              <p className="text-slate-400">
                No markdown files were found in the docs/ directory.
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-4 gap-6" data-testid="docs-content">
            {/* Sidebar - File Tree */}
            <Card className="${LAYOUT.card.base} lg:col-span-1">
              <CardHeader className="pb-3">
                <CardTitle className="flex items-center gap-2 text-base">
                  <Folder className="h-4 w-4 text-amber-400" />
                  Files
                </CardTitle>
              </CardHeader>
              <CardContent className="p-2">
                <div className="space-y-0.5 max-h-[60vh] overflow-y-auto" data-testid="docs-tree">
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
              </CardContent>
            </Card>

            {/* Main Content */}
            <Card className="${LAYOUT.card.base} lg:col-span-3">
              <CardHeader className="border-b border-white/10">
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <FileText className="h-5 w-5 text-blue-400" />
                      {selectedDoc?.title || 'Select a document'}
                    </CardTitle>
                    {selectedPath && (
                      <CardDescription className="mt-1 font-mono text-xs">
                        docs/{selectedPath}
                      </CardDescription>
                    )}
                  </div>
                  {selectedPath && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => window.open(`vscode://file/${selectedPath}`, '_blank')}
                      className="gap-2"
                      title="Open in VS Code"
                    >
                      <ExternalLink className="h-4 w-4" />
                      Open in editor
                    </Button>
                  )}
                </div>
              </CardHeader>
              <CardContent className="p-6">
                {loadingDoc ? (
                  <div className="text-slate-400" data-testid="docs-loading-doc">Loading document...</div>
                ) : selectedDoc ? (
                  <MarkdownRenderer content={selectedDoc.content} />
                ) : (
                  <div className="text-center py-12" data-testid="docs-no-selection">
                    <Book className="h-12 w-12 text-slate-500 mx-auto mb-4" />
                    <p className="text-slate-400">Select a document from the sidebar to view its contents.</p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}
