import { useState } from "react";
import { MessageList, type MessageListProps, type MessageListItem } from "./MessageList";
type Props = { args: Record<string, unknown> };
export function MessageListStory({ args }: Props) {
  const props = args as unknown as MessageListProps;
  const [retrying, setRetrying] = useState(false);
  return (
    <MessageList
      {...props}
      state={retrying ? "retry" : props.state}
      onRetry={props.onRetry ? () => setRetrying(true) : undefined}
    />
  );
}
const initial: MessageListItem[] = Array.from({ length: 24 }, (_, index) => ({
  id: `message-${index}`,
  actor: { name: index % 2 ? "Assistant" : "Operator" },
  content: `Message ${index + 1}: deployment checks and operational context remain available.`,
  group: "Today",
}));
export function LiveHistory() {
  const [messages, setMessages] = useState(initial);
  const [branch, setBranch] = useState("main");
  return (
    <div style={{ width: "min(100%, 42rem)" }}>
      <button
        type="button"
        style={{ minHeight: 44 }}
        onClick={() =>
          setMessages((rows) => [
            ...rows,
            {
              id: `new-${rows.length}`,
              actor: { name: "Assistant" },
              content: "A new response arrived.",
              group: "Today",
            },
          ])
        }
      >
        Append response
      </button>
      <button
        type="button"
        style={{ minHeight: 44 }}
        onClick={() =>
          setMessages((rows) =>
            rows.map((row, index) =>
              index === rows.length - 1
                ? {
                    ...row,
                    content: String(row.content) + " Additional streaming detail.",
                  }
                : row,
            ),
          )
        }
      >
        Stream response
      </button>
      <MessageList
        messages={messages}
        height={360}
        hasEarlier
        onLoadEarlier={() =>
          setMessages((rows) => [
            {
              id: `older-${rows.length}`,
              actor: { name: "Operator" },
              content: "Earlier deployment context.",
              group: "Yesterday",
            },
            ...rows,
          ])
        }
        branches={[
          { id: "main", label: "Main discussion" },
          { id: "alternative", label: "Alternative reply" },
        ]}
        branchId={branch}
        onBranchChange={(id) => {
          setBranch(id);
          setMessages(
            id === "main"
              ? initial
              : [
                  {
                    id: "alternate",
                    actor: { name: "Assistant" },
                    content: "An alternative explanation.",
                  },
                ],
          );
        }}
      />
    </div>
  );
}
