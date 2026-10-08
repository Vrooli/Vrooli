import { computeLineDiff } from "../lib/diff";

export function DiffViewer({ before, after, filePath }: { before: string; after: string; filePath: string }) {
  const lines = computeLineDiff(before, after);

  return (
    <div className="rounded-lg border border-white/10 bg-black/20 overflow-hidden">
      <div className="px-3 py-1.5 border-b border-white/5 bg-white/5">
        <span className="text-xs font-mono text-slate-400">{filePath}</span>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-xs font-mono">
          <tbody>
            {lines.map((line, i) => (
              <tr
                key={i}
                className={
                  line.type === "added"
                    ? "bg-green-500/10"
                    : line.type === "removed"
                    ? "bg-red-500/10"
                    : ""
                }
              >
                <td className="w-10 select-none px-2 text-right text-slate-600 align-top">
                  {line.oldLineNo ?? ""}
                </td>
                <td className="w-10 select-none px-2 text-right text-slate-600 align-top">
                  {line.newLineNo ?? ""}
                </td>
                <td className="w-4 select-none text-center align-top">
                  <span className={
                    line.type === "added"
                      ? "text-green-400"
                      : line.type === "removed"
                      ? "text-red-400"
                      : "text-slate-600"
                  }>
                    {line.type === "added" ? "+" : line.type === "removed" ? "\u2212" : " "}
                  </span>
                </td>
                <td className="whitespace-pre px-2 text-slate-200 align-top">{line.content}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
