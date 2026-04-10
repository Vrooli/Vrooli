// DOC: docs/reference/api-endpoints.md#documentation-search
import { useCallback, useMemo, useState } from "react";
import {
  searchDocFiles,
  searchDocText,
  searchDocUnified,
  type DocFileSearchResult,
  type DocTextSearchMatch,
  type DocUnifiedSearchResponse,
} from "../services/documentationApi";
import {
  buildFileSearchViewModel,
  buildTextSearchViewModel,
  buildUnifiedSearchViewModel,
} from "../controllers/documentationController";
import { recordActivity } from "../lib/activityStore";

export type DocSearchMode = "files" | "text" | "unified";

const DEFAULT_SCOPE = "global";
const DEFAULT_LIMIT = 50;

const modeLabel = (mode: DocSearchMode) => {
  switch (mode) {
    case "files":
      return "File search";
    case "text":
      return "Text search";
    case "unified":
      return "Unified search";
    default:
      return "Documentation search";
  }
};

const splitCSV = (value: string) =>
  value
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean);

export function useDocSearchController(mode: DocSearchMode) {
  const [pattern, setPattern] = useState("");
  const [query, setQuery] = useState("");
  const [scope, setScope] = useState(DEFAULT_SCOPE);
  const [scenario, setScenario] = useState("");
  const [basePath, setBasePath] = useState("");
  const [includeContent, setIncludeContent] = useState(false);
  const [fileTypes, setFileTypes] = useState("md,txt,json,yaml");
  const [caseSensitive, setCaseSensitive] = useState(false);
  const [contextLines, setContextLines] = useState(1);
  const [useSemantic, setUseSemantic] = useState(true);
  const [fileResults, setFileResults] = useState<DocFileSearchResult[] | null>(null);
  const [textResults, setTextResults] = useState<DocTextSearchMatch[] | null>(null);
  const [unifiedResponse, setUnifiedResponse] = useState<DocUnifiedSearchResponse | null>(null);
  const [errorMessage, setErrorMessage] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [hasSubmitted, setHasSubmitted] = useState(false);

  const normalizedScope = scope.trim() || DEFAULT_SCOPE;
  const trimmedScenario = scenario.trim();
  const trimmedBasePath = basePath.trim();

  const validateScope = useCallback(() => {
    if (normalizedScope === "scenario" && !trimmedScenario) {
      return "Scenario name is required for scenario scope.";
    }
    if (normalizedScope === "path" && !trimmedBasePath) {
      return "Base path is required for path scope.";
    }
    return "";
  }, [normalizedScope, trimmedScenario, trimmedBasePath]);

  const submit = useCallback(async (override?: { query?: string; pattern?: string }) => {
    setErrorMessage("");
    const scopeError = validateScope();
    if (scopeError) {
      setErrorMessage(scopeError);
      return;
    }

    if (mode === "files") {
      const patternValue = (override?.pattern ?? pattern).trim();
      if (!patternValue) {
        setErrorMessage("File pattern is required.");
        return;
      }
      setIsLoading(true);
      try {
        const results = await searchDocFiles({
          pattern: patternValue,
          scope: normalizedScope,
          scenario: trimmedScenario || undefined,
          base_path: trimmedBasePath || undefined,
          limit: DEFAULT_LIMIT,
          include_content: includeContent,
        });
        setFileResults(results);
        setHasSubmitted(true);
        recordActivity({
          type: "doc-search",
          title: "Documentation search",
          description: `${modeLabel(mode)} · ${patternValue}`,
          status: "completed",
          meta: {
            mode: modeLabel(mode),
            results: `${results.length}`,
          },
        });
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : "File search failed.");
      } finally {
        setIsLoading(false);
      }
      return;
    }

    if (mode === "text") {
      const queryValue = (override?.query ?? query).trim();
      if (!queryValue) {
        setErrorMessage("Search text is required.");
        return;
      }
      setIsLoading(true);
      try {
        const results = await searchDocText({
          query: queryValue,
          scope: normalizedScope,
          scenario: trimmedScenario || undefined,
          base_path: trimmedBasePath || undefined,
          file_types: splitCSV(fileTypes),
          case_sensitive: caseSensitive,
          limit: DEFAULT_LIMIT,
          context_lines: contextLines,
        });
        setTextResults(results);
        setHasSubmitted(true);
        recordActivity({
          type: "doc-search",
          title: "Documentation search",
          description: `${modeLabel(mode)} · ${queryValue}`,
          status: "completed",
          meta: {
            mode: modeLabel(mode),
            results: `${results.length}`,
          },
        });
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : "Text search failed.");
      } finally {
        setIsLoading(false);
      }
      return;
    }

    const queryValue = (override?.query ?? query).trim();
    const patternValue = (override?.pattern ?? pattern).trim();
    if (!queryValue && !patternValue) {
      setErrorMessage("Query or pattern is required.");
      return;
    }
    setIsLoading(true);
    try {
      const response = await searchDocUnified({
        query: queryValue || undefined,
        pattern: patternValue || undefined,
        scope: normalizedScope,
        scenario: trimmedScenario || undefined,
        base_path: trimmedBasePath || undefined,
        limit: DEFAULT_LIMIT,
        include_content: includeContent,
        file_types: splitCSV(fileTypes),
        case_sensitive: caseSensitive,
        context_lines: contextLines,
        use_semantic: useSemantic,
      });
      setUnifiedResponse(response);
      setHasSubmitted(true);
      recordActivity({
        type: "doc-search",
        title: "Documentation search",
        description: `${modeLabel(mode)} · ${queryValue || patternValue}`,
        status: "completed",
        meta: {
          mode: modeLabel(mode),
          results: `${response.results.length}`,
        },
      });
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Unified search failed.");
    } finally {
      setIsLoading(false);
    }
  }, [
    mode,
    pattern,
    query,
    normalizedScope,
    trimmedScenario,
    trimmedBasePath,
    includeContent,
    fileTypes,
    caseSensitive,
    contextLines,
    useSemantic,
    validateScope,
  ]);

  const clear = useCallback(() => {
    setErrorMessage("");
    setHasSubmitted(false);
    if (mode === "files") {
      setFileResults(null);
    } else if (mode === "text") {
      setTextResults(null);
    } else {
      setUnifiedResponse(null);
    }
  }, [mode]);

  const viewModel = useMemo(() => {
    if (mode === "files") {
      return buildFileSearchViewModel(fileResults ?? [], pattern);
    }
    if (mode === "text") {
      return buildTextSearchViewModel(textResults ?? [], query);
    }
    return buildUnifiedSearchViewModel(unifiedResponse, query || pattern);
  }, [mode, fileResults, textResults, unifiedResponse, pattern, query]);

  const isSubmitDisabled = isLoading;
  const isClearDisabled = !hasSubmitted;

  return {
    pattern,
    setPattern,
    query,
    setQuery,
    scope,
    setScope,
    scenario,
    setScenario,
    basePath,
    setBasePath,
    includeContent,
    setIncludeContent,
    fileTypes,
    setFileTypes,
    caseSensitive,
    setCaseSensitive,
    contextLines,
    setContextLines,
    useSemantic,
    setUseSemantic,
    isLoading,
    hasError: Boolean(errorMessage),
    errorMessage,
    hasData: hasSubmitted,
    isSubmitDisabled,
    isClearDisabled,
    submit,
    clear,
    viewModel,
  };
}
