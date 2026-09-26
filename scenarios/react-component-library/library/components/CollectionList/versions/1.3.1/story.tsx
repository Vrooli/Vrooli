import { useState } from "react";
import { CollectionList } from "./CollectionList";
const items = [
  {
    id: "support",
    title: "Support escalation",
    description: "Maya: Can you review the deployment results?",
    channel: "In-app \u00b7 Release assistant",
  },
  {
    id: "operations",
    title: "Operations handoff",
    description: "All services are healthy. The next check is scheduled.",
    channel: "Slack \u00b7 Operations assistant",
  },
  {
    id: "triage",
    title: "Incident triage",
    description: "An attachment is waiting for review.",
    channel: "Email \u00b7 Support assistant",
  },
];
export function Default() {
  return <CollectionList items={items} label="Conversations" />;
}
export function Interactive() {
  const [query, setQuery] = useState("");
  const [opened, setOpened] = useState("triage");
  return (
    <div>
      <label>
        Search conversations
        <input
          aria-label="Search conversations"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </label>
      <output aria-live="polite" data-opened>
        {opened ? `Opened ${opened}` : "Choose a conversation"}
      </output>
      <CollectionList
        items={items}
        fields={{
          key: "id",
          title: "title",
          description: "description",
          meta: "channel",
        }}
        label="Conversations"
        height={360}
        virtualize={{ estimateItemHeight: 120, overscan: 1 }}
        query={query}
        getSearchText={(item) => item.title}
        search={(item, query) => item.title.toLowerCase().includes(query.toLowerCase())}
        currentKey={opened}
        onOpen={(item) => setOpened(item.id)}
        bulkBar="none"
      />
    </div>
  );
}

export function KeyboardVirtualized() {
  const rows = Array.from({ length: 120 }, (_, index) => ({
    id: `record-${index}`,
    title: `Record ${index + 1}`,
    description:
      index % 3
        ? "Short summary"
        : "A longer summary verifies measured row navigation and stable identity.",
  }));
  return (
    <CollectionList
      items={rows}
      label="Keyboard collection"
      virtualize={{ estimateItemHeight: 100, overscan: 1 }}
      height={240}
      bulkBar="none"
    />
  );
}
