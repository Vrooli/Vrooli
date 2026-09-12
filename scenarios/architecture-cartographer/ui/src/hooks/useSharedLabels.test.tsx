import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import type { ReactNode } from "react";

import { i18n } from "../i18n";
import { useSharedLabels } from "./useSharedLabels";

const wrapper = ({ children }: { children: ReactNode }) => (
  <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
);

describe("useSharedLabels", () => {
  it("returns a populated label tree for every shared.* key", () => {
    const { result } = renderHook(() => useSharedLabels(), { wrapper });
    const labels = result.current;
    expect(labels.empty.title).not.toBe("");
    expect(labels.loading.label).not.toBe("");
    expect(labels.error.retry).not.toBe("");
    expect(labels.severity.critical).not.toBe("");
    expect(labels.diff.added).not.toBe("");
    expect(labels.splitPane.resizeHandle).not.toBe("");
    expect(labels.dataTable.empty).not.toBe("");
  });
});
