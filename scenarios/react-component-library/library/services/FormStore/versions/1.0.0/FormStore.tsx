/** @vrooliComponentSource services.form-store */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  createScopedStore,
  type ScopedStore,
} from "@vrooli/react-component-library/createScopedStore/1.0.0";

export type FormPhase =
  | "idle"
  | "validating"
  | "submitting"
  | "saving"
  | "success"
  | "error"
  | "offline";

export interface FormFieldState<T> {
  value: T;
  defaultValue: T;
  touched: boolean;
  dirty: boolean;
  validating: boolean;
  error?: string;
}

export interface FormState<TValues extends Record<string, unknown>> {
  fields: { [K in keyof TValues]: FormFieldState<TValues[K]> };
  phase: FormPhase;
  submitCount: number;
  error?: string;
  conflict?: { fields: Array<keyof TValues>; message: string };
}

export type FormValidationResult<TValues extends Record<string, unknown>> =
  | Partial<Record<keyof TValues, string | undefined>>
  | undefined;

type MaybeAsync<T> = T | Promise<T>;
type FormHandler<TValues extends Record<string, unknown>> = (
  values: TValues,
) => MaybeAsync<undefined>;

export interface FormStoreOptions<TValues extends Record<string, unknown>> {
  initialValues: TValues;
  validate?: (values: TValues) => MaybeAsync<FormValidationResult<TValues>>;
  validateField?: <K extends keyof TValues>(
    field: K,
    value: TValues[K],
    values: TValues,
  ) => string | undefined | Promise<string | undefined>;
  autosave?: (values: TValues) => MaybeAsync<undefined>;
  autosaveDelay?: number;
}

export interface FormStore<TValues extends Record<string, unknown>>
  extends ScopedStore<FormState<TValues>> {
  getValues: () => TValues;
  getField: <K extends keyof TValues>(field: K) => FormFieldState<TValues[K]>;
  setValue: <K extends keyof TValues>(field: K, value: TValues[K]) => void;
  setValues: (values: Partial<TValues>) => void;
  touch: (field: keyof TValues) => void;
  setError: (field: keyof TValues, message?: string) => void;
  clearErrors: () => void;
  reset: (values?: TValues) => void;
  validate: () => Promise<boolean>;
  submit: (handler: FormHandler<TValues>) => Promise<boolean>;
  save: (handler?: FormHandler<TValues>) => Promise<boolean>;
  scheduleAutosave: () => void;
  setPhase: (phase: FormPhase, error?: string) => void;
  setConflict: (fields: Array<keyof TValues>, message: string) => void;
}

function isPromise<T>(value: T | Promise<T>): value is Promise<T> {
  return typeof (value as Promise<T> | undefined)?.then === "function";
}

function createFields<TValues extends Record<string, unknown>>(
  values: TValues,
) {
  return Object.fromEntries(
    Object.entries(values).map(([key, value]) => [
      key,
      {
        value,
        defaultValue: value,
        touched: false,
        dirty: false,
        validating: false,
      },
    ]),
  ) as { [K in keyof TValues]: FormFieldState<TValues[K]> };
}

function valuesFromState<TValues extends Record<string, unknown>>(
  state: FormState<TValues>,
): TValues {
  return Object.fromEntries(
    (
      Object.entries(state.fields) as Array<[string, FormFieldState<unknown>]>
    ).map(([key, field]) => [key, field.value]),
  ) as TValues;
}

export function createFormStore<TValues extends Record<string, unknown>>(
  options: FormStoreOptions<TValues>,
): FormStore<TValues> {
  const baseValues = { ...options.initialValues };
  let autosaveTimer: ReturnType<typeof setTimeout> | undefined;
  const store = createScopedStore<FormState<TValues>>({
    fields: createFields(baseValues),
    phase: "idle",
    submitCount: 0,
  });

  const patch = (updater: (state: FormState<TValues>) => FormState<TValues>) =>
    store.set(updater);
  const getValues = () => valuesFromState(store.get());
  const getField = <K extends keyof TValues>(field: K) =>
    store.get().fields[field];

  const setValue = <K extends keyof TValues>(field: K, value: TValues[K]) => {
    patch((state) => ({
      ...state,
      fields: {
        ...state.fields,
        [field]: {
          ...state.fields[field],
          value,
          dirty: !Object.is(value, state.fields[field].defaultValue),
          error: undefined,
        },
      },
      error: undefined,
      conflict: undefined,
      phase:
        state.phase === "success" || state.phase === "error"
          ? "idle"
          : state.phase,
    }));
  };

  const validate = async () => {
    const values = getValues();
    patch((state) => ({
      ...state,
      phase: "validating",
      fields: Object.fromEntries(
        Object.entries(state.fields).map(([key, field]) => [
          key,
          { ...field, validating: true },
        ]),
      ) as FormState<TValues>["fields"],
    }));
    const result = options.validate?.(values);
    const formErrors = ((isPromise(result) ? await result : result) ??
      {}) as Partial<Record<keyof TValues, string | undefined>>;
    const fieldErrors = (
      await Promise.all(
        (Object.keys(values) as Array<keyof TValues>).map(
          async (field) =>
            [
              field,
              options.validateField
                ? await options.validateField(field, values[field], values)
                : undefined,
            ] as const,
        ),
      )
    ).reduce<Partial<Record<keyof TValues, string | undefined>>>(
      (all, [field, error]) => ({ ...all, [field]: error }),
      {},
    );
    const errors = { ...fieldErrors, ...formErrors };
    patch((state) => ({
      ...state,
      phase: "idle",
      fields: Object.fromEntries(
        Object.entries(state.fields).map(([key, field]) => [
          key,
          { ...field, validating: false, error: errors[key as keyof TValues] },
        ]),
      ) as FormState<TValues>["fields"],
    }));
    return Object.values(errors).every((error) => !error);
  };

  const submit = async (handler: FormHandler<TValues>) => {
    const valid = await validate();
    if (!valid) {
      patch((state) => ({
        ...state,
        submitCount: state.submitCount + 1,
        phase: "error",
      }));
      return false;
    }
    patch((state) => ({
      ...state,
      submitCount: state.submitCount + 1,
      phase: "submitting",
      error: undefined,
    }));
    try {
      await handler(getValues());
      patch((state) => ({ ...state, phase: "success" }));
      return true;
    } catch (error) {
      patch((state) => ({
        ...state,
        phase: "error",
        error:
          error instanceof Error ? error.message : "Unable to save changes.",
      }));
      return false;
    }
  };

  const save = async (handler = options.autosave) => {
    if (!handler) return false;
    patch((state) => ({ ...state, phase: "saving", error: undefined }));
    try {
      await handler(getValues());
      patch((state) => ({ ...state, phase: "success" }));
      return true;
    } catch (error) {
      patch((state) => ({
        ...state,
        phase: "error",
        error:
          error instanceof Error ? error.message : "Unable to save changes.",
      }));
      return false;
    }
  };

  const scheduleAutosave = () => {
    if (!options.autosave) return;
    if (autosaveTimer) clearTimeout(autosaveTimer);
    autosaveTimer = setTimeout(
      () => {
        autosaveTimer = undefined;
        void save();
      },
      Math.max(0, options.autosaveDelay ?? 600),
    );
  };

  return {
    ...store,
    getValues,
    getField,
    setValue: (field, value) => {
      setValue(field, value);
      scheduleAutosave();
    },
    setValues: (values) =>
      Object.entries(values).forEach(([field, value]) =>
        setValue(field as keyof TValues, value as TValues[keyof TValues]),
      ),
    touch: (field) =>
      patch((state) => ({
        ...state,
        fields: {
          ...state.fields,
          [field]: { ...state.fields[field], touched: true },
        },
      })),
    setError: (field, message) =>
      patch((state) => ({
        ...state,
        phase: message ? "error" : state.phase,
        fields: {
          ...state.fields,
          [field]: { ...state.fields[field], error: message },
        },
      })),
    clearErrors: () =>
      patch((state) => ({
        ...state,
        error: undefined,
        fields: Object.fromEntries(
          Object.entries(state.fields).map(([key, field]) => [
            key,
            { ...field, error: undefined },
          ]),
        ) as FormState<TValues>["fields"],
      })),
    reset: (values = baseValues) =>
      patch(() => ({
        fields: createFields(values),
        phase: "idle",
        submitCount: 0,
      })),
    validate,
    submit,
    save,
    scheduleAutosave,
    setPhase: (phase, error) => patch((state) => ({ ...state, phase, error })),
    setConflict: (fields, message) =>
      patch((state) => ({
        ...state,
        phase: "error",
        conflict: { fields, message },
      })),
  };
}

export function useFormStore<
  TValues extends Record<string, unknown>,
  K extends keyof TValues,
>(store: FormStore<TValues>, field: K) {
  const [, rerender] = useState(0);
  const stableStore = useRef(store).current;
  useEffect(
    () => stableStore.subscribe(() => rerender((count) => count + 1)),
    [stableStore],
  );
  const state = stableStore.getField(field);
  const setValue = useCallback(
    (value: TValues[K]) => stableStore.setValue(field, value),
    [field, stableStore],
  );
  return { ...state, setValue, touch: () => stableStore.touch(field) };
}
