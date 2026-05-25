#!/usr/bin/env node
// Tiny fake sidecar used by the Go supervisor tests. Speaks the
// line-delimited JSON IPC protocol described in plan §8.4.
//
// Env knobs (for tests):
//   KILL_AFTER_N=<n>      exit(0) after processing N messages (after replying)
//   IGNORE_HEARTBEAT=1    swallow heartbeat requests without replying
//   SLOW_EXTRACT_MS=<n>   delay extract reply by N ms (for cancel tests)
//   PROTOCOL_VERSION=<n>  override handshake reply protocol_version
//
// Stderr is fine for noise; stdout is framed JSON only.

const readline = require('readline');

const killAfter = parseInt(process.env.KILL_AFTER_N || '0', 10);
const ignoreHeartbeat = process.env.IGNORE_HEARTBEAT === '1';
const slowExtractMs = parseInt(process.env.SLOW_EXTRACT_MS || '0', 10);
const protoVersion = parseInt(process.env.PROTOCOL_VERSION || '1', 10);

let processed = 0;

function send(obj) {
  process.stdout.write(JSON.stringify(obj) + '\n');
}

function maybeExit() {
  processed++;
  if (killAfter > 0 && processed >= killAfter) {
    process.stderr.write('fake_sidecar: exiting after ' + processed + ' messages\n');
    process.exit(0);
  }
}

const rl = readline.createInterface({ input: process.stdin });

rl.on('line', (line) => {
  if (!line) return;
  let msg;
  try {
    msg = JSON.parse(line);
  } catch (e) {
    process.stderr.write('fake_sidecar: bad json: ' + e.message + '\n');
    return;
  }
  switch (msg.type) {
    case 'handshake':
      send({
        type: 'handshake',
        request_id: msg.request_id,
        protocol_version: protoVersion,
        sidecar_version: '0.0.0-fake',
      });
      maybeExit();
      break;
    case 'heartbeat':
      if (ignoreHeartbeat) {
        process.stderr.write('fake_sidecar: ignoring heartbeat\n');
        return;
      }
      send({ type: 'heartbeat', request_id: msg.request_id });
      maybeExit();
      break;
    case 'extract':
      if (slowExtractMs > 0) {
        setTimeout(() => {
          send({
            type: 'extract',
            request_id: msg.request_id,
            graph: { nodes: [], edges: [] },
            warnings: [],
            extraction_ms: slowExtractMs,
            graph_hash: 'deadbeef',
          });
          maybeExit();
        }, slowExtractMs);
      } else {
        send({
          type: 'extract',
          request_id: msg.request_id,
          graph: { nodes: [], edges: [] },
          warnings: [],
          extraction_ms: 0,
          graph_hash: 'deadbeef',
        });
        maybeExit();
      }
      break;
    case 'rewrite_apply':
      send({
        type: 'rewrite_apply',
        request_id: msg.request_id,
        results: (msg.operations || []).map(() => ({
          status: 'OPERATION_STATUS_OK',
          message: '',
        })),
      });
      maybeExit();
      break;
    case 'cancel':
      // Best-effort: swallow.
      process.stderr.write('fake_sidecar: cancel request_id=' + msg.request_id + '\n');
      break;
    case 'shutdown':
      process.stderr.write('fake_sidecar: shutdown\n');
      process.exit(0);
      break;
    default:
      send({
        type: 'error',
        request_id: msg.request_id,
        kind: 'internal',
        message: 'unknown type ' + msg.type,
      });
  }
});

rl.on('close', () => {
  process.exit(0);
});
