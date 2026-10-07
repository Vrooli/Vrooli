import {test} from 'node:test';
import assert from 'node:assert/strict';
import {world,responseFor} from './world.mjs';
test('authored world reproduces one coherent workstation at a supplied time',()=>{
 const a=world(1800000000000,0);assert.deepEqual(a,world(1800000000000,0));
 assert.equal(a.timeline.samples.at(-1).cpu.measured,a.current.cpu.measured);
 assert.equal(a.detailed.cpuDetails.usage,a.current.cpu.measured);
 assert.equal(a.detailed.gpuDetails.summary.usedMemoryMb,a.detailed.gpuDetails.devices[0].processes.reduce((n,p)=>n+p.memoryUsedMb,0));
 assert.equal(a.detailed.networkDetails.tcpStates.total,a.current.connections.measured);
 assert.ok(a.timeline.samples.every(s=>s.cpu.measured>=0&&s.cpu.measured<=100));
});
test('unwritten execution and remote-control procedures fail closed',()=>{
 for(const name of ['TriggerInvestigation','ExecuteScript','ReconcileCapacity','DeleteReport','StopAgent'])assert.equal(responseFor('/vrooli.system_monitor.v1.scripts.ScriptsService/'+name),undefined);
});
test('only exact read procedure paths receive authored responses',()=>{
 assert.ok(responseFor('/vrooli.system_monitor.v1.metrics.MetricsService/GetCurrentMetrics'));
 for(const path of ['/GetCurrentMetrics','/wrong.Service/GetCurrentMetrics','/vrooli.system_monitor.v1.settings.SettingsService/SetMaintenanceState'])assert.equal(responseFor(path),undefined);
});
