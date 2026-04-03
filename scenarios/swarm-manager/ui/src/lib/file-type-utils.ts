/**
 * File type detection, Monaco language mapping, and content type utilities.
 *
 * Extracted from file-preview.tsx to keep that component focused on rendering.
 */

import { FileCode, FileImage, FileText } from "lucide-react";
import { getFileExtension } from "./index";

export type FileType = "markdown" | "code" | "image" | "text";

/**
 * Determines the file type category from the file extension.
 */
export function getFileType(fileName: string): FileType {
  const ext = getFileExtension(fileName);

  if (["md", "markdown"].includes(ext)) {
    return "markdown";
  }

  if (["png", "jpg", "jpeg", "gif", "svg", "webp", "bmp", "ico"].includes(ext)) {
    return "image";
  }

  if ([
    "js", "jsx", "ts", "tsx", "json", "go", "py", "rs", "java", "c", "cpp", "h",
    "html", "css", "scss", "yaml", "yml", "toml", "xml", "sh", "bash", "zsh",
    "sql", "graphql", "proto", "dockerfile",
  ].includes(ext)) {
    return "code";
  }

  return "text";
}

/**
 * Returns the appropriate Lucide icon component for a file type.
 */
export function getFileTypeIcon(type: FileType): typeof FileText {
  switch (type) {
    case "markdown":
    case "text":
      return FileText;
    case "code":
      return FileCode;
    case "image":
      return FileImage;
  }
}

/**
 * Returns the icon CSS classes for a file type.
 */
export function getFileTypeIconClass(type: FileType): string {
  switch (type) {
    case "markdown":
    case "text":
      return "h-5 w-5 text-slate-400";
    case "code":
      return "h-5 w-5 text-cyan-400";
    case "image":
      return "h-5 w-5 text-purple-400";
  }
}

/**
 * Returns the Monaco language identifier for a file.
 */
export function getMonacoLanguage(fileName: string): string {
  const ext = getFileExtension(fileName);
  const languageMap: Record<string, string> = {
    js: "javascript",
    jsx: "javascript",
    ts: "typescript",
    tsx: "typescript",
    json: "json",
    md: "markdown",
    markdown: "markdown",
    yaml: "yaml",
    yml: "yaml",
    toml: "toml",
    html: "html",
    css: "css",
    scss: "scss",
    go: "go",
    py: "python",
    rs: "rust",
    java: "java",
    c: "c",
    cpp: "cpp",
    h: "cpp",
    sh: "shell",
    bash: "shell",
    zsh: "shell",
    sql: "sql",
    graphql: "graphql",
    proto: "protobuf",
    xml: "xml",
  };
  return languageMap[ext] ?? "plaintext";
}

/**
 * Maps file extensions to reasonable content types for saving.
 */
export function getContentTypeForFile(fileName: string): string {
  const ext = getFileExtension(fileName);
  switch (ext) {
    case "json":
      return "application/json";
    case "md":
    case "markdown":
      return "text/markdown";
    case "yaml":
    case "yml":
      return "text/yaml";
    case "toml":
      return "text/plain";
    case "html":
      return "text/html";
    case "css":
    case "scss":
      return "text/css";
    case "xml":
      return "text/xml";
    case "sql":
      return "text/plain";
    default:
      return "text/plain";
  }
}
