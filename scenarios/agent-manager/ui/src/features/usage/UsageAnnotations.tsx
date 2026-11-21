import { Card } from "@vrooli/react-component-library/Card/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Table } from "@vrooli/react-component-library/Table/1";
import { DataState } from "../../components/feedback/DataState";
import { formatCode, formatCount, formatPercent } from "../../lib/format";
import { ANNOTATION_WINDOWS, useAnnotationGate } from "./api";
import { calibrationText, gateWord, isSilent, measurementText, proofModel, whyUnproven, type AnnotationGate, type GateLabel } from "./annotationGate";

const CELL = "max-md:!block max-md:!border-0 max-md:!px-0 max-md:!py-0.5";
const TONES = { proven: "success", partial: "warning", unproven: "neutral" } as const;

/** Column labels return inline on a phone, where the header row is hidden. */
function Label({ children }: { children: string }) {
  return <span className="mr-2 text-xs text-muted-foreground md:hidden">{children}</span>;
}

/** The model a label's proof is for, how its probabilities are read, and what the hold-out measured: nothing when no model was measured. */
export function Proof({ label }: { label: GateLabel }) {
  const model = proofModel(label);
  if (!model) return null;
  return (
    <div className="text-xs text-muted-foreground" data-testid="usage-annotation-proof">
      <div className="break-words">{label.gateState === "proven" ? "Proven for" : "Measured for"} {model}{label.calibration ? ` · ${calibrationText(label)}` : ""}</div>
      <div>{measurementText(label)}</div>
    </div>
  );
}

/** Each model that answered for the label, so a proof for one is never read as proof for all. */
function Models({ label }: { label: GateLabel }) {
  if (label.models.length < 2 && label.models.every((model) => model.proven)) return null;
  return (
    <ul className="m-0 mt-1 list-none p-0 text-xs text-muted-foreground" aria-label="Answers by model" data-testid="usage-annotation-models">
      {label.models.map((model) => (
        <li key={`${model.model}/${model.role}`} className="break-words">{model.model}{model.role ? ` · ${model.role}` : ""}: {formatCount(model.observed)} observed, {model.proven ? "proven" : formatCode(model.gateState).toLowerCase()}</li>
      ))}
    </ul>
  );
}

/** The inferred-annotation lens: coverage first, a rate only over answers from a proven model. */
export function Annotations({ data, windowLabel }: { data: AnnotationGate; windowLabel: string }) {
  // The lens reports friction labels only; mood labels have their own card and run.type facets no rate.
  const frictionLabels = data.labels.filter((label) => label.friction);
  const shown = frictionLabels.filter((label) => !isSilent(label));
  const silent = frictionLabels.length - shown.length;
  return (
    <Card className="flex flex-col gap-3 p-4" data-testid="usage-annotations">
      <div>
        <h3 className="text-base font-semibold">Inferred annotations</h3>
        <p className="text-sm text-muted-foreground" data-testid="usage-annotations-coverage">
          Coverage {formatPercent(data.coverage, 0)} ({formatCount(data.observedRuns)} of {formatCount(data.eligibleRuns)} runs observed) · pack {data.pack} {data.packVersion} · last {windowLabel}
        </p>
      </div>
      {shown.length === 0 ? (
        <p className="text-sm" role="status">No label has an observation in this window. Unknown, not zero friction.</p>
      ) : (
        <Table>
          <table>
            <thead className="max-md:sr-only">
              <tr><th scope="col">Label</th><th scope="col">Gate</th><th scope="col" className="md:text-right">Observed</th><th scope="col" className="md:text-right">Rate</th></tr>
            </thead>
            <tbody>
              {shown.map((label) => {
                const word = gateWord(label);
                return (
                  <tr key={label.id} data-testid="usage-annotation-label" className="max-md:block max-md:border-b max-md:border-border max-md:py-2">
                    <td className={CELL}><b className="break-all">{label.id}</b><div className="text-xs text-muted-foreground">{formatCode(label.group)}{label.scope === "message" ? " · per message" : ""}</div></td>
                    <td className={CELL}>
                      <StatusBadge tone={TONES[word]}>{word}</StatusBadge>
                      {word === "unproven" ? <div className="text-xs text-muted-foreground">{whyUnproven(label)}</div> : null}
                      {word === "partial" ? <div className="text-xs text-muted-foreground">{formatCount(label.provenObserved)} of {formatCount(label.observed)} answers are from a proven model</div> : null}
                      <Proof label={label} />
                      <Models label={label} />
                    </td>
                    <td className={`${CELL} whitespace-nowrap tabular-nums md:text-right`}><Label>Observed</Label>{formatCount(label.observed)}/{formatCount(label.eligible)}</td>
                    <td className={`${CELL} whitespace-nowrap tabular-nums md:text-right`}>
                      <Label>Rate</Label>
                      {label.rate === undefined ? "unproven" : <>{formatPercent(label.rate, 1)}<div className="text-xs text-muted-foreground">of {formatCount(label.provenObserved)} proven</div></>}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </Table>
      )}
      {silent > 0 ? <p className="text-xs text-muted-foreground">{silent} label(s) have no observation in this window (unknown, not zero).</p> : null}
      {data.note ? <p className="text-xs text-muted-foreground">{data.note}</p> : null}
      <p className="text-xs text-muted-foreground">A rate counts only answers from a model whose hold-out measurement cleared the label's targets. An unproven label is evidence, not a rate. Coverage is always shown; an empty window is unknown.</p>
    </Card>
  );
}

/** Usage › Friction › Inferred annotations, read from the annotation gate measure. */
export function AnnotationsSection() {
  const window = ANNOTATION_WINDOWS[1];
  const gate = useAnnotationGate(window.id);
  return (
    <DataState subject="Inferred annotations" compact loading={gate.isLoading} error={gate.error} isEmpty={!gate.data} onRetry={() => void gate.refetch()} empty={null}>
      {gate.data ? <Annotations data={gate.data} windowLabel={window.label} /> : null}
    </DataState>
  );
}
