import { MarkdownRenderer } from "./MarkdownRenderer";
import { REPLAY_EXPORT_CONTENT } from "./content/replayExport";

export function ReplayExport() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6">
        <MarkdownRenderer content={REPLAY_EXPORT_CONTENT} />
      </div>
    </div>
  );
}
