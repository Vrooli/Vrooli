import { createClient } from "@connectrpc/connect";
import {
  ComponentsService,
  type Component,
  type ListComponentsResponse,
} from "@vrooli/proto-types/react-component-library/v1/components/components_pb";

import { transport } from "./client";

export const componentsClient = createClient(ComponentsService, transport);

export type { Component, ListComponentsResponse };
