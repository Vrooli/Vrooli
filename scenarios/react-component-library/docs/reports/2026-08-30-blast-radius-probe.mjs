import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join } from "node:path";
import { execSync } from "node:child_process";
const LIB="scenarios/react-component-library/library";
const cmp=(a,b)=>a.localeCompare(b,undefined,{numeric:true});
const assets=new Map(), byLib=new Map();
for(const kind of readdirSync(LIB,{withFileTypes:true}).filter(e=>e.isDirectory()&&e.name!==".retired").map(e=>e.name))
 for(const name of readdirSync(join(LIB,kind),{withFileTypes:true}).filter(e=>e.isDirectory()).map(e=>e.name)){
  const mp=join(LIB,kind,name,"component.json"); if(!existsSync(mp))continue;
  const m=JSON.parse(readFileSync(mp,"utf8"));
  const vr=join(LIB,kind,name,"versions");
  const versions=existsSync(vr)?readdirSync(vr,{withFileTypes:true}).filter(e=>e.isDirectory()&&/^\d+\.\d+\.\d+$/.test(e.name)).map(e=>e.name).sort(cmp):[];
  const r={kind,name,libraryId:String(m.libraryId||`react-component-library:${name}`),latest:(m.latest||"").trim(),versions};
  assets.set(name,r); byLib.set(r.libraryId,r);
}
// reverse graph over LATEST versions only (a live-graph blast radius)
const rev=new Map();
for(const[,a] of assets){ if(!a.latest||!a.versions.includes(a.latest))continue;
  const lp=join(LIB,a.kind,a.name,"versions",a.latest,"dependencies.json"); if(!existsSync(lp))continue;
  for(const d of (JSON.parse(readFileSync(lp,"utf8")).dependencies||[])){
    if(!rev.has(d.libraryId))rev.set(d.libraryId,new Set());
    rev.get(d.libraryId).add(a.libraryId);
  }}
const blast=(lib)=>{const s=new Set(),st=[lib];while(st.length){const k=st.pop();for(const p of (rev.get(k)||[])) if(!s.has(p)){s.add(p);st.push(p);}}return s.size;};
const rows=[...byLib.keys()].map(l=>({asset:l.replace("react-component-library:",""),dependents:blast(l)}))
  .sort((x,y)=>y.dependents-x.dependents);
const total=rows.length;
console.log(JSON.stringify({
  liveAssets: total,
  top10BlastRadius: rows.slice(0,10),
  medianBlastRadius: rows[Math.floor(total/2)].dependents,
  assetsWithZeroDependents: rows.filter(r=>r.dependents===0).length,
},null,2));
