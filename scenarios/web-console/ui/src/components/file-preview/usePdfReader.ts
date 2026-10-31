import { useEffect, useRef, useState } from "react";
import type { PdfSession } from "./pdfRuntime";

export function usePdfReader(href: string) {
  const containerRef = useRef<HTMLDivElement>(null);
  const pagesRef = useRef<HTMLDivElement>(null);
  const session = useRef<PdfSession | null>(null);
  const [pages, setPages] = useState(0);
  const [page, setPage] = useState(1);
  const [scale, setScale] = useState(1);
  const [failed, setFailed] = useState(false);
  const [passwordNeeded, setPasswordNeeded] = useState(false);
  const [incorrectPassword, setIncorrectPassword] = useState(false);
  const [matches, setMatches] = useState({ current: 0, total: 0, pending: false });
  useEffect(() => {
    const container = containerRef.current;
    const element = pagesRef.current;
    if (!container || !element) return;
    let disposed = false;
    void import("./pdfRuntime").then(({ createPdfSession }) => {
      if (disposed) return;
      return createPdfSession(container, element, href, {
        ready: (count) => { if (!disposed) { setPages(count); setPasswordNeeded(false); } },
        page: (value) => { if (!disposed) setPage(value); },
        scale: (value) => { if (!disposed) setScale(value); },
        matches: (current, total, pending) => { if (!disposed) setMatches({ current, total, pending }); },
        password: (incorrect) => { if (!disposed) { setPasswordNeeded(true); setIncorrectPassword(incorrect); } },
        error: () => { if (!disposed) setFailed(true); },
      });
    }).then((reader) => {
      if (!reader) return;
      if (disposed) reader.destroy();
      else session.current = reader;
    }).catch(() => { if (!disposed) setFailed(true); });
    return () => { disposed = true; session.current?.destroy(); session.current = null; };
  }, [href]);
  return {
    containerRef, pagesRef, pages, page, scale, failed, passwordNeeded, incorrectPassword, matches,
    goTo: (value: number) => { session.current?.page(value); },
    zoom: (value: string) => { session.current?.zoom(value); },
    find: (query: string, previous = false, again = false) => { session.current?.find(query, previous, again); },
    unlock: (password: string) => { session.current?.password(password); },
  };
}
