import { useState } from "react";
import { CollectionPage } from "./CollectionPage";
export function Default() {
  return null;
}
export function MobileNavigation() {
  const [pane, setPane] = useState<"collection" | "inspector">("collection");
  const [selected, setSelected] = useState<number | null>(null);
  return (
    <CollectionPage
      mobilePane={pane}
      onMobilePaneChange={setPane}
      backLabel="Back to conversations"
      detailTitle={selected === null ? undefined : `Conversation ${selected + 1}`}
      regions={{
        header: <h1>Conversations</h1>,
        filters: (
          <label>
            Find a conversation
            <input aria-label="Find a conversation" defaultValue="Support" />
          </label>
        ),
        collection: (
          <div data-story-list style={{ height: 180, overflowY: "auto" }}>
            {Array.from({ length: 20 }, (_, i) => (
              <button
                type="button"
                key={i}
                data-story-row={i}
                style={{ display: "block", minHeight: 44, width: "100%" }}
                onClick={() => {
                  setSelected(i);
                  setPane("inspector");
                }}
              >
                Conversation {i + 1}
              </button>
            ))}
          </div>
        ),
        inspector:
          selected === null ? undefined : (
            <div>
              <p>Message history for the selected conversation.</p>
              <button type="button">Reply</button>
            </div>
          ),
      }}
    />
  );
}
