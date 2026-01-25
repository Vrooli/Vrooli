export type FileCategory = "code" | "markdown" | "image" | "pdf" | "binary" | "text";

export interface FileTypeInfo {
  category: FileCategory;
  mimeType?: string;
  canPreview: boolean;
}

const imageExtensions = new Set([".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".bmp", ".tiff"]);
const markdownExtensions = new Set([".md", ".mdx", ".markdown"]);

export function getFileTypeInfo(path: string): FileTypeInfo {
  const lastDot = path.lastIndexOf(".");
  if (lastDot === -1) {
    // No extension - treat as code
    return { category: "code", canPreview: false };
  }

  const ext = path.slice(lastDot).toLowerCase();

  if (markdownExtensions.has(ext)) {
    return { category: "markdown", mimeType: "text/markdown", canPreview: true };
  }
  if (imageExtensions.has(ext)) {
    const imageType = ext === ".jpg" ? "jpeg" : ext.slice(1);
    return { category: "image", mimeType: `image/${imageType}`, canPreview: true };
  }
  if (ext === ".pdf") {
    return { category: "pdf", mimeType: "application/pdf", canPreview: false };
  }
  // Default to code (syntax highlighting)
  return { category: "code", canPreview: false };
}

/**
 * Check if a file extension indicates a binary file
 */
export function isBinaryExtension(path: string): boolean {
  const lastDot = path.lastIndexOf(".");
  if (lastDot === -1) return false;

  const ext = path.slice(lastDot).toLowerCase();
  return imageExtensions.has(ext) || ext === ".pdf";
}
