import { createClient } from "@connectrpc/connect";
import { PublicationIntentService } from "@vrooli/proto-types/content-desk/v1/intents/intents_pb";
import { transport } from "./client";

/** Editorial timing intentions only; these requests do not book or schedule time. */
export const intentsClient = createClient(PublicationIntentService, transport);
