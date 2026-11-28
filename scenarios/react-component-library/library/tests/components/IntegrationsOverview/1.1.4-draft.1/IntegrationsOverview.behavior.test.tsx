import { describe, expect, it, vi } from "vitest";

import {
  integrationGroups,
  type IntegrationDependency,
} from "../../../../components/IntegrationsOverview/versions/1.1.4-draft.1/IntegrationsOverview";

describe("IntegrationsOverview dependency warnings", () => {
  it("shows one provider warning while retaining every capability row", () => {
    const dependency: IntegrationDependency = {
      id: "provider-ai",
      name: "AI provider",
      description: "Provider health",
      dependencyKind: "service",
      dependencySlug: "shared-provider",
      uses: [
        { id: "transcription", name: "Transcription", description: "Transcribe audio", group: "media" },
        { id: "summarization", name: "Summarization", description: "Summarize text", group: "media" },
      ],
      features: [],
      status: "unavailable",
      message: "Provider is offline",
    };

    const decorate = vi.fn(() => ({
      problem: { title: "Provider is offline", description: "Check the provider once." },
      action: { label: "Retry provider", onRun: () => undefined },
    }));
    const groups = integrationGroups([dependency], "capability", decorate);
    const items = groups.flatMap((group) => group.items);

    expect(items.filter((item) => item.problem)).toHaveLength(1);
    expect(items.map((item) => item.name)).toEqual(["Transcription", "Summarization"]);
    expect(items.map((item) => item.subtitle)).toEqual(["Transcribe audio", "Summarize text"]);
    expect(decorate).toHaveBeenCalledTimes(2);
  });
});
