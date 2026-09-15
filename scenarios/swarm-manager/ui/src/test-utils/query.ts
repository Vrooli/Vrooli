import {
  QueryClientProvider,
} from "@tanstack/react-query";
import { createTestQueryClient } from "@vrooli/api-base/testing";
import { createElement, type ReactNode } from "react";

export function createQueryWrapper(queryClient = createTestQueryClient()) {
  return function QueryWrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}
