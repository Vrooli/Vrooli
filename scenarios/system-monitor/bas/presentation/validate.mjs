import fs from 'node:fs';
import { fromJson } from '../../ui/node_modules/@bufbuild/protobuf/dist/esm/index.js';
import * as metrics from '../../../../packages/proto/gen/typescript/system-monitor/v1/metrics/metrics_pb.ts';
import * as device from '../../../../packages/proto/gen/typescript/system-monitor/v1/devicegraph/devicegraph_pb.ts';
import * as investigation from '../../../../packages/proto/gen/typescript/system-monitor/v1/investigations/investigations_pb.ts';
import * as settings from '../../../../packages/proto/gen/typescript/system-monitor/v1/settings/settings_pb.ts';
import * as reports from '../../../../packages/proto/gen/typescript/system-monitor/v1/reports/reports_pb.ts';
import * as scripts from '../../../../packages/proto/gen/typescript/system-monitor/v1/scripts/scripts_pb.ts';
const schemas={...metrics,...device,...investigation,...settings,...reports,...scripts};
const input=JSON.parse(fs.readFileSync(process.argv[2],'utf8'));
let checked=0;const failures=[];
for(const r of input){
 if(!r.path.includes('Service/'))continue;
 const name=r.path.split('/').pop()+'ResponseSchema';
 if(!schemas[name]){failures.push('missing '+name);continue;}
 try{fromJson(schemas[name],r.response);checked++;}catch(e){failures.push(name+': '+e.message);}
}
console.log(JSON.stringify({checked,failures,status:failures.length?'fail':'pass',method:'strict fromJson with current generated schemas; unknown fields rejected'},null,2));
process.exitCode=failures.length?1:0;
