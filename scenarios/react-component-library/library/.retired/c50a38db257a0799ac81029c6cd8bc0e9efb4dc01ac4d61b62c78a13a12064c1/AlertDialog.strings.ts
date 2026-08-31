import { defineStrings } from "@vrooli/react-component-library/useLocale/1.1.0";

export const AlertDialogStrings = defineStrings(
  "react-component-library:AlertDialog",
  {
    "overlays.alert-dialog.description":
      "This removes the workspace and its saved views from your account.",
    "overlays.alert-dialog.description.your-changes-are-still-safe-but-publishing-did-n":
      "Your changes are still safe, but publishing did not finish.",
    "overlays.alert-dialog.description.your-changes-will-become-visible-to-everyone-wit":
      "Your changes will become visible to everyone with access.",
    "overlays.alert-dialog.no-partial-publish-was-created":
      "No partial publish was created.",
    "overlays.alert-dialog.publishing-takes-a-moment-while-we-verify-the-final-version":
      "Publishing takes a moment while we verify the final version.",
    "overlays.alert-dialog.this-action-cannot-be-undone-export-anything-you-may-need-before-continuing":
      "This action cannot be undone. Export anything you may need before continuing.",
    "overlays.alert-dialog.title": "Remove this workspace?",
    "overlays.alert-dialog.title.publish-changes": "Publish changes?",
  },
);
