import { lazy, Suspense, useEffect, useState, type ComponentProps } from "react";

const SettingsModal = lazy(() => import("./SettingsModal").then(module => ({ default: module.SettingsModal })));

// Load settings at first use, then retain the mounted modal so its selected
// tab and unsaved form state survive closing and reopening as before.
export function DeferredSettingsModal(props: ComponentProps<typeof SettingsModal>) {
  const [opened, setOpened] = useState(props.isOpen);
  useEffect(() => { if (props.isOpen) setOpened(true); }, [props.isOpen]);
  if (!opened && !props.isOpen) return null;
  return <Suspense fallback={<div role="status" className="fixed bottom-4 right-4 z-50 rounded bg-slate-900 p-3 text-sm text-slate-300">Loading settings…</div>}>
    <SettingsModal {...props} />
  </Suspense>;
}
