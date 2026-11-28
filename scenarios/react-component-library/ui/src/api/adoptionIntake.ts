import { createClient } from "@connectrpc/connect";
import { AdoptionIntakeService } from "@vrooli/proto-types/react-component-library/v1/adoptionintake/adoption_intake_pb";

import { transport } from "./client";

export const adoptionIntakeClient = createClient(AdoptionIntakeService, transport);
