import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { ApprovalPrompt } from "@vrooli/react-component-library/ApprovalPrompt/1";
import { useState } from "react";
import { CollectionPage } from "./CollectionPage";
import { PromptComposer } from "@vrooli/react-component-library/PromptComposer/1";
export function Default() {
  return null;
}
export function DetailActions() {
  const [value, setValue] = useState("Please share the final rollout summary.");
  return (
    <CollectionPage
      gutter="none"
      detailTitle="Release review"
      regions={{
        header: <h1>Conversations</h1>,
        collection: <p>Release assistant · Updated just now</p>,
        inspector: <p>The rollout is healthy. Would you like the final summary?</p>,
        inspectorActions: (
          <PromptComposer value={value} onValueChange={setValue} onSend={async () => true} />
        ),
      }}
    />
  );
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

export function ContextualFlows() {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState("");
  return (
    <CollectionPage
      filterPlacement="collection"
      detailTitle="Selected conversation"
      regions={{
        header: (
          <PageHeader
            title="Conversations"
            actions={<Button onClick={() => setOpen(true)}>Start conversation</Button>}
          />
        ),
        collection: <p>Support escalation</p>,
        inspector: <p>The assistant needs your decision before continuing.</p>,
        inspectorNotice: (
          <ApprovalPrompt
            title="Allow attachment review?"
            action="Read attachment"
            target="Launch brief.pdf"
            scope="This conversation only"
            onApprove={async () => {}}
            onDeny={() => {}}
          />
        ),
        overlays: (
          <ResponsiveDialog
            open={open}
            onClose={() => setOpen(false)}
            title="Start a conversation"
            closeLabel="Close"
            footer={
              <Button disabled={!selected} onClick={() => setOpen(false)}>
                Start conversation
              </Button>
            }
          >
            <Select
              aria-label="Agent"
              value={selected}
              onChange={(event) => setSelected(event.currentTarget.value)}
              placeholder="Choose an agent"
              options={[
                { value: "release", label: "Release assistant" },
                { value: "support", label: "Support assistant" },
              ]}
            />
          </ResponsiveDialog>
        ),
      }}
    />
  );
}

export function PageGutters() {
  return (
    <CollectionPage
      gutter="page"
      regions={{
        header: <PageHeader title="Conversations" />,
        collection: <p>Choose a conversation.</p>,
      }}
    />
  );
}
