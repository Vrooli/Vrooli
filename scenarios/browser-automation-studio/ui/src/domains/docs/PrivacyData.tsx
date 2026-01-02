import { MarkdownRenderer } from "./MarkdownRenderer";
import { PRIVACY_DATA_CONTENT } from "./content/privacyData";

export function PrivacyData() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6">
        <MarkdownRenderer content={PRIVACY_DATA_CONTENT} />
      </div>
    </div>
  );
}
