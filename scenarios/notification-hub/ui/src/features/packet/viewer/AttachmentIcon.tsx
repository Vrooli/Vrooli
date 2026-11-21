import {
  Box,
  Database,
  File as FileIcon,
  FileArchive,
  FileAudio,
  FileCode,
  FileSpreadsheet,
  FileText,
  FileType,
  Film,
  Folder,
  Image as ImageIcon,
  Presentation,
  type LucideProps,
} from "lucide-react";

const ICONS: Record<string, typeof FileIcon> = {
  markdown: FileText,
  text: FileText,
  document: FileText,
  ebook: FileText,
  pdf: FileText,
  code: FileCode,
  diff: FileCode,
  image: ImageIcon,
  svg: ImageIcon,
  video: Film,
  audio: FileAudio,
  model3d: Box,
  directory: Folder,
  archive: FileArchive,
  database: Database,
  spreadsheet: FileSpreadsheet,
  csv: FileSpreadsheet,
  presentation: Presentation,
  font: FileType,
};

/** The glyph for a preview kind (the server's classification), with a plain file as the fallback. */
export function AttachmentIcon({ kind, folder = false, ...props }: { kind: string; folder?: boolean } & LucideProps) {
  const Icon = folder ? Folder : ICONS[kind] ?? FileIcon;
  return <Icon aria-hidden="true" {...props} />;
}
