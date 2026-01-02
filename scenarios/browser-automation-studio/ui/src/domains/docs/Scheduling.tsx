import { MarkdownRenderer } from "./MarkdownRenderer";
import { SCHEDULING_CONTENT } from "./content/scheduling";

export function Scheduling() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6">
        <MarkdownRenderer content={SCHEDULING_CONTENT} />
      </div>
    </div>
  );
}
