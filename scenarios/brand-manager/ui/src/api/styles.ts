import { createClient } from "@connectrpc/connect";
import {
  StylesService,
  type ContainerStyle,
  type ProductLine,
} from "@vrooli/proto-types/brand-manager/v1/styles/styles_pb";

import { transport } from "./client";

export const stylesClient = createClient(StylesService, transport);

/** listContainerStyles returns every container style record. */
export async function listContainerStyles(): Promise<ContainerStyle[]> {
  const resp = await stylesClient.listContainerStyles({});
  return resp.styles;
}

/** getContainerStyle fetches one container style by id. */
export async function getContainerStyle(id: string): Promise<ContainerStyle | undefined> {
  const resp = await stylesClient.getContainerStyle({ id });
  return resp.style;
}

/** listProductLines returns every product line record. */
export async function listProductLines(): Promise<ProductLine[]> {
  const resp = await stylesClient.listProductLines({});
  return resp.lines;
}

export type { ContainerStyle, ProductLine };
