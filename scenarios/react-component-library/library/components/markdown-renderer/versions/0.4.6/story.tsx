import { MarkdownRenderer } from "./markdown-renderer";

// The caption identifies the specimen frame; assertions target renderer output.
export function MarkdownStory({ args }: { args: { content: string } }) {
  return (
    <figure>
      <figcaption>Markdown sample</figcaption>
      <div data-testid="markdown-story-output">
        <MarkdownRenderer content={args.content} />
      </div>
    </figure>
  );
}
