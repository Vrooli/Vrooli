export type FileCategory = "code" | "markdown" | "image" | "pdf" | "binary" | "text";

/**
 * How the API delivers a file's bytes in `full_content`.
 * - "base64": raster images, which cannot survive a JSON string as-is.
 * - "text": everything else, including SVG (it is XML).
 * The API decides this by extension too; the two lists must agree, or a preview
 * will decode the wrong way. See `binaryImageExtensions` in api/diff_helpers.go.
 */
export type ContentEncoding = "base64" | "text";

export interface FileTypeInfo {
  category: FileCategory;
  mimeType?: string;
  canPreview: boolean;
  encoding: ContentEncoding;
}

/**
 * Extension to MIME type. Spelled out rather than derived from the extension:
 * `image/${ext}` silently produces invalid types like `image/svg` (the real one
 * is `image/svg+xml`) and `image/ico` (`image/x-icon`), which browsers refuse to
 * decode in a data URL, leaving a broken-image placeholder.
 */
const imageMimeTypes = new Map<string, string>([
  [".png", "image/png"],
  [".jpg", "image/jpeg"],
  [".jpeg", "image/jpeg"],
  [".gif", "image/gif"],
  [".svg", "image/svg+xml"],
  [".webp", "image/webp"],
  [".ico", "image/x-icon"],
  [".bmp", "image/bmp"],
  [".tiff", "image/tiff"]
]);

/** SVG is an image to render but text to transport, diff, and count lines for. */
const textImageExtensions = new Set([".svg"]);

const markdownExtensions = new Set([".md", ".mdx", ".markdown"]);

export function getFileTypeInfo(path: string): FileTypeInfo {
  const lastDot = path.lastIndexOf(".");
  if (lastDot === -1) {
    // No extension - treat as code
    return { category: "code", canPreview: false, encoding: "text" };
  }

  const ext = path.slice(lastDot).toLowerCase();

  if (markdownExtensions.has(ext)) {
    return { category: "markdown", mimeType: "text/markdown", canPreview: true, encoding: "text" };
  }
  const imageMimeType = imageMimeTypes.get(ext);
  if (imageMimeType) {
    return {
      category: "image",
      mimeType: imageMimeType,
      canPreview: true,
      encoding: textImageExtensions.has(ext) ? "text" : "base64"
    };
  }
  if (ext === ".pdf") {
    return { category: "pdf", mimeType: "application/pdf", canPreview: false, encoding: "base64" };
  }
  // Default to code (syntax highlighting)
  return { category: "code", canPreview: false, encoding: "text" };
}

/**
 * Build an `<img src>` for previewable image content returned by the diff API.
 * Returns null when the file type cannot be rendered as an image.
 *
 * Text-encoded images (SVG) are percent-encoded rather than base64-encoded:
 * btoa() throws on any character above U+00FF, which a diagram with a non-ASCII
 * label would contain. Rendering via `<img>` keeps SVG inert — scripts and
 * external references in the file cannot run in an image context.
 */
export function buildImagePreviewSrc(fileType: FileTypeInfo, content: string): string | null {
  if (fileType.category !== "image" || !fileType.mimeType) return null;
  if (fileType.encoding === "text") {
    return `data:${fileType.mimeType};charset=utf-8,${encodeURIComponent(content)}`;
  }
  return `data:${fileType.mimeType};base64,${content}`;
}

/**
 * Check if a file extension indicates a binary file.
 * SVG is excluded: it is text, and is diffed and displayed as such.
 */
export function isBinaryExtension(path: string): boolean {
  const lastDot = path.lastIndexOf(".");
  if (lastDot === -1) return false;

  const ext = path.slice(lastDot).toLowerCase();
  if (textImageExtensions.has(ext)) return false;
  return imageMimeTypes.has(ext) || ext === ".pdf";
}
