import { MarkdownRenderer } from "./MarkdownRenderer";
import { SESSIONS_CONTENT } from "./content/sessions";

export function Sessions() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6">
        <MarkdownRenderer content={SESSIONS_CONTENT} />
      </div>
    </div>
  );
}
