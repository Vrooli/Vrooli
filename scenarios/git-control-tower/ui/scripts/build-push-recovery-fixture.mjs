// Optional real-browser fixture for the Go HTTP integration test. Never served
// by the application. Only authentication/client routing are substituted.
import { build } from "vite";
import { fileURLToPath } from "node:url";
import path from "node:path";
import fs from "node:fs/promises";

const root = fileURLToPath(new URL("..", import.meta.url));
const outDir = process.argv[2];
if (!outDir || !path.isAbsolute(outDir) || !outDir.startsWith("/tmp/")) {
  throw new Error("Supply a newly created absolute fixture directory under /tmp");
}
if ((await fs.readdir(outDir)).length) throw new Error("Fixture output must be empty");
const entry = "virtual:recovery-browser.tsx";
const auth = "virtual:recovery-auth.ts";
const transport = "virtual:recovery-transport.ts";
await build({
  root, configFile: false, define: {"process.env.NODE_ENV": JSON.stringify("production")},
  esbuild: { jsx: "automatic" },
  plugins: [{
    name: "isolated-recovery-fixture", enforce: "pre",
    resolveId(id, importer) {
      if (id.endsWith(entry)) return `\0${entry}`;
      if ([entry, auth, transport].includes(id)) return `\0${id}`;
      if (importer?.endsWith("/lib/api-push-safety.ts")) {
        if (id === "./api-core") return `\0${auth}`;
        if (id === "./connect") return `\0${transport}`;
      }
    },
    load(id) {
      if (id === `\0${entry}`) return `
        import React, {useState} from "react";
        import {createRoot} from "react-dom/client";
        import {PushSafetyDialog} from ${JSON.stringify(path.join(root, "src/components/PushSafetyDialog.tsx"))};
        import ${JSON.stringify(path.join(root, "src/styles.css"))};
        function Fixture(){const [open,setOpen]=useState(true); return React.createElement(React.Fragment,null,
          React.createElement("button",{id:"reopen",onClick:()=>setOpen(true)},"Reopen recovery"),
          open && React.createElement(PushSafetyDialog,{onClose:()=>setOpen(false),onPush:()=>{throw new Error("Publication is forbidden in this fixture")}}));
        };createRoot(document.getElementById("root")).render(React.createElement(Fixture));
        const until = predicate => new Promise((resolve,reject)=>{
          const observer=new MutationObserver(check);const timeout=setTimeout(()=>{observer.disconnect();reject(new Error("Browser fixture condition timed out"))},20000);
          function check(){const value=predicate();if(value){clearTimeout(timeout);observer.disconnect();resolve(value)}}
          observer.observe(document.body,{childList:true,subtree:true,attributes:true});check();
        });
        const button = text => Array.from(document.querySelectorAll("button")).find(b=>b.textContent===text);
        (async()=>{
          const checkbox=await until(()=>document.querySelector("input[type=checkbox]"));
          if(!button("Prepare isolated recovery").disabled)throw new Error("Consent was not required");
          checkbox.click();await until(()=>!button("Prepare isolated recovery").disabled);button("Prepare isolated recovery").click();
          await until(()=>document.body.textContent.includes("Prepared artifacts checked"));
          button("Close").click();await until(()=>!document.querySelector("[role=dialog]"));
          await fetch("/test/remote-moved",{method:"POST"});button("Reopen recovery").click();
          await until(()=>button("Check preparation status") && !button("Check preparation status").disabled);
          button("Check preparation status").click();
          await until(()=>document.body.textContent.includes("Recovery state: stale"));
          await fetch("/test/browser-passed",{method:"POST"});document.body.dataset.recoveryCheck="passed";
        })().catch(error=>{document.body.dataset.recoveryCheck="failed";console.error(error);});`;
      if (id === `\0${transport}`) return `
        import {createClient} from "@connectrpc/connect";
        import {createConnectTransport} from "@connectrpc/connect-web";
        import {RepoService} from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
        export const repoClient=createClient(RepoService,createConnectTransport({baseUrl:location.origin,
          interceptors:[next=>async req=>{req.header.set("X-Test-Principal","human");return next(req)}]}));`;
      if (id === `\0${auth}`) return `
        async function post(path,body){const r=await fetch(path,{method:"POST",headers:{"Content-Type":"application/json","X-Test-Principal":"human"},body:JSON.stringify(body)});if(!r.ok)throw new Error("Fixture intent refused");return r.json()}
        export async function issueMutationIntentForOperation(operation,repo,subject){
          const p=await post("/preview",{repository_id:repo,operation,subject_context:subject});
          const i=await post("/intent",{repository_id:p.repository_id,operation,expected_revision:p.expected_revision,subject_digest:p.subject_digest,subject_context:subject});
          return {repositoryId:i.repository_id,intentId:i.intent_id};}`;
    },
  }],
  build: { outDir, emptyOutDir: false, lib: { entry, formats: ["iife"], name: "RecoveryFixture", fileName: () => "fixture.js" } },
});
await fs.writeFile(path.join(outDir,"index.html"), '<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Isolated recovery test</title><link rel="stylesheet" href="/style.css"></head><body><div id="root"></div><script src="/fixture.js"></script></body></html>');
