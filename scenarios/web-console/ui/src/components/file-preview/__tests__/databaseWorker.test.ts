import { afterEach,beforeEach,describe,expect,it,vi } from "vitest";
import sqlite3InitModule from '@sqlite.org/sqlite-wasm';
import { DatabaseEngine } from '../databaseEngine';
import type { DatabaseRequest } from '../databaseData';
vi.mock('@sqlite.org/sqlite-wasm',()=>({default:vi.fn()}));vi.mock('../databaseEngine');
const postMessage=vi.fn(),fetch=vi.fn(),page=vi.fn(),close=vi.fn();
let receive:(event:MessageEvent<DatabaseRequest>)=>Promise<void>;
beforeEach(async()=>{vi.resetModules();vi.clearAllMocks();vi.stubGlobal('self',{postMessage});vi.stubGlobal('fetch',fetch);vi.mocked(sqlite3InitModule).mockResolvedValue({} as never);vi.mocked(DatabaseEngine).mockImplementation(()=>({tables:['tasks'],page,close}) as never);await import('../database.worker');receive=(self as unknown as {onmessage:typeof receive}).onmessage;});
afterEach(()=>{vi.unstubAllGlobals();});
const open=()=>receive(new MessageEvent('message',{data:{type:'open',href:'/snapshot'}}));
describe('database worker transport',()=>{
 it.each([[413,'limit'],[409,'stale'],[404,'stale'],[410,'stale'],[503,'busy'],[504,'timeout'],[422,'invalid'],[500,'fetch']])('surfaces HTTP %s as %s',async(status,error)=>{fetch.mockResolvedValue(new Response('failure',{status}));await open();expect(postMessage).toHaveBeenLastCalledWith({type:'error',error});expect(sqlite3InitModule).not.toHaveBeenCalled();});
 it('loads a bounded snapshot then keeps page work inside the worker',async()=>{fetch.mockResolvedValue(new Response(new Uint8Array(100),{headers:{'X-Preview-Snapshot-Time':'2026-10-18T12:00:00Z'}}));await open();expect(postMessage).toHaveBeenLastCalledWith({type:'ready',info:{tables:['tasks'],bytes:100,snapshotTime:'2026-10-18T12:00:00Z'}});await receive(new MessageEvent('message',{data:{type:'page',id:2,query:{table:'tasks',page:0,filter:'',filterColumn:'',sortColumn:'',descending:false}}}));expect(page).toHaveBeenCalledOnce();expect(fetch).toHaveBeenCalledOnce();});
 it('reports bounded query failures with their request identity',async()=>{fetch.mockResolvedValue(new Response(new Uint8Array(100)));await open();page.mockImplementationOnce(()=>{throw Error('interrupted');});await receive(new MessageEvent('message',{data:{type:'page',id:3,query:{table:'tasks',page:0,filter:'',filterColumn:'',sortColumn:'',descending:false}}}));expect(postMessage).toHaveBeenLastCalledWith({type:'error',id:3,error:'timeout'});});
});
