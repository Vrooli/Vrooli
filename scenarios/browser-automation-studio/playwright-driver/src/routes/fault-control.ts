import type { IncomingMessage, ServerResponse } from 'http';
import type { Config } from '../config';
import type { FaultController, ArmFaultRequest } from '../fault-control';
import { parseJsonBody, sendJson } from '../middleware';

function loopback(req: IncomingMessage): boolean {
  return ['127.0.0.1', '::1', '::ffff:127.0.0.1'].includes(req.socket?.remoteAddress ?? '');
}
function authorized(req: IncomingMessage, config: Config): boolean {
  return config.faultControl.enabled && loopback(req) && Boolean(config.server.adminSecret) && req.headers['x-playwright-admin-secret'] === config.server.adminSecret;
}
function reject(res: ServerResponse): void { sendJson(res, 403, { error: { code: 'FAULT_CONTROL_FORBIDDEN', message: 'development fault control requires loopback administrative authorization' } }); }

export async function handleFaultArm(req: IncomingMessage, res: ServerResponse, config: Config, controller: FaultController): Promise<void> {
  if (!authorized(req, config)) return reject(res);
  try { sendJson(res, 200, { fault: controller.arm(await parseJsonBody(req, config) as unknown as ArmFaultRequest) }); }
  catch (error) { sendJson(res, 400, { error: { code: 'INVALID_FAULT_REQUEST', message: error instanceof Error ? error.message : 'invalid fault request' } }); }
}
export function handleFaultSnapshot(req: IncomingMessage, res: ServerResponse, config: Config, controller: FaultController): void {
  if (!authorized(req, config)) return reject(res);
  sendJson(res, 200, { faults: controller.snapshot(), audit: controller.auditEvents() });
}
export async function handleFaultDisarm(req: IncomingMessage, res: ServerResponse, config: Config, controller: FaultController): Promise<void> {
  if (!authorized(req, config)) return reject(res);
  const body = await parseJsonBody(req, config); const token = typeof body.token === 'string' ? body.token : '';
  if (!token) return sendJson(res, 400, { error: { code: 'INVALID_FAULT_REQUEST', message: 'token is required' } });
  sendJson(res, 200, { disarmed: controller.disarm(token) });
}
