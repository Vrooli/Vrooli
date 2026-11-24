import { useEffect, useMemo, useState } from "react";
import { Code2, SlidersHorizontal } from "lucide-react";
import { useTranslation } from "../../i18n";
import { Chip, Empty, Select, TextArea, WbButton } from "../primitives";
import { VERDICT_ICON, VERDICT_KEY, asVerdict } from "../verdicts";
import type { WorkbenchStory } from "../data";

function parseArgs(story: WorkbenchStory | undefined): Record<string, unknown> {
  return story?.args ?? {};
}

function displayPropValue(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (value === null || value === undefined) return "";
  return JSON.stringify(value);
}

interface StoryField {
  path: string;
  kind?: string;
  options?: unknown[];
  default?: unknown;
}

function parseStoryFields(fieldsJson: string): StoryField[] {
  try {
    const parsed: unknown = JSON.parse(fieldsJson);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return [];
    const fields = (parsed as { fields?: unknown }).fields;
    if (!Array.isArray(fields)) return [];
    return fields
      .filter(
        (field): field is StoryField =>
          Boolean(field) &&
          typeof field === "object" &&
          !Array.isArray(field) &&
          typeof (field as StoryField).path === "string" &&
          (field as StoryField).path.length > 0,
      )
      .map((field) => ({
        ...field,
        options: Array.isArray(field.options) ? field.options : [],
      }));
  } catch {
    return [];
  }
}

function valueAtPath(value: Record<string, unknown>, path: string): unknown {
  return path.split(".").reduce<unknown>((current, key) => {
    if (!current || typeof current !== "object" || Array.isArray(current)) return undefined;
    return (current as Record<string, unknown>)[key];
  }, value);
}

function setPathValue(
  value: Record<string, unknown>,
  path: string,
  nextValue: unknown,
): Record<string, unknown> {
  const [head, ...tail] = path.split(".");
  if (!head) return value;
  if (tail.length === 0) return { ...value, [head]: nextValue };
  const child = value[head];
  const nested =
    child && typeof child === "object" && !Array.isArray(child)
      ? (child as Record<string, unknown>)
      : {};
  return { ...value, [head]: setPathValue(nested, tail.join("."), nextValue) };
}

function applyFieldDefaults(
  value: Record<string, unknown>,
  fields: StoryField[],
): Record<string, unknown> {
  return fields.reduce((props, field) => {
    const hasDefault = field.default !== undefined;
    const hasEnumFallback = field.kind === "enum" && (field.options?.length ?? 0) > 0;
    if ((!hasDefault && !hasEnumFallback) || valueAtPath(props, field.path) !== undefined) {
      return props;
    }
    const fallback = hasDefault ? field.default : field.options?.[0];
    return setPathValue(props, field.path, fallback);
  }, value);
}

function optionValue(value: unknown): string {
  return JSON.stringify(value);
}

function parseOptionValue(value: string): unknown {
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return value;
  }
}

export function ChecksList({
  rows,
  empty,
}: {
  rows: Array<{ id: string; label: string; verdict: string; detail?: string }>;
  empty: string;
}) {
  const { t } = useTranslation();
  if (rows.length === 0) {
    return (
      <Empty title={t("workbench.component.noChecksTitle", { defaultValue: "No checks declared" })}>
        {empty}
      </Empty>
    );
  }
  return (
    <div>
      {rows.map((row) => {
        const verdict = asVerdict(row.verdict);
        const Icon = VERDICT_ICON[verdict];
        return (
          <div key={row.id} className={`wb-checkrow is-${verdict}`}>
            <span aria-hidden className="wb-checkrow-icon">
              <Icon size={12} />
            </span>
            <span title={row.label}>{row.label}</span>
            <b className="wb-checkrow-verdict">
              {t(VERDICT_KEY[verdict], { defaultValue: verdict })}
            </b>
          </div>
        );
      })}
      <p className="wb-muted-2 wb-small" style={{ marginTop: 10 }}>
        {t("workbench.component.checksNote", {
          defaultValue:
            "Checks are captured evidence. An absent capture is unmeasured, not a pass.",
        })}
      </p>
    </div>
  );
}

export function PropsEditor({
  story,
  override,
  onApply,
  onReset,
  status,
}: {
  story: WorkbenchStory;
  override: Record<string, unknown> | null;
  onApply: (next: Record<string, unknown>) => void;
  onReset: () => void;
  status: string;
}) {
  const { t } = useTranslation();
  const baseline = useMemo(() => parseArgs(story), [story]);
  const fields = useMemo(() => parseStoryFields(story.fieldsJson), [story.fieldsJson]);
  const current = useMemo(
    () => applyFieldDefaults(override ?? baseline, fields),
    [baseline, fields, override],
  );
  const [draft, setDraft] = useState(() => JSON.stringify(current, null, 2));
  const [error, setError] = useState("");
  const [showJson, setShowJson] = useState(false);

  useEffect(() => {
    setDraft(JSON.stringify(current, null, 2));
    setError("");
    setShowJson(false);
  }, [current, story.id]);
  const controls =
    fields.length > 0
      ? fields.map((field) => ({
          key: field.path,
          value: valueAtPath(current, field.path) ?? field.default,
          kind: field.kind,
          options: field.options ?? [],
        }))
      : Object.entries(current).map(([key, value]) => ({
          key,
          value,
          kind: undefined,
          options: [],
        }));

  const setValue = (key: string, value: unknown) => {
    let source = current;
    try {
      const parsed: unknown = JSON.parse(draft);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        source = parsed as Record<string, unknown>;
      }
    } catch {
      // Keep the latest valid values when the advanced JSON draft is malformed.
    }
    const next = setPathValue(source, key, value);
    setDraft(JSON.stringify(next, null, 2));
    setError("");
  };

  const apply = () => {
    try {
      const parsed = JSON.parse(draft) as unknown;
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        setError(
          t("workbench.component.propsObject", { defaultValue: "Props must be a JSON object." }),
        );
        return;
      }
      onApply(parsed as Record<string, unknown>);
    } catch {
      setError(t("workbench.component.propsParse", { defaultValue: "That is not valid JSON." }));
    }
  };

  return (
    <div className="wb-stack tight">
      <div className="wb-row-between">
        <span className="wb-caption">
          {t("workbench.component.publicProps", { defaultValue: "Public props" })}
        </span>
        <Chip tone={status === "applied" ? "green" : status === "error" ? "red" : "ice"}>
          {status}
        </Chip>
      </div>
      {controls.length === 0 ? (
        <p className="wb-small wb-muted">
          {t("workbench.component.noProps", {
            defaultValue: "This story declares no public props.",
          })}
        </p>
      ) : (
        controls.map(({ key, value, kind, options }) => (
          <div key={key} className="wb-field">
            <label htmlFor={`prop-${story.id}-${key}`}>{key}</label>
            {kind === "enum" && options.length > 0 ? (
              <Select
                id={`prop-${story.id}-${key}`}
                value={optionValue(value ?? options[0])}
                onChange={(event) => setValue(key, parseOptionValue(event.target.value))}
              >
                {options.map((option) => (
                  <option key={optionValue(option)} value={optionValue(option)}>
                    {displayPropValue(option)}
                  </option>
                ))}
              </Select>
            ) : kind === "boolean" || (kind !== "number" && typeof value === "boolean") ? (
              <Select
                id={`prop-${story.id}-${key}`}
                value={value === true ? "true" : "false"}
                onChange={(event) => setValue(key, event.target.value === "true")}
              >
                <option value="true">{String(true)}</option>
                <option value="false">{String(false)}</option>
              </Select>
            ) : kind === "number" || typeof value === "number" ? (
              <input
                id={`prop-${story.id}-${key}`}
                className="wb-input"
                type="number"
                value={typeof value === "number" ? value : ""}
                onChange={(event) =>
                  setValue(key, event.target.value === "" ? "" : Number(event.target.value))
                }
              />
            ) : (
              <input
                id={`prop-${story.id}-${key}`}
                className="wb-input"
                value={displayPropValue(value)}
                onChange={(event) => setValue(key, event.target.value)}
              />
            )}
          </div>
        ))
      )}
      <WbButton variant="ghost" size="sm" onClick={() => setShowJson((shown) => !shown)}>
        <Code2 aria-hidden size={13} />
        {showJson
          ? t("workbench.component.hidePropsJson", { defaultValue: "Hide JSON" })
          : t("workbench.component.editPropsJson", { defaultValue: "Edit JSON" })}
      </WbButton>
      {showJson ? (
        <div className="wb-field">
          <label htmlFor={`props-json-${story.id}`}>
            {t("workbench.component.propsJson", { defaultValue: "Override JSON" })}
          </label>
          <TextArea
            id={`props-json-${story.id}`}
            value={draft}
            onChange={(event) => {
              setDraft(event.target.value);
              setError("");
            }}
            spellCheck={false}
          />
        </div>
      ) : null}
      {error ? (
        <span className="wb-small" style={{ color: "var(--wb-red)" }}>
          {error}
        </span>
      ) : null}
      <div className="wb-row">
        <WbButton variant="primary" size="sm" onClick={apply}>
          <SlidersHorizontal aria-hidden size={13} />
          {t("workbench.component.applyProps", { defaultValue: "Apply props" })}
        </WbButton>
        <WbButton variant="ghost" size="sm" onClick={onReset}>
          {t("workbench.component.resetProps", { defaultValue: "Reset" })}
        </WbButton>
      </div>
      <p className="wb-muted-2 wb-small">
        {t("workbench.component.propsNote", {
          defaultValue:
            "Overrides are applied to the real harness. They are a local experiment and do not publish a version.",
        })}
      </p>
    </div>
  );
}
