import { act,fireEvent,screen,within } from '@testing-library/react';
import { afterEach,beforeEach,describe,expect,it,vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils';
import { previewProps,previewModel } from '../../../test-utils/filePreview';
import { DatabasePreview } from '../renderers/DatabasePreview';
import { useDatabasePreview } from '../useDatabasePreview';
import { strings } from '../../../consts/strings';
import type { DatabasePage,DatabaseQuery } from '../databaseData';
vi.mock('../useDatabasePreview');
const browse=vi.fn();
const query:DatabaseQuery={table:'tasks',page:0,filterColumn:'',filter:'',sortColumn:'',descending:false};
const page=(q=query):DatabasePage=>({query:q,page:q.page,hasMore:true,schema:{columns:[{name:'id',type:'INTEGER',primaryKey:1,required:false,defaultValue:null,generated:false},{name:'title',type:'TEXT',primaryKey:0,required:true,defaultValue:null,generated:false}],indexes:[],foreignKeys:[]},rows:[[{type:'integer',value:'1',bytes:1,truncated:false},{type:'text',value:'Professional preview',bytes:20,truncated:false}]]});
const state=()=>({info:{tables:['tasks','projects'],snapshotTime:'2026-10-18T12:00:00Z',bytes:100},page:page(),error:null,queryError:null,busy:false,browse});
const props=()=>previewProps({model:previewModel({kind:'database',basename:'atlas.sqlite'})});
beforeEach(()=>{vi.useFakeTimers();vi.clearAllMocks();vi.mocked(useDatabasePreview).mockReturnValue(state());});
afterEach(()=>{vi.useRealTimers();});
describe('SQLite drawer experience',()=>{
 it('uses an unframed mobile record list with local Back and retains the drawer',()=>{
  const onNavigate=vi.fn();render(<DatabasePreview {...props()} onNavigate={onNavigate}/>);
  const records=screen.getByTestId('database-records');expect(records.className).not.toMatch(/rounded|border/);
  fireEvent.click(screen.getByText('Professional preview'));expect(screen.getByRole('button',{name:strings.databasePreview.back})).toHaveFocus();fireEvent.click(screen.getByRole('button',{name:strings.databasePreview.back}));expect(screen.getByTestId('database-records')).toBeInTheDocument();expect(onNavigate).not.toHaveBeenCalled();
 });
 it('shows bounded cell details and schema',()=>{
  render(<DatabasePreview {...props()}/>);fireEvent.click(screen.getByText('Professional preview'));fireEvent.click(screen.getByRole('button',{name:'Professional preview'}));expect(screen.getByRole('dialog',{name:strings.databasePreview.cell})).toHaveTextContent('Professional preview');fireEvent.click(within(screen.getByRole('dialog',{name:strings.databasePreview.cell})).getByRole('button',{name:strings.inspectionPreview.closeOptions}));
  fireEvent.click(screen.getByRole('button',{name:strings.databasePreview.schema}));expect(screen.getByTestId('database-schema')).toHaveTextContent('INTEGER');expect(screen.getByText(strings.databasePreview.indexes)).toBeInTheDocument();
 });
 it('targets filters to a column and resets pagination when filtering',()=>{
  render(<DatabasePreview {...props()}/>);fireEvent.change(screen.getByRole('combobox',{name:strings.databasePreview.filterColumn}),{target:{value:'title'}});fireEvent.change(screen.getByRole('searchbox',{name:strings.databasePreview.filter}),{target:{value:'preview'}});act(()=>{vi.advanceTimersByTime(200);});expect(browse).toHaveBeenLastCalledWith({...query,filterColumn:'title',filter:'preview'});
 });
 it('resets table-specific state when selecting another table',()=>{
  render(<DatabasePreview {...props()}/>);fireEvent.click(screen.getByText('Professional preview'));fireEvent.change(screen.getByRole('combobox',{name:strings.databasePreview.tables}),{target:{value:'projects'}});act(()=>{vi.advanceTimersByTime(200);});expect(browse).toHaveBeenLastCalledWith({...query,table:'projects'});expect(screen.queryByRole('button',{name:strings.databasePreview.back})).not.toBeInTheDocument();
 });
 it('requests fresh source handles and preserves download on failure',()=>{
  vi.mocked(useDatabasePreview).mockReturnValue({...state(),info:null,page:null,error:'invalid'});const onReopen=vi.fn();render(<DatabasePreview {...props()} onReopen={onReopen}/>);expect(screen.getByText(strings.databasePreview.invalid)).toBeInTheDocument();fireEvent.click(screen.getByRole('button',{name:strings.messagesFileViewer.reopen}));expect(onReopen).toHaveBeenCalledOnce();expect(screen.getByTestId('file-preview-download')).toHaveAttribute('download','atlas.sqlite');
 });
 it('provides a complete empty-table state',()=>{const value=state();value.page={...page(),rows:[],hasMore:false};vi.mocked(useDatabasePreview).mockReturnValue(value);render(<DatabasePreview {...props()}/>);expect(screen.getByText(strings.databasePreview.empty)).toBeInTheDocument();expect(screen.getByRole('button',{name:strings.databasePreview.next})).toBeDisabled();});
 it('does not present views as empty ordinary tables',()=>{vi.mocked(useDatabasePreview).mockReturnValue({...state(),info:{tables:[],bytes:100,snapshotTime:'2026-10-18T12:00:00Z'},page:null});render(<DatabasePreview {...props()}/>);expect(screen.getByText(strings.databasePreview.noTables)).toBeInTheDocument();});
});
