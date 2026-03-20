// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
// [REQ:P1-002] One-tap export of thought graphs
import { useState, useEffect, useRef } from "react";
import { useMutation } from "@tanstack/react-query";
import { Download, Check, AlertTriangle } from "lucide-react";
import { exportScheme } from "../lib/api";
import { downloadJSON, slugify } from "../lib/download";

interface Props {
  schemeId: string;
  schemeName: string;
}

export function ExportButton({ schemeId, schemeName }: Props) {
  const [exported, setExported] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current);
    };
  }, []);

  const exportMut = useMutation({
    mutationFn: () => exportScheme(schemeId),
    onSuccess: (data) => {
      downloadJSON(data, `${slugify(schemeName)}-export.json`);
      setExported(true);
      if (timerRef.current !== null) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => setExported(false), 2000);
    },
  });

  const icon = exportMut.isError ? (
    <AlertTriangle className="h-4 w-4 text-red-400" />
  ) : exported ? (
    <Check className="h-4 w-4 text-green-400" />
  ) : (
    <Download className="h-4 w-4" />
  );

  return (
    <button
      data-testid="export-btn"
      onClick={() => {
        exportMut.reset();
        exportMut.mutate();
      }}
      disabled={exportMut.isPending}
      className="p-1.5 rounded text-slate-400 hover:text-white hover:bg-white/10 disabled:opacity-50"
      aria-label={exportMut.isError ? "Export failed — click to retry" : "Export scheme"}
    >
      {icon}
    </button>
  );
}
