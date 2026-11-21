import { useCallback, useState } from "react";

import { useFileViewerHost } from "../i18n/context.js";
import { clampReaderFontSize, READER_FONT_DEFAULT } from "../lib/typography.js";

/**
 * The Markdown reader's text size: the host's persisted preference when the
 * provider supplies one, otherwise internal state that lasts as long as the
 * reader is mounted.
 */
export function useReaderFontSize(): [size: number, setSize: (size: number) => void] {
  const { readerFontSize } = useFileViewerHost();
  const [local, setLocal] = useState(READER_FONT_DEFAULT);
  const hostOnChange = readerFontSize?.onChange;
  const setSize = useCallback((size: number) => {
    const next = clampReaderFontSize(size);
    if (hostOnChange) hostOnChange(next); else setLocal(next);
  }, [hostOnChange]);
  return [clampReaderFontSize(readerFontSize?.value ?? local), setSize];
}
