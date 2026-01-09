/**
 * AssetPreviewPane Component
 *
 * Preview pane for viewing non-workflow asset files (images, text, JSON, etc.)
 * when selecting from the file tree.
 */

import { useState, useEffect, lazy, Suspense, useMemo } from "react";
import {
  X,
  FileText,
  Image as ImageIcon,
  FileJson,
  File,
  Download,
  Loader,
  AlertCircle,
} from "lucide-react";
import { getConfig } from "@/config";
import { logger } from "@utils/logger";

// Lazy load Monaco editor to avoid large initial bundle
const Editor = lazy(() => import("@monaco-editor/react"));

// ============================================================================
// Types
// ============================================================================

interface AssetPreviewPaneProps {
  path: string;
  projectId: string;
  onClose: () => void;
}

type FileType = "image" | "text" | "json" | "unknown";

interface FileInfo {
  type: FileType;
  extension: string;
  mimeType: string;
  language?: string; // Monaco language for syntax highlighting
}

// ============================================================================
// File Type Detection
// ============================================================================

const IMAGE_EXTENSIONS = new Set(["png", "jpg", "jpeg", "gif", "svg", "webp", "ico", "bmp"]);
const TEXT_EXTENSIONS: Record<string, { mimeType: string; language: string }> = {
  txt: { mimeType: "text/plain", language: "plaintext" },
  md: { mimeType: "text/markdown", language: "markdown" },
  html: { mimeType: "text/html", language: "html" },
  css: { mimeType: "text/css", language: "css" },
  js: { mimeType: "text/javascript", language: "javascript" },
  ts: { mimeType: "text/typescript", language: "typescript" },
  jsx: { mimeType: "text/javascript", language: "javascript" },
  tsx: { mimeType: "text/typescript", language: "typescript" },
  yaml: { mimeType: "text/yaml", language: "yaml" },
  yml: { mimeType: "text/yaml", language: "yaml" },
  xml: { mimeType: "text/xml", language: "xml" },
  sh: { mimeType: "text/x-shellscript", language: "shell" },
  py: { mimeType: "text/x-python", language: "python" },
  go: { mimeType: "text/x-go", language: "go" },
  rs: { mimeType: "text/x-rust", language: "rust" },
  sql: { mimeType: "text/x-sql", language: "sql" },
};

function getFileInfo(path: string): FileInfo {
  const filename = path.split("/").pop() ?? path;
  const ext = filename.includes(".") ? filename.split(".").pop()?.toLowerCase() ?? "" : "";

  // Check for JSON
  if (ext === "json") {
    return { type: "json", extension: ext, mimeType: "application/json", language: "json" };
  }

  // Check for images
  if (IMAGE_EXTENSIONS.has(ext)) {
    const mimeMap: Record<string, string> = {
      png: "image/png",
      jpg: "image/jpeg",
      jpeg: "image/jpeg",
      gif: "image/gif",
      svg: "image/svg+xml",
      webp: "image/webp",
      ico: "image/x-icon",
      bmp: "image/bmp",
    };
    return { type: "image", extension: ext, mimeType: mimeMap[ext] ?? "image/png" };
  }

  // Check for text files
  const textInfo = TEXT_EXTENSIONS[ext];
  if (textInfo) {
    return { type: "text", extension: ext, ...textInfo };
  }

  // Unknown file type
  return { type: "unknown", extension: ext, mimeType: "application/octet-stream" };
}

function getFileIcon(fileType: FileType) {
  switch (fileType) {
    case "image":
      return <ImageIcon size={18} className="text-purple-400" />;
    case "json":
      return <FileJson size={18} className="text-yellow-400" />;
    case "text":
      return <FileText size={18} className="text-blue-400" />;
    default:
      return <File size={18} className="text-gray-400" />;
  }
}

// ============================================================================
// Image Preview Component
// ============================================================================

interface ImagePreviewProps {
  src: string;
  alt: string;
}

function ImagePreview({ src, alt }: ImagePreviewProps) {
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  return (
    <div className="flex-1 flex items-center justify-center p-4 bg-gray-900/50">
      {isLoading && !error && (
        <div className="absolute inset-0 flex items-center justify-center">
          <Loader size={24} className="animate-spin text-gray-400" />
        </div>
      )}
      {error ? (
        <div className="text-center text-gray-400">
          <AlertCircle size={48} className="mx-auto mb-2 opacity-50" />
          <p>{error}</p>
        </div>
      ) : (
        <img
          src={src}
          alt={alt}
          className="max-w-full max-h-full object-contain"
          onLoad={() => setIsLoading(false)}
          onError={() => {
            setIsLoading(false);
            setError("Failed to load image");
          }}
          style={{ display: isLoading ? "none" : "block" }}
        />
      )}
    </div>
  );
}

// ============================================================================
// Text Preview Component
// ============================================================================

interface TextPreviewProps {
  content: string;
  language: string;
}

function TextPreview({ content, language }: TextPreviewProps) {
  return (
    <div className="flex-1 overflow-hidden">
      <Suspense
        fallback={
          <div className="flex items-center justify-center h-full">
            <Loader size={24} className="animate-spin text-gray-400" />
          </div>
        }
      >
        <Editor
          height="100%"
          defaultLanguage={language}
          value={content}
          theme="vs-dark"
          options={{
            readOnly: true,
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: "on",
            scrollBeyondLastLine: false,
            automaticLayout: true,
            tabSize: 2,
            wordWrap: "on",
            folding: true,
            scrollbar: {
              verticalScrollbarSize: 8,
              horizontalScrollbarSize: 8,
            },
          }}
        />
      </Suspense>
    </div>
  );
}

// ============================================================================
// Unknown File Preview Component
// ============================================================================

interface UnknownPreviewProps {
  filename: string;
  extension: string;
  downloadUrl: string;
}

function UnknownPreview({ filename, extension, downloadUrl }: UnknownPreviewProps) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center p-6 text-center">
      <File size={64} className="text-gray-500 mb-4" />
      <h4 className="text-lg font-medium text-gray-300 mb-2">{filename}</h4>
      <p className="text-sm text-gray-500 mb-4">
        {extension ? `.${extension.toUpperCase()} file` : "Unknown file type"} - Preview not available
      </p>
      <a
        href={downloadUrl}
        download={filename}
        className="inline-flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
      >
        <Download size={16} />
        Download File
      </a>
    </div>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export function AssetPreviewPane({
  path,
  projectId,
  onClose,
}: AssetPreviewPaneProps) {
  const [content, setContent] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [apiUrl, setApiUrl] = useState<string | null>(null);

  // Parse file info from path
  const filename = path.split("/").pop() ?? path;
  const fileInfo = useMemo(() => getFileInfo(path), [path]);

  // Get API URL on mount
  useEffect(() => {
    getConfig().then((config) => {
      setApiUrl(config.API_URL);
    });
  }, []);

  // Build asset URL
  const assetUrl = apiUrl ? `${apiUrl}/projects/${projectId}/files/${encodeURIComponent(path)}` : null;

  // Load text content for text/json files
  useEffect(() => {
    if (!assetUrl || fileInfo.type === "image" || fileInfo.type === "unknown") {
      return;
    }

    const loadContent = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const response = await fetch(assetUrl);
        if (!response.ok) {
          throw new Error(`Failed to load file: ${response.status}`);
        }
        const text = await response.text();
        setContent(text);
      } catch (err) {
        logger.error(
          "Failed to load asset content",
          { component: "AssetPreviewPane", path, projectId },
          err
        );
        setError(err instanceof Error ? err.message : "Failed to load file");
      } finally {
        setIsLoading(false);
      }
    };

    void loadContent();
  }, [assetUrl, fileInfo.type, path, projectId]);

  return (
    <div className="bg-flow-node border border-gray-700 rounded-lg h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-gray-700 flex items-center justify-between gap-4">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center bg-gradient-to-br from-gray-500/20 to-gray-500/10 border border-gray-500/20">
            {getFileIcon(fileInfo.type)}
          </div>
          <div className="min-w-0">
            <h3 className="text-lg font-semibold text-surface truncate">{filename}</h3>
            <span className="text-xs text-gray-500">
              {fileInfo.extension ? `.${fileInfo.extension}` : "File"}
            </span>
          </div>
        </div>
        <button
          onClick={onClose}
          className="p-1.5 text-gray-500 hover:text-gray-300 hover:bg-gray-700 rounded-lg transition-colors flex-shrink-0"
          title="Close preview"
        >
          <X size={18} />
        </button>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex-1 flex items-center justify-center">
          <Loader size={24} className="animate-spin text-gray-400" />
          <span className="ml-2 text-gray-400">Loading...</span>
        </div>
      ) : error ? (
        <div className="flex-1 flex flex-col items-center justify-center p-6 text-center">
          <AlertCircle size={48} className="text-red-400 mb-4" />
          <p className="text-gray-300 mb-2">Failed to load file</p>
          <p className="text-sm text-gray-500">{error}</p>
        </div>
      ) : fileInfo.type === "image" && assetUrl ? (
        <ImagePreview src={assetUrl} alt={filename} />
      ) : (fileInfo.type === "text" || fileInfo.type === "json") && content !== null ? (
        <TextPreview content={content} language={fileInfo.language ?? "plaintext"} />
      ) : assetUrl ? (
        <UnknownPreview filename={filename} extension={fileInfo.extension} downloadUrl={assetUrl} />
      ) : (
        <div className="flex-1 flex items-center justify-center">
          <Loader size={24} className="animate-spin text-gray-400" />
        </div>
      )}
    </div>
  );
}

export default AssetPreviewPane;
