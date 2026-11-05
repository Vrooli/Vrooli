import type { Database, Sqlite3Static, SqlValue } from "@sqlite.org/sqlite-wasm";
import { DATABASE_MAX_PAGE, DATABASE_PAGE_SIZE, inspectDatabaseHeader, quoteIdentifier, type DatabaseCell, type DatabasePage, type DatabaseQuery, type DatabaseSchema } from "./databaseData";
function scalarText(value: SqlValue | undefined) {
  if (value === null || value === undefined)
    return "";
  if (typeof value === "string" || typeof value === "number" || typeof value === "bigint")
    return String(value);
  throw new Error("invalid");
}
// Only internally generated SELECTs and metadata PRAGMAs reach this connection.
// Schema SQL is never executed as a statement; views and virtual tables are omitted.
export class DatabaseEngine {
  private db: Database;
  private schemas = new Map<string, DatabaseSchema>();
  readonly tables: string[];
  private deadline = 0;
  constructor(private sqlite: Sqlite3Static, bytes: ArrayBuffer) {
    inspectDatabaseHeader(bytes);
    const { capi: c, wasm, oo1 } = sqlite;
    this.db = new oo1.DB(":memory:", "c");
    let allocated = 0, owned = false;
    try {
      c.sqlite3_db_config(this.db, c.SQLITE_DBCONFIG_TRUSTED_SCHEMA, 0);
      c.sqlite3_db_config(this.db, c.SQLITE_DBCONFIG_DEFENSIVE, 1);
      c.sqlite3_db_config(this.db, c.SQLITE_DBCONFIG_ENABLE_VIEW, 0);
      c.sqlite3_db_config(this.db, c.SQLITE_DBCONFIG_ENABLE_TRIGGER, 0);
      c.sqlite3_db_config(this.db, c.SQLITE_DBCONFIG_ENABLE_LOAD_EXTENSION, 0);
      c.sqlite3_limit(this.db, c.SQLITE_LIMIT_LENGTH, 1024 * 1024);
      c.sqlite3_limit(this.db, c.SQLITE_LIMIT_SQL_LENGTH, 65536);
      c.sqlite3_limit(this.db, c.SQLITE_LIMIT_COLUMN, 512);
      c.sqlite3_limit(this.db, c.SQLITE_LIMIT_EXPR_DEPTH, 32);
      c.sqlite3_limit(this.db, c.SQLITE_LIMIT_ATTACHED, 0);
      c.sqlite3_limit(this.db, c.SQLITE_LIMIT_VDBE_OP, 100000);
      c.sqlite3_limit(this.db, c.SQLITE_LIMIT_WORKER_THREADS, 0);
      // SQLite recommends 20 trailing bytes when deserializing untrusted
      // images. They are capacity only, never part of the logical database.
      const padded = new Uint8Array(bytes.byteLength + 20);
      padded.set(new Uint8Array(bytes));
      allocated = wasm.allocFromTypedArray(padded);
      const rc = c.sqlite3_deserialize(this.db, "main", allocated, bytes.byteLength, padded.byteLength, c.SQLITE_DESERIALIZE_FREEONCLOSE | c.SQLITE_DESERIALIZE_READONLY);
      // FREEONCLOSE consumes the allocation even when deserialize fails.
      owned = true;
      if (rc !== c.SQLITE_OK)
        throw new Error("invalid");
      c.sqlite3_progress_handler(this.db, 1000, () => Date.now() > this.deadline ? 1 : 0, 0);
      this.deadline = Date.now() + 2000;
      this.db.exec("PRAGMA query_only=ON; PRAGMA trusted_schema=OFF; PRAGMA temp_store=MEMORY; PRAGMA cache_size=-1024");
      this.tables = this.read("SELECT name FROM pragma_table_list WHERE schema='main' AND type='table' AND name NOT GLOB 'sqlite_*' ORDER BY name LIMIT 257", []).map(r => scalarText(r[0]));
      if (this.tables.length > 256 || this.tables.some(n => n.length > 1024))
        throw new Error("limit");
    }
    catch (error) {
      this.db.close();
      if (allocated && !owned)
        wasm.dealloc(allocated);
      throw error;
    }
  }
  close() {
    this.db.close();
  }
  private read(sql: string, bind: SqlValue[], max = 1024, consume?: (row: SqlValue[]) => void): SqlValue[][] {
    const stmt = this.db.prepare(sql), rows: SqlValue[][] = [];
    let count = 0;
    try {
      if (bind.length)
        stmt.bind(bind);
      while (stmt.step()) {
        if (count++ >= max)
          throw new Error("limit");
        const row: SqlValue[] = [];
        for (let i = 0;i < stmt.columnCount;i++)
          row.push(stmt.get(i));
        if (consume)
          consume(row);
        else
          rows.push(row);
      }
    }
    finally {
      stmt.finalize();
    }
    return rows;
  }
  private schema(table: string): DatabaseSchema {
    const cached = this.schemas.get(table);
    if (cached)
      return cached;
    const columns = this.read('SELECT name,type,"notnull",dflt_value,pk,hidden FROM pragma_table_xinfo(?) ORDER BY cid', [table], 129).map(r => ({ name: scalarText(r[0]), type: scalarText(r[1]), required: Boolean(r[2]), defaultValue: r[3] === null ? null : scalarText(r[3]), primaryKey: Number(r[4]), generated: Number(r[5]) > 1 }));
    if (!columns.length || columns.length > 128 || columns.some(c => c.name.length > 1024 || c.type.length > 1024 || (c.defaultValue?.length ?? 0) > 65536))
      throw new Error("limit");
    const indexes = this.read('SELECT name,"unique",partial FROM pragma_index_list(?)', [table], 65).map(r => ({ name: scalarText(r[0]), unique: Boolean(r[1]), partial: Boolean(r[2]), columns: this.read('SELECT name FROM pragma_index_info(?) ORDER BY seqno', [scalarText(r[0])], 128).map(x => x[0] === null ? "(expression)" : scalarText(x[0])) }));
    if (indexes.length > 64)
      throw new Error("limit");
    const foreignKeys = this.read('SELECT "from","table","to" FROM pragma_foreign_key_list(?)', [table], 129).map(r => ({ from: scalarText(r[0]), table: scalarText(r[1]), to: r[2] === null ? "" : scalarText(r[2]) }));
    if (foreignKeys.length > 128)
      throw new Error("limit");
    const result = { columns, indexes, foreignKeys };
    if (new TextEncoder().encode(JSON.stringify(result)).length > 1024 * 1024)
      throw new Error("limit");
    this.schemas.set(table, result);
    return result;
  }
  page(query: DatabaseQuery): DatabasePage {
    this.deadline = Date.now() + 2000;
    if (!this.tables.includes(query.table) || !Number.isInteger(query.page) || query.page < 0 || query.page > DATABASE_MAX_PAGE || query.filter.length > 256)
      throw new Error("invalid");
    const schema = this.schema(query.table), names = schema.columns.map(c => c.name);
    if (query.sortColumn && !names.includes(query.sortColumn) || query.filter && !names.includes(query.filterColumn))
      throw new Error("invalid");
    const projection = names.flatMap(name => {
      const col = quoteIdentifier(name);
      return [`typeof(${col})`, `CASE WHEN typeof(${col})='blob' THEN length(${col}) ELSE length(CAST(${col} AS BLOB)) END`, `CASE WHEN typeof(${col})='blob' THEN NULL ELSE substr(CAST(${col} AS TEXT),1,16384) END`];
    });
    const pk = schema.columns.filter(c => c.primaryKey > 0).sort((a, b) => a.primaryKey - b.primaryKey).map(c => quoteIdentifier(c.name));
    const rowid = ["rowid", "_rowid_", "oid"].find(n => !names.some(x => x.toLowerCase() === n));
    const stable = pk.length ? pk : rowid ? [quoteIdentifier(rowid)] : names.map(quoteIdentifier);
    const order = [...(query.sortColumn ? [`${quoteIdentifier(query.sortColumn)} ${query.descending ? 'DESC' : 'ASC'}`] : []), ...stable].join(',');
    const where = query.filter ? ` WHERE typeof(${quoteIdentifier(query.filterColumn)})!='blob' AND instr(lower(CAST(${quoteIdentifier(query.filterColumn)} AS TEXT)),lower(?))>0` : "";
    let budget = 0, count = 0;
    const rows: DatabaseCell[][] = [];
    this.read(`SELECT ${projection.join(',')} FROM ${quoteIdentifier(query.table)}${where} ORDER BY ${order} LIMIT ? OFFSET ?`, [...(query.filter ? [query.filter] : []), DATABASE_PAGE_SIZE + 1, query.page * DATABASE_PAGE_SIZE], DATABASE_PAGE_SIZE + 1, row => {
      if (count++ >= DATABASE_PAGE_SIZE)
        return;
      rows.push(names.map((_, index): DatabaseCell => {
        const type = row[index * 3], bytes = Number(row[index * 3 + 1] ?? 0), value = scalarText(row[index * 3 + 2]);
        if (type !== "null" && type !== "text" && type !== "blob" && type !== "integer" && type !== "real")
          throw new Error("invalid");
        const valueBytes = new TextEncoder().encode(value).length;
        budget += valueBytes + 64;
        if (budget > 1024 * 1024)
          throw new Error("limit");
        return { type, bytes, value, truncated: type === "text" && valueBytes < bytes };
      }));
    });
    return { query: { ...query }, schema, rows, hasMore: count > DATABASE_PAGE_SIZE, page: query.page };
  }
}
