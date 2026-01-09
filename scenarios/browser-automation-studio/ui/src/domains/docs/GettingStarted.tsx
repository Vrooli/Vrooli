import { MarkdownRenderer } from "./MarkdownRenderer";
import { GETTING_STARTED_CONTENT } from "./content/gettingStarted";

export function GettingStarted() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6">
        <MarkdownRenderer content={GETTING_STARTED_CONTENT} />
      </div>
    </div>
  );
}
