import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { highlightCodeToHtml, escapeHtml, languageFromPath } from "../utils/highlighting";
import { selectors } from "../../../consts/selectors";

export type CodeViewProps = {
  content?: string;
  path?: string;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
};

export function CodeView({ content, path, isLoading, hasError, errorMessage }: CodeViewProps) {
  const [highlighted, setHighlighted] = useState<string>("");
  const language = useMemo(() => (path ? languageFromPath(path) : undefined), [path]);

  useEffect(() => {
    let cancelled = false;
    const source = content ?? "";
    if (!source) {
      setHighlighted("");
      return;
    }
    highlightCodeToHtml(source, language).then((html) => {
      if (!cancelled) {
        setHighlighted(html);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [content, language]);

  if (isLoading) {
    return (
      <div className="ko-code-state">
        <Loader2 className="h-5 w-5 animate-spin" />
        <span>Loading document...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-code-state ko-code-error">
        <AlertTriangle className="h-5 w-5" />
        <span>{errorMessage}</span>
      </div>
    );
  }

  if (!content) {
    return <div className="ko-code-state">Select a document to view its contents.</div>;
  }

  const safeHtml = highlighted || escapeHtml(content);
  const lines = safeHtml.split("\n");

  return (
    <div className="ko-code-view" data-testid={selectors.viewer.codeView}>
      {lines.map((line, index) => (
        <div key={`${index}-${line.length}`} className="ko-code-line">
          <span className="ko-code-line-number">{index + 1}</span>
          <span
            className="ko-code-line-content"
            dangerouslySetInnerHTML={{ __html: line || "&nbsp;" }}
          />
        </div>
      ))}
    </div>
  );
}
