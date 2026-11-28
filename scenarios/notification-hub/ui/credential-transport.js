// A local socket peer is trusted ingress (the Cloudflare connector or a local
// browser). Forwarded headers are deliberately never treated as transport proof.
export function allowCredentialTransport(path, socket) {
  let decoded
  try { decoded = decodeURIComponent(path) } catch { return false }
  // Do not let an encoded method name pass this gate and then decode at Go.
  if (decoded !== path && (decoded.startsWith('/vrooli.notification_hub.v1.') || decoded.startsWith('/api/v1/identity/'))) return false
  if (path !== '/api/v1/identity/root-decision-read' && path !== '/api/v1/identity/root-enrollment' && path !== '/vrooli.notification_hub.v1.identity.IdentityService/Login' && path !== '/vrooli.notification_hub.v1.conversations.ConversationsService/AnswerRootWithPassword') return true
  if (socket.encrypted === true) return true
  const peer = socket.remoteAddress || ''
  return peer === '::1' || /^127(?:\.\d{1,3}){3}$/.test(peer) || /^::ffff:127(?:\.\d{1,3}){3}$/i.test(peer)
}
