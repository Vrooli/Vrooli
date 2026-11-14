import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../ui/src/test-utils";
import { AlertDialog } from "../components/AlertDialog/versions/2.0.11/AlertDialog";
import { Dialog } from "../components/Dialog/versions/1.3.9/Dialog";
import { CommandPalette } from "../components/CommandPalette/versions/1.1.11/CommandPalette";

import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

const subjects: Array<{ name: string; selector: string; element: (open: boolean) => ReactElement }> = [
  { name: "FullPageDrawer", selector: "[data-rcl-full-page-drawer]", element: (open) => <FullPageDrawer open={open} title="Drawer" closeLabel="Close">Content</FullPageDrawer> },
  { name: "BottomSheet", selector: "[data-rcl-bottom-sheet]", element: (open) => <BottomSheet open={open} title="Sheet" closeLabel="Close">Content</BottomSheet> },
  { name: "ResponsiveDialog", selector: "[data-rcl-responsive-dialog]", element: (open) => <ResponsiveDialog open={open} title="Dialog" closeLabel="Close">Content</ResponsiveDialog> },
  { name: "AlertDialog", selector: "[data-rcl-alert-dialog-layer]", element: (open) => <AlertDialog open={open} title="Confirm" description="Continue?" onCancel={() => {}} onConfirm={() => {}} /> },
  { name: "Dialog", selector: "[data-rcl-dialog]", element: (open) => <Dialog open={open} title="Dialog" closeLabel="Close" onClose={() => {}}>Content</Dialog> },
  { name: "CommandPalette", selector: "[data-rcl-command-palette]", element: (open) => <CommandPalette open={open} /> },
];

for (const subject of subjects) {
  describe(`${subject.name} exit input ownership`, () => {
    it("keeps the exiting presentation inert and restores it when reopened", () => {
      const { rerender } = renderWithProviders(subject.element(true), { withoutRouter: true });
      expect(document.querySelector(subject.selector)).not.toHaveAttribute("inert");
      rerender(subject.element(false));
      const exiting = document.querySelector(subject.selector);
      expect(exiting).toBeInTheDocument();
      expect(exiting).toHaveAttribute("inert");
      rerender(subject.element(true));
      expect(document.querySelector(subject.selector)).not.toHaveAttribute("inert");
    });
  });
}
