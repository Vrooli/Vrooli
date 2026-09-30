import { createClient } from "@connectrpc/connect";
import { DesktopSessionService, OperatorSessionService } from "@vrooli/proto-types/portal/v1/surfaces/surfaces_pb";
import { transport } from "./client";
export const desktopClient = createClient(DesktopSessionService, transport);
export const operatorClient = createClient(OperatorSessionService, transport);
export const operatorHeaders = (token: string) => ({ Authorization: `Bearer ${token}` });
