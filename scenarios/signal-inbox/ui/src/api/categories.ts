import { createClient } from "@connectrpc/connect";

import {
  CategoriesService,
  type Category,
  type Classification,
  type CreateCategoryRequest,
  type RenameCategoryRequest,
} from "../../../../../packages/proto/gen/typescript/signal-inbox/v1/categories/categories_pb";
import { transport } from "./client";

// The category service is intentionally separate from capture: category state
// controls ambient review only and never decides whether a signal is stored.
export const categoriesClient = createClient(CategoriesService, transport);

export type { Category, Classification, CreateCategoryRequest, RenameCategoryRequest };
