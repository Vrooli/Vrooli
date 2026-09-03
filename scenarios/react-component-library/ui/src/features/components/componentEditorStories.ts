import type { ComponentStory } from "../../api/components";
import type { PreviewSpecimen, SpecimenIdentity } from "./ComponentEditorStage";

export function specimenIdentity(
  example?: Pick<PreviewSpecimen, "version" | "name">,
): SpecimenIdentity {
  return `${example?.version || "__current__"}:${example?.name || "__default__"}`;
}

export function parseStorySpecimens(stories: ComponentStory[]): PreviewSpecimen[] {
  return stories.flatMap((contract) => {
    try {
      const definitions = JSON.parse(contract.storiesJson) as Array<{
        id?: unknown;
        name?: unknown;
        description?: unknown;
        args?: unknown;
        environment?: unknown;
        expect?: unknown;
      }>;
      if (!Array.isArray(definitions)) return [];
      return definitions.flatMap((definition) => {
        if (
          typeof definition.id !== "string" ||
          typeof definition.name !== "string" ||
          !definition.args ||
          typeof definition.args !== "object" ||
          Array.isArray(definition.args)
        )
          return [];
        const environment: Record<string, string> =
          definition.environment &&
          typeof definition.environment === "object" &&
          !Array.isArray(definition.environment)
            ? (Object.fromEntries(
                Object.entries(definition.environment as Record<string, unknown>).filter(
                  ([, value]) => typeof value === "string",
                ),
              ) as Record<string, string>)
            : {};
        return [
          {
            id: `${contract.id}:${definition.id}`,
            componentId: contract.componentId,
            libraryId: contract.libraryId,
            version: contract.version,
            name: definition.id,
            displayName: definition.name,
            description:
              typeof definition.description === "string" && definition.description.trim()
                ? definition.description
                : undefined,
            propsJson: JSON.stringify(definition.args),
            environment,
            expectJson: JSON.stringify(Array.isArray(definition.expect) ? definition.expect : []),
            sourcePath: contract.sourcePath,
            storyId: definition.id,
          },
        ];
      });
    } catch {
      return [];
    }
  });
}
