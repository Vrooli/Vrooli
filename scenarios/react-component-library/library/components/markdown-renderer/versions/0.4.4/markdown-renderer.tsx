/**
 * @libraryId react-component-library:markdown-renderer
 * @displayName markdown-renderer
 * @description Reusable markdown-renderer component implementation for the React component library.
 * @version 0.4.4
 * @tags []
 * @deps {"react":"^18","react-markdown":"^10.1.0","remark-gfm":"^4.0.1","shiki":"^4.3.1","mermaid":"^11.4.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

import {
  Children,
  Component,
  cloneElement,
  Fragment,
  isValidElement,
  type CSSProperties,
  type ErrorInfo,
  type MouseEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { CodeBlock } from "./CodeBlock";
import { InlineCode, type InlineTokenResolution } from "./InlineCode";
import { MermaidDiagram } from "./MermaidDiagram";
import { remarkProsePaths } from "./languageDetection";
import { markdownStyles } from "./markdownStyles";

export type { InlineTokenResolution } from "./InlineCode";
export { CodeBlock } from "./CodeBlock";
export { InlineCode } from "./InlineCode";
export { MermaidDiagram } from "./MermaidDiagram";
export { normalizeCodeLanguage, languageLabel, remarkProsePaths } from "./languageDetection";
export { useCodeCopy } from "./useCodeCopy";
export { resetMermaidRenderCacheForTests, useMermaidSvg } from "./useMermaidSvg";

export interface MarkdownRendererProps {
  content: string;
  className?: string;
  inline?: boolean;
  resolveInlineToken?: (text: string) => InlineTokenResolution | null;
  looksLikeFileReference?: (text: string) => boolean;
  onLinkClick?: (href: string, event: MouseEvent<HTMLAnchorElement>) => void;
  onFileReferenceClick?: (path: string) => void;
  onMermaidOpen?: (code: string) => void;
}

export interface ResizableMarkdownTableProps {
  children?: ReactNode;
  minColumnWidth?: number;
}

type TablePart = ReactElement<{ children?: ReactNode }>;

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

const elementName = (element: ReactElement): string => {
  if (typeof element.type === "string") return element.type;
  const component = element.type as { displayName?: string; name?: string };
  return component.displayName ?? component.name ?? "";
};

const isHeaderPart = (part: ReactNode): part is TablePart => {
  if (!isValidElement(part)) return false;
  if (elementName(part) === "thead") return true;
  const row = Children.toArray(part.props.children)[0];
  const cell = isValidElement(row) ? Children.toArray(row.props.children)[0] : null;
  return isValidElement(cell) && elementName(cell) === "th";
};

/**
 * The markdown table keeps its natural content-based sizing until a user
 * touches a boundary. Once a boundary is moved, the two adjacent columns
 * share the delta, so the table does not silently grow or squeeze every other
 * column. The handles live in the header and also expose keyboard resizing.
 */
export function ResizableMarkdownTable({
  children,
  minColumnWidth = 72,
}: ResizableMarkdownTableProps) {
  const tableRef = useRef<HTMLTableElement>(null);
  const [widths, setWidths] = useState<number[] | null>(null);
  const dragRef = useRef<{ index: number; startX: number; initial: number[] } | null>(null);
  const parts = Children.toArray(children);
  const header = parts.find(isHeaderPart);
  const headerRows = header ? Children.toArray(header.props.children) : [];
  const firstRow = headerRows[0] as TablePart | undefined;
  const columnCount =
    firstRow && isValidElement(firstRow) ? Children.count(firstRow.props.children) : 0;

  const measureWidths = () => {
    const cells = Array.from(
      tableRef.current?.querySelectorAll<HTMLElement>("thead tr:first-child > th") ?? [],
    );
    const measured = cells.map((cell) => cell.getBoundingClientRect().width);
    if (measured.length === columnCount && measured.every((width) => width > 0)) return measured;
    const fallbackWidth = Math.max(
      minColumnWidth,
      (tableRef.current?.clientWidth || 640) / Math.max(columnCount, 1),
    );
    return Array.from({ length: columnCount }, () => fallbackWidth);
  };

  useLayoutEffect(() => {
    if (!tableRef.current || widths || columnCount === 0) return;
    const frame = requestAnimationFrame(() => {
      const measured = measureWidths();
      if (measured.length === columnCount && measured.every((width) => width > 0))
        setWidths(measured);
    });
    return () => cancelAnimationFrame(frame);
  }, [columnCount, widths]);

  useEffect(() => {
    const onMove = (event: PointerEvent) => {
      const drag = dragRef.current;
      if (!drag) return;
      const left = drag.initial[drag.index] ?? 0;
      const right = drag.initial[drag.index + 1] ?? 0;
      const nextLeft = clamp(
        left + event.clientX - drag.startX,
        minColumnWidth,
        left + right - minColumnWidth,
      );
      const delta = nextLeft - left;
      const next = [...drag.initial];
      next[drag.index] = nextLeft;
      next[drag.index + 1] = right - delta;
      setWidths(next);
    };
    const onUp = () => {
      dragRef.current = null;
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
  }, [minColumnWidth]);

  const beginResize = (index: number, event: ReactPointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    const initial = widths ?? measureWidths();
    if (initial.length !== columnCount) return;
    dragRef.current = { index, startX: event.clientX, initial };
    setWidths(initial);
  };

  const adjustResize = (index: number, delta: number) => {
    const initial = widths ?? measureWidths();
    const left = initial[index] ?? 0;
    const right = initial[index + 1] ?? 0;
    const nextLeft = clamp(left + delta, minColumnWidth, left + right - minColumnWidth);
    const next = [...initial];
    next[index] = nextLeft;
    next[index + 1] = right - (nextLeft - left);
    setWidths(next);
  };

  const enhanceHeader = (part: ReactNode) => {
    if (!isHeaderPart(part)) return part;
    const rows = Children.toArray(part.props.children);
    const row = rows[0];
    if (!row || !isValidElement(row)) return part;
    const rowElement = row as ReactElement<{ children?: ReactNode }>;
    const cells = Children.toArray(rowElement.props.children);
    const enhancedRow = cloneElement(
      rowElement,
      {},
      cells.map((cell, index) => {
        if (!isValidElement<{ children?: ReactNode; "aria-label"?: string }>(cell) || index >= cells.length - 1)
          return cell;
        const value = widths?.[index] ?? 0;
        const headerLabel = Children.toArray(cell.props.children)
          .filter((child): child is string | number => typeof child === "string" || typeof child === "number")
          .join("")
          .trim() || undefined;
        return cloneElement(cell, {
          ...(headerLabel ? { "aria-label": headerLabel } : {}),
          children: (
            <>
              {cell.props.children}
              <button
                type="button"
                role="separator"
                aria-orientation="vertical"
                aria-label={`Resize column ${index + 1}`}
                aria-valuemin={minColumnWidth}
                aria-valuemax={Math.max(
                  minColumnWidth,
                  value + (widths?.[index + 1] ?? 0) - minColumnWidth,
                )}
                aria-valuenow={Math.round(value)}
                tabIndex={0}
                data-rcl-md-resize-handle
                onPointerDown={(event) => beginResize(index, event)}
                onKeyDown={(event) => {
                  if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
                    event.preventDefault();
                    adjustResize(index, event.key === "ArrowLeft" ? -16 : 16);
                  }
                }}
              />
            </>
          ),
        });
      }),
    );
    return cloneElement(part, {}, [enhancedRow, ...rows.slice(1)]);
  };

  const totalWidth = widths?.reduce((sum, width) => sum + width, 0);
  return (
    <div className="rcl-md__table-scroll">
      <table
        ref={tableRef}
        style={totalWidth ? { width: `${totalWidth}px`, tableLayout: "fixed" } : undefined}
      >
        {widths ? (
          <colgroup>
            {widths.map((width, index) => (
              <col key={index} style={{ width: `${width}px` }} />
            ))}
          </colgroup>
        ) : null}
        {parts.map((part, index) => (
          <Fragment key={index}>
            {header && index === parts.indexOf(header) ? enhanceHeader(part) : part}
          </Fragment>
        ))}
      </table>
    </div>
  );
}

class MarkdownErrorBoundary extends Component<
  { content: string; children: ReactNode },
  { failed: boolean }
> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  componentDidCatch(_error: Error, _info: ErrorInfo) {}
  render() {
    return this.state.failed ? (
      <pre className="rcl-md__error">{this.props.content}</pre>
    ) : (
      this.props.children
    );
  }
}

const markdownTokens: CSSProperties & Record<`--${string}`, string> = {
  "--markdown-border": "var(--color-border)",
  "--markdown-code-surface": "var(--color-surface-muted)",
  "--markdown-code-text": "var(--color-foreground)",
  "--markdown-muted": "var(--color-muted-foreground)",
  "--markdown-link": "var(--color-accent)",
  "--markdown-error": "var(--color-danger)",
};

export const MarkdownRenderer = withClassName(function MarkdownRenderer({
  content,
  className,
  inline = false,
  resolveInlineToken,
  looksLikeFileReference,
  onLinkClick,
  onFileReferenceClick,
  onMermaidOpen,
}: MarkdownRendererProps) {
  const components = useMemo(
    () => ({
      code: ({ children, className: codeClass }: { children?: ReactNode; className?: string }) => {
        const text = (
          typeof children === "string"
            ? children
            : typeof children === "number"
              ? String(children)
              : ""
        ).replace(/\n$/, "");
        const language = codeClass?.replace(/^language-/, "");
        if (language === "mermaid")
          return <MermaidDiagram code={text} onMermaidOpen={onMermaidOpen} />;
        if (language) return <CodeBlock code={text} language={language} />;
        return (
          <InlineCode
            resolveInlineToken={resolveInlineToken}
            looksLikeFileReference={looksLikeFileReference}
            onLinkClick={onLinkClick}
            onFileReferenceClick={onFileReferenceClick}
          >
            {text}
          </InlineCode>
        );
      },
      a: ({ href = "", children }: { href?: string; children?: ReactNode }) => (
        <a href={href} onClick={(event) => onLinkClick?.(href, event)} className="rcl-md__link">
          {children}
        </a>
      ),
      blockquote: ({ children }: { children?: ReactNode }) => (
        <blockquote className="rcl-md__blockquote">{children}</blockquote>
      ),
      table: ({ children }: { children?: ReactNode }) => (
        <ResizableMarkdownTable>{children}</ResizableMarkdownTable>
      ),
      th: ({ children }: { children?: ReactNode }) => <th>{children}</th>,
      td: ({ children }: { children?: ReactNode }) => <td>{children}</td>,
    }),
    [looksLikeFileReference, onFileReferenceClick, onLinkClick, onMermaidOpen, resolveInlineToken],
  );
  if (!content) return null;
  const Wrapper = inline ? "span" : "div";
  return (
    <MarkdownErrorBoundary content={content}>
      <Wrapper
        className={`rcl-md__root ${className ?? ""}`}
        style={markdownTokens}
        data-rcl-markdown
      >
        <StyleSheet
          libraryId="react-component-library:markdown-renderer"
          version="0.4.3"
          css={markdownStyles}
        />
        <ReactMarkdown remarkPlugins={[remarkGfm, remarkProsePaths]} components={components}>
          {content}
        </ReactMarkdown>
      </Wrapper>
    </MarkdownErrorBoundary>
  );
});

export default MarkdownRenderer;
