// AI_CHECK: TYPE_SAFETY=fixed-i18n-mock-any-types | LAST: 2025-06-28
import { vi } from "vitest";
import type React from "react";

export const mockUseTranslation = vi.fn(() => ({
  t: (key: string, options?: Record<string, unknown>) => {
    // Handle default values first
    if (options && typeof options === "object" && "defaultValue" in options) {
      return options.defaultValue;
    }
    
    // Handle pluralization
    if (options && typeof options === "object" && "count" in options) {
      return `${key}_${options.count}`;
    }
    
    // Handle interpolation
    if (options && typeof options === "object") {
      let result = key;
      Object.entries(options).forEach(([k, v]) => {
        if (k !== "ns" && k !== "defaultValue") {
          result = result.replace(`{{${k}}}`, String(v));
        }
      });
      return result;
    }
    
    return key;
  },
  i18n: {
    language: "en",
    changeLanguage: vi.fn(),
  },
}));

// Mock i18next object for tests that need it directly
export const mockI18next = {
  t: vi.fn((key: string, options?: Record<string, unknown>) => {
    // Use the same implementation as useTranslation
    const { t } = mockUseTranslation();
    return t(key, options);
  }),
  changeLanguage: vi.fn(),
  language: "en",
  use: vi.fn(() => mockI18next),
  init: vi.fn(),
};

export const reactI18nextMock = {
  useTranslation: mockUseTranslation,
  Trans: ({ children }: { children: React.ReactNode }) => children,
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  i18next: mockI18next,
};
