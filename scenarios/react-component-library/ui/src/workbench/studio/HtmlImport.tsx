import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type DragEvent,
} from "react";
import { FileUp, X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { WbButton } from "../primitives";
import { useFocusTrap } from "../useFocusTrap";
import { HtmlImportRegions } from "./HtmlImportRegions";
import { HtmlImportOverview } from "./HtmlImportOverview";
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
  const bundleInputId = useId();
  const nameId = useId();
  const trapRef = useFocusTrap<HTMLDivElement>(true, onClose);
  const requestRef = useRef(0);

  const [phase, setPhase] = useState<Phase>("empty");
  const [fileName, setFileName] = useState<string | null>(null);
  const [fileSize, setFileSize] = useState<number | null>(null);
  const [parsed, setParsed] = useState<ParsedHtml | null>(null);
  const [stylesheets, setStylesheets] = useState<string[]>([]);
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
            defaultValue:
              "{{name}} is larger than 2 MiB after including its local styles. Choose a smaller source bundle.",
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

  const chooseFiles = useCallback(
    async (selected: FileList | File[]) => {
      const files = Array.from(selected);
      requestRef.current += 1;
      const request = requestRef.current;
      const htmlFile = files
        .filter((file) => /\.html?$/i.test(file.name))
        .sort(
          (left, right) =>
            Number(!/index\.html?$/i.test(left.name)) - Number(!/index\.html?$/i.test(right.name)),
        )[0];
      if (!htmlFile) {
        setParsed(null);
        setFileName(null);
        setFileSize(null);
        setStylesheets([]);
        setSelectedRegion(null);
        setPhase("error");
        setError(
          t("workbench.studio.htmlImportMissingHtml", {
            defaultValue: "Choose a folder or selection that includes an HTML file.",
          }),
        );
        return;
      }
      const cssFiles = files.filter((file) => /\.css$/i.test(file.name));
      const supported = [htmlFile, ...cssFiles];
      const sourceBytes = supported.reduce((total, file) => total + file.size, 0);
      setFileName(htmlFile.name);
      setFileSize(sourceBytes);
      setParsed(null);
      setStylesheets([]);
      setSelectedRegion(null);
      setDesignName("");
      setError(null);

      if (sourceBytes > HTML_LIMITS.bytes) {
        setPhase("error");
        setError(
          t("workbench.studio.htmlImportTooLarge", {
            defaultValue:
              "{{name}} is larger than 2 MiB after including its local styles. Choose a smaller source bundle.",
            name: htmlFile.name,
          }),
        );
        return;
      }

      setPhase("parsing");
      try {
        const [source, css] = await Promise.all([
          readFileText(htmlFile),
          Promise.all(cssFiles.map(readFileText)),
        ]);
        if (request !== requestRef.current) return;
        const result = parseHtml(source, htmlFile.name, css);
        if (request !== requestRef.current) return;
        setParsed(result);
        setStylesheets(css);
        setDesignName(result.title);
        setPhase(result.layers > 0 ? "ready" : "no-content");
      } catch (cause) {
        if (request !== requestRef.current) return;
        setPhase("error");
        setError(explain(cause, htmlFile.name));
      }
    },
    [explain, t],
  );

  const onFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (files?.length) void chooseFiles(files);
    // Allow choosing the same file again after a refusal.
    event.target.value = "";
  };

  const onDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setDragActive(false);
    void chooseFiles(event.dataTransfer.files);
  };

  const commit = () => {
    if (!parsed || phase !== "ready") return;
    const name = designName.trim() || parsed.title;
    const template = htmlTemplate(
      parsed,
      selectedRegion,
      name,
      parsed.sourcePreview(selectedRegion, stylesheets),
    );
    onImport(template, name);
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
                "Choose an HTML file with its local CSS, or a folder. The editable layers come from structure; a read-only source view keeps the original styling.",
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
              accept=".html,.htm,.css,text/html,text/css"
              multiple
              onChange={onFileChange}
              data-testid="html-import-file"
            />
            <label className="hi-field-label" htmlFor={bundleInputId}>
              {t("workbench.studio.htmlImportBundle", {
                defaultValue: "Or choose a source folder",
              })}
            </label>
            <input
              id={bundleInputId}
              className="hi-file"
              type="file"
              multiple
              // Chromium exposes directory selection; other browsers keep the multi-file picker.
              {...({ webkitdirectory: "", directory: "" } as Record<string, string>)}
              accept=".html,.htm,.css,text/html,text/css"
              onChange={onFileChange}
              data-testid="html-import-bundle"
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
              <HtmlImportOverview parsed={parsed} />

              <HtmlImportRegions
                regions={parsed.regions}
                layers={parsed.layers}
                selectedRegion={selectedRegion}
                onSelectRegion={setSelectedRegion}
              />

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
