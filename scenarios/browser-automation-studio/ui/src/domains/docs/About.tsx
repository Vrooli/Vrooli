import { MarkdownRenderer } from "./MarkdownRenderer";
import { ABOUT_CONTENT } from "./content/about";

export function About() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6">
        <MarkdownRenderer content={ABOUT_CONTENT} />
      </div>
    </div>
  );
}
