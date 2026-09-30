import { useEffect } from "react";
import {
  ToastManagerProvider,
  useToastActions,
  type ToastActions,
} from "@vrooli/react-component-library/ToastManager/1";
import { Toast } from "./Toast";

/**
 * The previous specimen returned null. Every stage of the pipeline passed against the
 * blank page it produced, which is how a component with no exit animation, text that
 * overflows its own viewport and a dismiss control drawn as a punctuation mark stayed
 * green for ten releases. Each story below renders the real component and asserts
 * something that would break if that regressed.
 */

/** A path with no spaces — the shape of text a failure notice actually carries. */
const UNBROKEN =
  "git push failed: /home/runner/work/repository/packages/react-component-library/dist/components/Toast/versions/1.0.11/Toast.js";

function Seeder({ seed }: { seed: (actions: ToastActions) => void }) {
  const actions = useToastActions();
  useEffect(() => {
    seed(actions);
  }, [actions, seed]);
  return null;
}

function Stage({ seed }: { seed: (actions: ToastActions) => void }) {
  return (
    <ToastManagerProvider maxVisible={4}>
      <Seeder seed={seed} />
      <Toast />
    </ToastManagerProvider>
  );
}

/** One notice per tone, so the four icons are distinguishable at a glance. */
export function Tones() {
  return (
    <Stage
      seed={(actions) => {
        actions.push({
          id: "info",
          tone: "info",
          title: "Checking remote",
          durationMs: 0,
          dismissible: false,
        });
        actions.push({
          id: "success",
          tone: "success",
          title: "Pushed to origin/agi",
          durationMs: 0,
        });
        actions.push({
          id: "warning",
          tone: "warning",
          title: "Remote has 4 new commits",
          message: "Pull before pushing.",
          durationMs: 0,
        });
        actions.push({
          id: "error",
          tone: "error",
          title: "Push to origin/agi failed",
          message: "git push timed out before the transfer finished.",
          durationMs: 0,
        });
      }}
    />
  );
}

/**
 * A single unbroken token longer than the viewport. The notice has to wrap it: an error
 * that renders past its own edge is unreadable exactly when it matters.
 */
export function LongMessage() {
  return (
    <Stage
      seed={(actions) => {
        actions.push({
          id: "long",
          tone: "error",
          title: "Push to origin/agi failed",
          message: UNBROKEN,
          durationMs: 0,
          action: { label: "Try again", onSelect: () => undefined },
        });
      }}
    />
  );
}

/** A notice that never expires still offers the reader a way out. */
export function Dismissible() {
  return (
    <Stage
      seed={(actions) => {
        actions.push({
          id: "sticky",
          tone: "error",
          title: "Push failed",
          message: "Permission denied (publickey).",
          durationMs: 0,
        });
      }}
    />
  );
}
