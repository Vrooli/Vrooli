import { create } from "zustand";
import { parseTagsInput, tagsToInput } from "../lib";
import type { BacklogFormValues, BacklogKind } from "../types";

interface BacklogFormStoreState {
  values: BacklogFormValues;
  tagsInput: string;
  nameDirty: boolean;
  error: string | null;
  setField: <K extends keyof BacklogFormValues>(field: K, value: BacklogFormValues[K]) => void;
  setTagsInput: (value: string) => void;
  setNameDirty: (dirty: boolean) => void;
  setError: (error: string | null) => void;
  initialize: (options: {
    isEditMode: boolean;
    defaultKind: BacklogKind;
    initialValues?: BacklogFormValues;
  }) => void;
  reset: () => void;
}

const buildFormValues = (
  defaultKind: BacklogKind,
  initialValues?: BacklogFormValues
): BacklogFormValues => {
  const nextKind = initialValues?.kind ?? defaultKind;
  return {
    name: initialValues?.name ?? "",
    title: initialValues?.title ?? "",
    description: initialValues?.description ?? "",
    status: initialValues?.status ?? "backlog",
    priority: initialValues?.priority ?? 5,
    tags: initialValues?.tags ?? [],
    kind: nextKind,
    dependsOn: initialValues?.dependsOn ?? [],
    milestone: initialValues?.milestone ?? "",
    effort: initialValues?.effort ?? "",
    acceptanceAllow: initialValues?.acceptanceAllow ?? [],
    acceptanceDeny: initialValues?.acceptanceDeny ?? [],
  };
};

export const backlogFormInitialState = {
  values: buildFormValues("idea"),
  tagsInput: "",
  nameDirty: false,
  error: null,
};

export const useBacklogFormStore = create<BacklogFormStoreState>((set) => ({
  ...backlogFormInitialState,

  setField: (field, value) =>
    set((state) => ({
      values: {
        ...state.values,
        [field]: value,
      },
    })),

  setTagsInput: (value) =>
    set((state) => ({
      tagsInput: value,
      values: {
        ...state.values,
        tags: parseTagsInput(value),
      },
    })),

  setNameDirty: (dirty) => set({ nameDirty: dirty }),

  setError: (error) => set({ error }),

  initialize: ({ isEditMode, defaultKind, initialValues }) => {
    const nextValues = buildFormValues(defaultKind, initialValues);
    set({
      values: nextValues,
      tagsInput: tagsToInput(nextValues.tags),
      nameDirty: isEditMode,
      error: null,
    });
  },

  reset: () => set({ ...backlogFormInitialState }),
}));
