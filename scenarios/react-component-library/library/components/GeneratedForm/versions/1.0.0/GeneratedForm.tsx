/** @vrooliComponentSource forms.generated-form */
import {
  useId,
  useState,
  type CSSProperties,
  type ReactElement,
  type ReactNode,
} from "react";
import { ArrayField } from "../../../../components/ArrayField/versions/1.0.0/ArrayField";
import { ComputedField } from "../../../../components/ComputedField/versions/1.0.0/ComputedField";
import { ConditionalField } from "../../../../components/ConditionalField/versions/1.0.0/ConditionalField";
import { Form } from "../../../../components/Form/versions/1.0.0/Form";
import { FormActions } from "../../../../components/FormActions/versions/1.0.0/FormActions";
import { FormField } from "../../../../components/FormField/versions/1.0.0/FormField";
import { FormSection } from "../../../../components/FormSection/versions/1.0.0/FormSection";
import { ObjectField } from "../../../../components/ObjectField/versions/1.0.0/ObjectField";
import { ValidationSummary } from "../../../../components/ValidationSummary/versions/1.0.0/ValidationSummary";
import {
  createFormStore,
  type FormStore,
} from "../../../../services/FormStore/versions/1.0.0/FormStore";

export type GeneratedFieldType =
  | "text"
  | "number"
  | "email"
  | "textarea"
  | "select"
  | "checkbox"
  | "array"
  | "object"
  | "computed";

export interface GeneratedField<
  TValues extends Record<string, unknown> = Record<string, unknown>,
> {
  name: keyof TValues & string;
  type: GeneratedFieldType;
  label: ReactNode;
  defaultValue?: unknown;
  description?: ReactNode;
  section?: string;
  required?: boolean;
  options?: Array<{ value: string; label: ReactNode }>;
  when?: (values: TValues) => boolean;
  conditionalMode?: "hide" | "disable" | "reset";
  compute?: (values: TValues) => TValues[keyof TValues];
  format?: (value: unknown) => ReactNode;
  createItem?: () => unknown;
  itemLabel?: (index: number) => ReactNode;
  renderItem?: (context: {
    item: unknown;
    index: number;
    setValue: (value: unknown) => void;
  }) => ReactElement;
  objectChildren?: (context: {
    value: Record<string, unknown>;
    setValue: (key: string, value: unknown) => void;
    getError: (key: string) => string | undefined;
  }) => ReactNode;
  objectDefaultValue?: Record<string, unknown>;
}

export interface GeneratedFormSection {
  id: string;
  title: ReactNode;
  description?: ReactNode;
  collapsible?: boolean;
  defaultOpen?: boolean;
}

export interface GeneratedFormProps<
  TValues extends Record<string, unknown> = Record<string, unknown>,
> {
  /** Supply a store when the host owns values and validation. */
  store?: FormStore<TValues>;
  /** Uncontrolled forms create and retain their own scoped store. */
  mode?: "controlled" | "uncontrolled";
  /** Initial values used when no external store is supplied. */
  initialValues?: TValues;
  fields: Array<GeneratedField<TValues>>;
  sections?: GeneratedFormSection[];
  title?: ReactNode;
  description?: ReactNode;
  submitLabel?: ReactNode;
  onSubmit?: (values: TValues) => void | Promise<void>;
  className?: string;
  style?: CSSProperties;
}

function initialValuesFor<TValues extends Record<string, unknown>>(
  fields: Array<GeneratedField<TValues>>,
): TValues {
  const values = Object.fromEntries(
    fields.map((field) => {
      if (field.defaultValue !== undefined)
        return [field.name, field.defaultValue];
      if (field.type === "checkbox") return [field.name, false];
      if (field.type === "number") return [field.name, 0];
      if (field.type === "array") return [field.name, []];
      if (field.type === "object") return [field.name, {}];
      return [field.name, ""];
    }),
  );
  return values as TValues;
}

const styles = `
  [data-rcl-generated-form] { min-inline-size: 0; }
  [data-rcl-generated-form-sections] { display: grid; gap: var(--space-md, 1rem); min-inline-size: 0; }
  [data-rcl-generated-form-section-fields] { display: grid; gap: var(--space-md, 1rem); min-inline-size: 0; }
  [data-rcl-generated-form-field] { min-inline-size: 0; }
  [data-rcl-generated-form-control] { box-sizing: border-box; display: block; inline-size: 100%; min-block-size: var(--tap-target-min, 44px); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .625rem); background: var(--color-surface, #fff); color: var(--color-foreground, #0f172a); padding-inline: var(--space-sm, .75rem); font: var(--text-body, 400 .875rem/1.375rem system-ui, sans-serif); }
  [data-rcl-generated-form-control]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus, #2563eb) 30%, transparent); outline-offset: 2px; border-color: var(--color-primary, #2563eb); }
  textarea[data-rcl-generated-form-control] { min-block-size: 7rem; padding-block: var(--space-sm, .75rem); resize: vertical; }
  select[data-rcl-generated-form-control] { appearance: auto; }
  [data-rcl-generated-form-checkbox] { display: flex; align-items: center; gap: var(--space-xs, .625rem); min-block-size: var(--tap-target-min, 44px); color: var(--color-foreground, #0f172a); font: var(--text-body, 400 .875rem/1.375rem system-ui, sans-serif); }
  [data-rcl-generated-form-checkbox] input { inline-size: 1.125rem; block-size: 1.125rem; accent-color: var(--color-primary, #2563eb); }
  @media (max-width: 34rem) { [data-rcl-generated-form-sections] { gap: var(--space-sm, .75rem); } }
`;

function displayValue(value: unknown) {
  if (typeof value === "string" || typeof value === "number")
    return String(value);
  return "";
}

function InputControl<TValues extends Record<string, unknown>>({
  field,
  store,
}: {
  field: GeneratedField<TValues>;
  store: FormStore<TValues>;
}) {
  const value = store.getField(field.name).value;
  const setValue = (next: unknown) =>
    store.setValue(field.name, next as TValues[typeof field.name]);
  if (field.type === "checkbox")
    return (
      <label data-rcl-generated-form-checkbox>
        <input
          type="checkbox"
          checked={Boolean(value)}
          onChange={(event) => setValue(event.target.checked)}
        />
        {field.label}
      </label>
    );
  if (field.type === "select")
    return (
      <select
        data-rcl-generated-form-control
        value={String(value ?? "")}
        onChange={(event) => setValue(event.target.value)}
      >
        {field.options?.map((option) => (
          <option value={option.value} key={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    );
  if (field.type === "textarea")
    return (
      <textarea
        data-rcl-generated-form-control
        value={String(value ?? "")}
        onChange={(event) => setValue(event.target.value)}
      />
    );
  return (
    <input
      data-rcl-generated-form-control
      type={
        field.type === "number"
          ? "number"
          : field.type === "email"
            ? "email"
            : "text"
      }
      value={String(value ?? "")}
      onChange={(event) =>
        setValue(
          field.type === "number"
            ? Number(event.target.value)
            : event.target.value,
        )
      }
    />
  );
}

function RenderField<TValues extends Record<string, unknown>>({
  field,
  store,
}: {
  field: GeneratedField<TValues>;
  store: FormStore<TValues>;
}) {
  const fieldState = store.getField(field.name);
  if (field.type === "computed")
    return (
      <ComputedField
        store={store}
        field={field.name}
        label={field.label}
        description={field.description}
        compute={field.compute ?? (() => "" as TValues[keyof TValues])}
        format={field.format ?? ((value) => String(value))}
      />
    );
  if (field.type === "array")
    return (
      <ArrayField
        store={store}
        field={field.name}
        label={field.label}
        description={field.description}
        createItem={field.createItem ?? (() => "")}
        itemLabel={field.itemLabel}
        renderItem={({ item, actions }) =>
          field.renderItem?.({
            item,
            index: actions.index,
            setValue: actions.setValue,
          }) ?? (
            <input
              data-rcl-generated-form-control
              aria-label={`${typeof field.label === "string" ? field.label : "Item"} ${actions.index + 1}`}
              value={displayValue(item)}
              onChange={(event) => actions.setValue(event.target.value)}
            />
          )
        }
      />
    );
  if (field.type === "object")
    return (
      <ObjectField
        store={store}
        field={field.name}
        title={field.label}
        description={field.description}
        defaultValue={field.objectDefaultValue}
      >
        {(context) =>
          field.objectChildren?.({
            value: context.value,
            setValue: (key, value) => context.setValue(key, value),
            getError: (key) => context.getError(key),
          })
        }
      </ObjectField>
    );
  if (field.type === "checkbox")
    return <InputControl field={field} store={store} />;
  return (
    <FormField
      label={field.label}
      description={field.description}
      required={field.required}
      error={fieldState.error}
      control={<InputControl field={field} store={store} />}
    />
  );
}

export function GeneratedForm<
  TValues extends Record<string, unknown> = Record<string, unknown>,
>({
  store,
  mode,
  initialValues,
  fields,
  sections = [],
  title,
  description,
  submitLabel = "Save changes",
  onSubmit,
  className,
  style,
}: GeneratedFormProps<TValues>) {
  const generatedId = useId().replace(/:/g, "");
  const [internalStore] = useState(() =>
    createFormStore<TValues>({
      initialValues: initialValues ?? initialValuesFor(fields),
    }),
  );
  const resolvedStore = store ?? internalStore;
  const resolvedMode = mode ?? (store ? "controlled" : "uncontrolled");
  const values = resolvedStore.getValues();
  const sectionsById = new Map(
    sections.map((section) => [section.id, section]),
  );
  const unsectioned = fields.filter((field) => !field.section);
  const renderField = (field: GeneratedField<TValues>) => {
    const content = (
      <div key={field.name} data-rcl-generated-form-field>
        {field.when ? (
          <ConditionalField
            store={resolvedStore}
            field={field.name}
            when={field.when}
            mode={field.conditionalMode ?? "hide"}
          >
            <RenderField field={field} store={resolvedStore} />
          </ConditionalField>
        ) : (
          <RenderField field={field} store={resolvedStore} />
        )}
      </div>
    );
    return content;
  };
  const fieldLabels = Object.fromEntries(
    fields.map((field) => [field.name, field.label]),
  ) as Partial<Record<keyof TValues, ReactNode>>;
  return (
    <div
      className={className}
      style={style}
      data-rcl-generated-form
      data-form-id={generatedId}
    >
      <style
        data-rcl-generated-form-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <Form
        mode={resolvedMode}
        store={resolvedStore}
        title={title}
        description={description}
        aria-label={typeof title === "string" ? title : "Generated form"}
        onSubmit={onSubmit}
        footer={<FormActions store={resolvedStore} submitLabel={submitLabel} />}
      >
        <ValidationSummary store={resolvedStore} fieldLabels={fieldLabels} />
        <div data-rcl-generated-form-sections>
          {unsectioned.map(renderField)}
          {sections.map((section) => (
            <FormSection
              key={section.id}
              title={section.title}
              description={section.description}
              collapsible={section.collapsible}
              defaultOpen={section.defaultOpen}
            >
              <div data-rcl-generated-form-section-fields>
                {fields
                  .filter((field) => field.section === section.id)
                  .map(renderField)}
              </div>
            </FormSection>
          ))}
        </div>
        <span hidden data-rcl-generated-form-value-count>
          {Object.keys(values).length}
        </span>
      </Form>
      {sectionsById.size === 0 && (
        <span hidden data-rcl-generated-form-no-sections />
      )}
    </div>
  );
}
