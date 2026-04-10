import type { FormEvent } from "react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Input } from "../ui/input";
import { Select } from "../ui/select";
import type { BacklogKind } from "../../types";

export type SimulationPayload = {
  kind: BacklogKind;
  mode: string;
  item_name: string;
  item_title: string;
  item_description: string;
  item_status: string;
  item_priority: string;
  item_tags: string;
  item_folder: string;
};

// eslint-disable-next-line react-refresh/only-export-components -- factory shared with PromptsPage
export const defaultSimulationPayload = (): SimulationPayload => ({
  kind: "idea",
  mode: "workshop",
  item_name: "sample-item",
  item_title: "Sample Item",
  item_description: "Sample description for simulation preview.",
  item_status: "backlog",
  item_priority: "3",
  item_tags: "sample, prompt-center",
  item_folder: "scenarios/swarm-manager/ideas/sample-item",
});

export interface SimulationDialogProps {
  isOpen: boolean;
  onClose: () => void;
  isPending: boolean;
  selectedSkillId: string;
  payload: SimulationPayload;
  onPayloadChange: (updater: (prev: SimulationPayload) => SimulationPayload) => void;
  kindOptions: BacklogKind[];
  modeOptions: string[];
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

export function SimulationDialog({
  isOpen,
  onClose,
  isPending,
  selectedSkillId,
  payload,
  onPayloadChange,
  kindOptions,
  modeOptions,
  onSubmit,
}: SimulationDialogProps) {
  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Simulation Preview"
      maxWidth="max-w-3xl"
      isLoading={isPending}
    >
      <p className="text-sm text-slate-400 -mt-4 mb-4">Enter context values and generate an exact prompt preview.</p>

      <form className="space-y-3" onSubmit={onSubmit}>
        <div className="grid gap-2 md:grid-cols-3">
          <Select
            value={payload.kind}
            onChange={(event) =>
              onPayloadChange((prev) => ({ ...prev, kind: event.target.value as BacklogKind }))
            }
          >
            {kindOptions.map((kind) => (
              <option key={kind} value={kind}>{kind}</option>
            ))}
          </Select>
          <Select
            value={payload.mode}
            onChange={(event) =>
              onPayloadChange((prev) => ({ ...prev, mode: event.target.value }))
            }
          >
            {modeOptions.map((mode) => (
              <option key={mode} value={mode}>{mode}</option>
            ))}
          </Select>
          <div className="rounded-md border border-slate-700/70 bg-slate-950 px-3 py-2 text-sm text-slate-400">
            {selectedSkillId ? `Skill: ${selectedSkillId}` : "Backlog prompt simulation"}
          </div>
        </div>

        <div className="grid gap-2 md:grid-cols-2">
          <Input
            value={payload.item_name}
            onChange={(event) => onPayloadChange((prev) => ({ ...prev, item_name: event.target.value }))}
            placeholder="item_name"
          />
          <Input
            value={payload.item_title}
            onChange={(event) => onPayloadChange((prev) => ({ ...prev, item_title: event.target.value }))}
            placeholder="item_title"
          />
          <Input
            value={payload.item_status}
            onChange={(event) => onPayloadChange((prev) => ({ ...prev, item_status: event.target.value }))}
            placeholder="item_status"
          />
          <Input
            value={payload.item_priority}
            onChange={(event) => onPayloadChange((prev) => ({ ...prev, item_priority: event.target.value }))}
            placeholder="item_priority"
          />
          <Input
            value={payload.item_tags}
            onChange={(event) => onPayloadChange((prev) => ({ ...prev, item_tags: event.target.value }))}
            placeholder="item_tags"
          />
          <Input
            value={payload.item_folder}
            onChange={(event) => onPayloadChange((prev) => ({ ...prev, item_folder: event.target.value }))}
            placeholder="item_folder"
          />
        </div>

        <textarea
          value={payload.item_description}
          onChange={(event) => onPayloadChange((prev) => ({ ...prev, item_description: event.target.value }))}
          className="min-h-[110px] w-full rounded-md border border-slate-700/70 bg-slate-950 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none"
          placeholder="item_description"
        />

        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onClose} disabled={isPending}>
            Cancel
          </Button>
          <Button type="submit" disabled={isPending}>
            {isPending ? "Generating..." : "Generate Preview"}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
