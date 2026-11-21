import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type CSSProperties,
  type DragEvent,
} from "react";
import { FileUp, TriangleAlert, X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { WbButton } from "../primitives";
import { useFocusTrap } from "../useFocusTrap";
import type { Template } from "../templates/spec";
import {
  HTML_LIMITS,
  HtmlImportError,
  htmlTemplate,
  parseHtml,
  type ParsedHtml,
} from "./htmlImport";
import "./html-import.css";

type Phase = "empty" | "parsing" | "ready" | "no-content" | "error";

/** Source tag shown on the whole-document row; data, not interface copy. */
const WHOLE_TAG = "body";

export interface HtmlImportProps {
  onClose: () => void;
  onImport: (template: Template, name: string) => void;
}

function formatBytes(bytes: number): string {
  const formatter = new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 });
  if (bytes < 1024) return `${formatter.format(bytes)} B`;
  if (bytes < 1024 * 1024) return `${formatter.format(bytes / 1024)} KB`;
  return `${formatter.format(bytes / (1024 * 1024))} MB`;
}

function readFileText(file: File): Promise<string> {
  if (typeof file.text === "function") return file.text();
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.onerror = () => reject(reader.error ?? new Error("read-failed"));
    reader.readAsText(file);
  });
}

function depthStyle(depth: number): CSSProperties {
  return { "--hi-depth": Math.min(depth, 8) } as CSSProperties;
}

/**
 * HTML import dialog. The source document is parsed in memory and becomes a new
 * custom design draft; it is reference material, never evidence, and no remote
 * asset is fetched. The dialog owns the file picker, the region chooser and the
 * honest provenance note.
 */
export function HtmlImport({ onClose, onImport }: HtmlImportProps) {
  const { t } = useTranslation();
  const headingId = useId();
  const inputId = useId();
  const nameId = useId();
  const warnId = useId();
  const regionId = useId();
  const trapRef = useFocusTrap<HTMLDivElement>(true, onClose);
  const requestRef = useRef(0);

  const [phase, setPhase] = useState<Phase>("empty");
  const [fileName, setFileName] = useState<string | null>(null);
  const [fileSize, setFileSize] = useState<number | null>(null);
  const [parsed, setParsed] = useState<ParsedHtml | null>(null);
  const [selectedRegion, setSelectedRegion] = useState<string | null>(null);
  const [designName, setDesignName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState(false);

  // Ignore a parse that finishes after the dialog unmounts or a second file
  // replaces the first.
  useEffect(() => {
    return () => {
      requestRef.current += 1;
    };
  }, []);

  const explain = useCallback(
    (cause: unknown, name: string): string => {
      if (cause instanceof HtmlImportError) {
        if (cause.reason === "size") {
          return t("workbench.studio.htmlImportTooLarge", {
            defaultValue: "{{name}} is larger than 2 MiB. Choose a smaller HTML file.",
            name,
          });
        }
        if (cause.reason === "empty") {
          return t("workbench.studio.htmlImportEmpty", {
            defaultValue: "{{name}} is empty. Choose an HTML file with content.",
            name,
          });
        }
      }
      return t("workbench.studio.htmlImportParseFailed", {
        defaultValue: "{{name}} could not be read as HTML.",
        name,
      });
    },
    [t],
  );

  const chooseFile = useCallback(
    async (file: File | undefined) => {
      if (!file) return;
      requestRef.current += 1;
      const request = requestRef.current;
      setFileName(file.name);
      setFileSize(file.size);
      setParsed(null);
      setSelectedRegion(null);
      setDesignName("");
      setError(null);

      if (file.size > HTML_LIMITS.bytes) {
        setPhase("error");
        setError(
          t("workbench.studio.htmlImportTooLarge", {
            defaultValue: "{{name}} is larger than 2 MiB. Choose a smaller HTML file.",
            name: file.name,
          }),
        );
        return;
      }

      setPhase("parsing");
      try {
        const source = await readFileText(file);
        if (request !== requestRef.current) return;
        const result = parseHtml(source, file.name);
        if (request !== requestRef.current) return;
        setParsed(result);
        setDesignName(result.title);
        setPhase(result.layers > 0 ? "ready" : "no-content");
      } catch (cause) {
        if (request !== requestRef.current) return;
        setPhase("error");
        setError(explain(cause, file.name));
      }
    },
    [explain, t],
  );

  const onFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    void chooseFile(file);
    // Allow choosing the same file again after a refusal.
    event.target.value = "";
  };

  const onDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setDragActive(false);
    void chooseFile(event.dataTransfer.files[0]);
  };

  const commit = () => {
    if (!parsed || phase !== "ready") return;
    const name = designName.trim() || parsed.title;
    onImport(htmlTemplate(parsed, selectedRegion, name), name);
    onClose();
  };

  return (
    <div className="hi-layer">
      <div className="hi-backdrop" aria-hidden="true" onPointerDown={onClose} />
      <div
        ref={trapRef}
        className="hi-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={headingId}
        aria-busy={phase === "parsing"}
        data-testid="html-import"
      >
        <header className="hi-head">
          <h2 id={headingId} className="hi-title">
            {t("workbench.studio.htmlImportTitle", {
              defaultValue: "Import an HTML page",
            })}
          </h2>
          <button
            type="button"
            className="hi-close"
            aria-label={t("workbench.studio.htmlImportClose", {
              defaultValue: "Close HTML import",
            })}
            onClick={onClose}
            data-testid="html-import-close"
          >
            <X aria-hidden size={14} />
          </button>
        </header>

        <div className="hi-body">
          <p className="hi-intro">
            {t("workbench.studio.htmlImportIntro", {
              defaultValue:
                "Choose an HTML file. Only text, headings and layout are converted; nothing is fetched.",
            })}
          </p>

          {/* The file input is the accessible control; this box is a pointer-only drop target. */}
          {/* eslint-disable-next-line jsx-a11y/no-static-element-interactions */}
          <div
            className={dragActive ? "hi-drop is-active" : "hi-drop"}
            onDragOver={(event) => {
              event.preventDefault();
              setDragActive(true);
            }}
            onDragLeave={() => setDragActive(false)}
            onDrop={onDrop}
          >
            <label className="hi-field-label" htmlFor={inputId}>
              {t("workbench.studio.htmlImportFile", { defaultValue: "HTML file" })}
            </label>
            <input
              id={inputId}
              className="hi-file"
              type="file"
              accept=".html,.htm,text/html"
              onChange={onFileChange}
              data-testid="html-import-file"
            />
            <p className="hi-hint">
              {t("workbench.studio.htmlImportHint", {
                defaultValue:
                  "Drop an HTML file here, or choose one. Files over 2 MiB are refused.",
              })}
            </p>
          </div>

          {fileName !== null && fileSize !== null ? (
            <p className="hi-file-meta" data-testid="html-import-file-meta">
              {t("workbench.studio.htmlImportFileMeta", {
                defaultValue: "{{name}} · {{size}}",
                name: fileName,
                size: formatBytes(fileSize),
              })}
            </p>
          ) : null}

          {phase === "parsing" ? (
            <p className="hi-status" role="status" data-testid="html-import-parsing">
              {t("workbench.studio.htmlImportParsing", {
                defaultValue: "Reading {{name}}…",
                name: fileName ?? "",
              })}
            </p>
          ) : null}

          {phase === "error" && error ? (
            <p className="hi-error" role="alert" data-testid="html-import-error">
              {error}
            </p>
          ) : null}

          {phase === "no-content" ? (
            <p className="hi-status" role="status" data-testid="html-import-no-content">
              {t("workbench.studio.htmlImportNoContent", {
                defaultValue:
                  "This page has no importable text, headings or layout. Media-only pages produce nothing.",
              })}
            </p>
          ) : null}

          {parsed !== null && (phase === "ready" || phase === "no-content") ? (
            <>
              <div className="hi-summary">
                <span className="hi-doc-title" data-testid="html-import-title">
                  {parsed.title}
                </span>
                <span className="hi-counts" data-testid="html-import-counts">
                  {t("workbench.studio.htmlImportCounts", {
                    defaultValue: "{{layers}} layers · {{regions}} regions",
                    layers: parsed.layers,
                    regions: parsed.regions.length,
                  })}
                </span>
              </div>

              {parsed.warnings.length > 0 ? (
                <section className="hi-warnings" aria-labelledby={warnId}>
                  <h3 id={warnId} className="hi-warnings-title">
                    <TriangleAlert aria-hidden size={12} />
                    {t("workbench.studio.htmlImportWarnings", { defaultValue: "Warnings" })}
                  </h3>
                  <ul data-testid="html-import-warnings">
                    {parsed.warnings.map((warning) => (
                      <li key={warning}>{warning}</li>
                    ))}
                  </ul>
                </section>
              ) : null}

              <div className="hi-regions" role="group" aria-labelledby={regionId}>
                <h3 id={regionId} className="hi-regions-title">
                  {t("workbench.studio.htmlImportRegions", {
                    defaultValue: "Choose a region to import",
                  })}
                </h3>
                <div className="hi-region-list">
                  <button
                    type="button"
                    className="hi-region"
                    aria-pressed={selectedRegion === null}
                    onClick={() => setSelectedRegion(null)}
                    data-testid="html-import-region-whole"
                    style={depthStyle(0)}
                  >
                    <span className="hi-region-tag">{WHOLE_TAG}</span>
                    <span className="hi-region-label">
                      {t("workbench.studio.htmlImportWholeDocument", {
                        defaultValue: "Whole document",
                      })}
                    </span>
                    <span className="hi-region-count">{parsed.layers}</span>
                  </button>
                  {parsed.regions.map((region) => (
                    <button
                      key={region.id}
                      type="button"
                      className="hi-region"
                      aria-pressed={selectedRegion === region.id}
                      onClick={() => setSelectedRegion(region.id)}
                      data-testid={`html-import-region-${region.id}`}
                      style={depthStyle(region.depth)}
                    >
                      <span className="hi-region-tag">{region.tag}</span>
                      <span className="hi-region-label">
                        {region.label ||
                          t("workbench.studio.htmlImportNoText", { defaultValue: "No text" })}
                      </span>
                      <span className="hi-region-count">{region.layers}</span>
                    </button>
                  ))}
                </div>
              </div>

              <div className="hi-name-field">
                <label className="hi-field-label" htmlFor={nameId}>
                  {t("workbench.studio.htmlImportName", { defaultValue: "Design name" })}
                </label>
                <input
                  id={nameId}
                  className="hi-name"
                  value={designName}
                  onChange={(event) => setDesignName(event.target.value)}
                  data-testid="html-import-name"
                />
              </div>
            </>
          ) : null}

          <p className="hi-note" data-testid="html-import-note">
            {t("workbench.studio.htmlImportNote", {
              defaultValue:
                "Imported as a new design draft. Source HTML is reference material; styles and remote assets are not imported. RCL share reflects the layers this import maps to.",
            })}
          </p>
        </div>

        <footer className="hi-foot">
          <div className="hi-actions">
            <WbButton variant="ghost" size="sm" onClick={onClose} data-testid="html-import-cancel">
              {t("workbench.studio.htmlImportCancel", { defaultValue: "Cancel" })}
            </WbButton>
            <WbButton
              variant="primary"
              size="sm"
              onClick={commit}
              disabled={!parsed || phase !== "ready"}
              data-testid="html-import-import"
            >
              <FileUp aria-hidden size={13} />
              {t("workbench.studio.htmlImportAction", {
                defaultValue: "Import as a new design",
              })}
            </WbButton>
          </div>
        </footer>
      </div>
    </div>
  );
}
