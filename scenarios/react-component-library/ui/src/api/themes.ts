import { createClient } from "@connectrpc/connect";
import {
  ThemesService,
  type GetBuiltinThemeResponse,
  type GetThemeFromScenarioResponse,
  type ListBuiltinThemesResponse,
  type Theme,
} from "@vrooli/proto-types/react-component-library/v1/themes/themes_pb";

import { transport } from "./client";

export const themesClient = createClient(ThemesService, transport);

export type {
  GetBuiltinThemeResponse,
  GetThemeFromScenarioResponse,
  ListBuiltinThemesResponse,
  Theme,
};
