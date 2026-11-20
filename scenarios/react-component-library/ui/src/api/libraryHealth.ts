import { createClient } from "@connectrpc/connect";
import {
  LibraryHealthService,
  SectionState,
  type LibraryHealth,
} from "@vrooli/proto-types/react-component-library/v1/libraryhealth/library_health_pb";

import { slowTransport } from "./client";

const client = createClient(LibraryHealthService, slowTransport);

export { SectionState };
export type { LibraryHealth };

export async function getLibraryHealth(activityDays = 0): Promise<LibraryHealth | null> {
  const response = await client.getLibraryHealth({ activityDays });
  return response.health ?? null;
}
