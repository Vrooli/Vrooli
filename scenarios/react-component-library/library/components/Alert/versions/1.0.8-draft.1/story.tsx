import { Alert } from "./Alert";

// Preview contract exports for Alert 1.0.7.
// The declarative story contract owns expectations; this module supplies the
// typed specimen seam for versions whose composition is supplied by a harness.
export function Default() {
  return null;
}

/** Both supported description elements preserve the same announced message. */
export function DescriptionElements({
  args,
}: StoryHarnessProps<{ descriptionAs: "span" | "div" }>) {
  const otherElement = args.descriptionAs === "span" ? "div" : "span";
  return (
    <div>
      <Alert
        title="Workspace saved"
        description="Your changes are available to the team."
        descriptionAs={args.descriptionAs}
        tone="success"
      />
      <Alert
        title="Sync complete"
        description="The latest updates are ready."
        descriptionAs={otherElement}
      />
    </div>
  );
}
