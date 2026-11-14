import { beforeAll, describe, expect, it, vi } from "vitest";
import { readFileSync } from "node:fs";
import sqlite3InitModule, { type Sqlite3Static } from "@sqlite.org/sqlite-wasm";
import { DatabaseEngine } from "../databaseEngine";
import { databaseCsv, databaseDetail, inspectDatabaseHeader, quoteIdentifier, type DatabaseQuery } from "../databaseData";
let sqlite:Sqlite3Static;
beforeAll(async()=>{sqlite=await sqlite3InitModule();});
const fixture=()=>{const bytes=readFileSync('../fixtures/file-preview/project-atlas.sqlite');return Uint8Array.from(bytes).buffer;};
const query=(partial:Partial<DatabaseQuery>={}):DatabaseQuery=>({table:"tasks",page:0,filterColumn:"",filter:"",sortColumn:"",descending:false,...partial});
function withEngine(run:(engine:DatabaseEngine)=>void){const engine=new DatabaseEngine(sqlite,fixture());try{run(engine);}finally{engine.close();}}
describe("SQLite inspection with the real WASM engine",()=>{
 it("lists ordinary tables and omits executable schema objects",()=>{withEngine(engine=>{expect(engine.tables).toEqual(['projects','tasks','zz_edge "cases','zz_empty_table']);});});
 it("paginates a stable snapshot and reports whether another page exists",()=>{withEngine(engine=>{const first=engine.page(query()),second=engine.page(query({page:1})),last=engine.page(query({page:2}));expect(first.rows).toHaveLength(50);expect(first.hasMore).toBe(true);expect(second.rows[0]?.[0]?.value).toBe('51');expect(last.rows).toHaveLength(23);expect(last.hasMore).toBe(false);});});
 it("filters literal text with bound values and sorts by selected columns",()=>{withEngine(engine=>{const data=engine.page(query({filterColumn:'status',filter:'complete',sortColumn:'id',descending:true}));expect(data.rows[0]?.[0]?.value).toBe('121');expect(data.rows).toHaveLength(41);expect(engine.page(query({filterColumn:'title',filter:"' OR 1=1 --"})).rows).toHaveLength(0);expect(engine.page(query({filterColumn:'title',filter:'%'})).rows).toHaveLength(0);});});
 it("preserves null, empty text, signed 64-bit integers, binary size and text truncation",()=>{withEngine(engine=>{const data=engine.page(query({table:'zz_edge "cases'})),row=data.rows[0];expect(row?.[2]).toMatchObject({type:'null',value:''});expect(row?.[3]).toMatchObject({type:'text',value:'',truncated:false});expect(row?.[4]?.value).toBe('9223372036854775807');expect(row?.[5]).toMatchObject({type:'blob',bytes:2,value:''});expect(row?.[6]).toMatchObject({type:'text',truncated:true});expect(row?.[6]?.value).toHaveLength(16384);expect(row?.[1]?.value).toContain('<script>');});});
 it("includes column defaults, indexes and foreign keys",()=>{withEngine(engine=>{const schema=engine.page(query()).schema;expect(schema.columns[0]).toMatchObject({name:'id',primaryKey:1});expect(schema.columns[2]).toMatchObject({defaultValue:"'planned'",required:true});expect(schema.indexes.find(i=>i.name==='tasks_status')?.columns).toEqual(['status']);expect(schema.foreignKeys).toContainEqual({from:'project_id',table:'projects',to:'id'});expect(engine.page(query({table:'zz_empty_table'})).rows).toHaveLength(0);});});
 it("rejects unknown identifiers, pages and oversized filters",()=>{withEngine(engine=>{for(const q of [query({table:'tasks; DROP TABLE tasks'}),query({sortColumn:'missing'}),query({filterColumn:'missing',filter:'x'}),query({page:-1}),query({page:2000}),query({page:0.5}),query({filter:'x'.repeat(257)})])expect(()=>engine.page(q)).toThrow('invalid');expect(engine.page(query()).rows).toHaveLength(50);});});
 it("enforces engine read-only state even if query_only is disabled",()=>{withEngine(engine=>{engine['db'].exec('PRAGMA query_only=OFF');expect(()=>engine['db'].exec("DELETE FROM tasks")).toThrow(/readonly/i);expect(engine.page(query()).rows).toHaveLength(50);});});
 it("escapes identifier quotes and rejects malformed, oversized or WAL-only images",()=>{expect(quoteIdentifier('a";drop')).toBe('"a"";drop"');expect(()=>{ inspectDatabaseHeader(new ArrayBuffer(100)); }).toThrow('invalid');const wal=fixture();new DataView(wal).setUint8(18,2);expect(()=>{ inspectDatabaseHeader(wal); }).toThrow('invalid');const size=fixture();new DataView(size).setUint32(28,0xffffffff);expect(()=>{ inspectDatabaseHeader(size); }).toThrow();});
 it("formats JSON details and exports only the bounded page with formula protection",()=>{withEngine(engine=>{const data=engine.page(query({table:'zz_edge "cases'}));const csv=databaseCsv(data);expect(csv).toContain("'=HYPERLINK");expect(csv.split('\r\n')).toHaveLength(2);expect(csv).toContain('[truncated]');expect(csv).toContain('9223372036854775807');const detail=engine.page(query()).rows[0]?.[5];if(!detail)throw Error('Missing cell');expect(databaseDetail(detail)).toContain('\n  "summary"');expect(databaseDetail({type:'text',value:'plain',bytes:5,truncated:false})).toBe('plain');});});
 it("rejects excessive schema cardinality and excessive result bytes",()=>{
  const db=new sqlite.oo1.DB(':memory:','c');try{
   db.exec('CREATE TABLE large(value TEXT); INSERT INTO large VALUES('+"'"+'x'.repeat(16384)+"'"+');');
   db.exec('INSERT INTO large SELECT value FROM large; INSERT INTO large SELECT value FROM large; INSERT INTO large SELECT value FROM large; INSERT INTO large SELECT value FROM large; INSERT INTO large SELECT value FROM large; INSERT INTO large SELECT value FROM large');
   const bytes=sqlite.capi.sqlite3_js_db_export(db);const engine=new DatabaseEngine(sqlite,Uint8Array.from(bytes).buffer);try{const data=engine.page(query({table:'large'}));expect(data.rows).toHaveLength(50);}finally{engine.close();}
   db.exec('ALTER TABLE large ADD COLUMN extra TEXT');db.exec("UPDATE large SET extra=value");
   const engine2=new DatabaseEngine(sqlite,Uint8Array.from(sqlite.capi.sqlite3_js_db_export(db)).buffer);try{expect(()=>engine2.page(query({table:'large'}))).toThrow('limit');}finally{engine2.close();}
  }finally{db.close();}
 });
});

describe('SQLite schema admission limits',()=>{
 it('rejects more than 256 ordinary tables',()=>{
  const db=new sqlite.oo1.DB(':memory:','c');try{
   for(let i=0;i<257;i++)db.exec(`CREATE TABLE t${String(i)}(id INTEGER)`);
   expect(()=>new DatabaseEngine(sqlite,Uint8Array.from(sqlite.capi.sqlite3_js_db_export(db)).buffer)).toThrow('limit');
  }finally{db.close();}
 });
 it('rejects tables with more than 128 columns without losing other tables',()=>{
  const db=new sqlite.oo1.DB(':memory:','c');try{
   db.exec(`CREATE TABLE too_wide(${Array.from({length:129},(_,i)=>`c${String(i)} TEXT`).join(',')})`);
   db.exec('CREATE TABLE small(id INTEGER); CREATE VIEW hostile_view AS SELECT load_extension(\'/unavailable\')');
   const engine=new DatabaseEngine(sqlite,Uint8Array.from(sqlite.capi.sqlite3_js_db_export(db)).buffer);try{
    expect(()=>engine.page(query({table:'too_wide'}))).toThrow('limit');
    expect(()=>engine.page(query({table:'hostile_view'}))).toThrow('invalid');
    expect(engine.page(query({table:'small'})).rows).toHaveLength(0);
   }finally{engine.close();}
  }finally{db.close();}
 });
});

describe('SQLite image allocation ownership',()=>{
 it('pads untrusted images and transfers allocation ownership on failure too',()=>{
  const original=sqlite.capi.sqlite3_deserialize;let imagePointer=0;
  const free=vi.spyOn(sqlite.wasm,'dealloc');
  const deserialize=vi.spyOn(sqlite.capi,'sqlite3_deserialize').mockImplementation((db,schema,pointer,size,capacity,flags)=>{
   imagePointer=pointer;expect(capacity).toBe(size+20);
   expect(flags&sqlite.capi.SQLITE_DESERIALIZE_READONLY).toBeTruthy();
   // Model the documented C failure contract: FREEONCLOSE frees P before return.
   sqlite.wasm.dealloc(pointer);
   return sqlite.capi.SQLITE_NOMEM;
  });
  try{expect(()=>new DatabaseEngine(sqlite,fixture())).toThrow('invalid');expect(imagePointer).toBeGreaterThan(0);expect(free.mock.calls.filter(([pointer])=>pointer===imagePointer)).toHaveLength(1);}finally{deserialize.mockRestore();free.mockRestore();}
  expect(sqlite.capi.sqlite3_deserialize).toBe(original);
  withEngine(engine=>{expect(engine.page(query()).rows).toHaveLength(50);});
 });
});

describe('SQLite user table names',()=>{
 it('includes user names beginning with sqlite without the reserved underscore prefix',()=>{
  const db=new sqlite.oo1.DB(':memory:','c');try{
   db.exec('CREATE TABLE sqliteReports(id INTEGER)');
   const engine=new DatabaseEngine(sqlite,Uint8Array.from(sqlite.capi.sqlite3_js_db_export(db)).buffer);
   try{expect(engine.tables).toEqual(['sqliteReports']);}finally{engine.close();}
  }finally{db.close();}
 });
});

describe('SQLite large binary cells',()=>{
 it('reports large blob length without materializing its payload',()=>{
  const db=new sqlite.oo1.DB(':memory:','c');try{
   db.exec('CREATE TABLE files(id INTEGER PRIMARY KEY, payload BLOB); INSERT INTO files VALUES(1,zeroblob(2097152))');
   const engine=new DatabaseEngine(sqlite,Uint8Array.from(sqlite.capi.sqlite3_js_db_export(db)).buffer);
   try{expect(engine.page(query({table:'files'})).rows[0]?.[1]).toEqual({type:'blob',bytes:2097152,value:'',truncated:false});}finally{engine.close();}
  }finally{db.close();}
 });
});
