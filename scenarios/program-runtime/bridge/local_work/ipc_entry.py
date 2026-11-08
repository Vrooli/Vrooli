"""Finite managed-Python entry point. Private stdin + inherited FDs3/4 only."""
import json
import importlib.util
import importlib.abc
import os
import pathlib
import sys
sys.dont_write_bytecode=True
if sys.implementation.name!='cpython' or sys.version_info[:2]!=(3,12):
    sys.stdout.write('{"http_status":503}\n')
    raise SystemExit(0)

# Only exact reviewed source modules may satisfy local_work imports. Normal
# import machinery could select timestamp-valid .pyc, a package directory or an
# extension before a pinned .py file. This loader compiles source bytes directly,
# never reads bytecode caches, and does not add a directory to global sys.path.
package_dir=pathlib.Path(__file__).resolve().parent
MODULES={'local_work':'__init__.py',**{'local_work.'+name:name+'.py'
    for name in ('adapter','projection','owners','managed','ipc')}}


class ReviewedSourceLoader(importlib.abc.Loader):
    def create_module(self,spec):return None
    def exec_module(self,module):
        source=package_dir/MODULES[module.__name__]
        with source.open('rb') as handle:raw=handle.read(1024*1024+1)
        if len(raw)>1024*1024:raise ImportError('reviewed module too large')
        module.__file__=str(source)
        module.__cached__=None
        if module.__name__=='local_work':module.__path__=[]
        exec(compile(raw,str(source),'exec',dont_inherit=True),module.__dict__)


class ReviewedSourceFinder(importlib.abc.MetaPathFinder):
    def find_spec(self,fullname,path=None,target=None):
        if fullname=='local_work' or fullname.startswith('local_work.'):
            if fullname not in MODULES:raise ImportError('unreviewed local module')
            return importlib.util.spec_from_loader(fullname,ReviewedSourceLoader(),
                is_package=fullname=='local_work')
        return None


sys.meta_path.insert(0,ReviewedSourceFinder())
from local_work.ipc import MAX_INPUT,serve_packet,parse_packet,Hold


def main():
    result={'http_status':503}
    try:
        raw=sys.stdin.buffer.read(MAX_INPUT+1)
        if len(raw)>MAX_INPUT:
            result={'http_status':413}
        elif parse_packet(raw)[1] is None:
            result={'http_status':404}
        else:
            # Go parent creates both bounded private pipes. No filesystem path,
            # socket or ambient credential source is opened for secret access.
            with os.fdopen(3,'wb',buffering=0,closefd=False) as requests,os.fdopen(4,'rb',buffering=0,closefd=False) as responses:
                result=serve_packet(raw,requests,responses)
    except Hold:
        result={'http_status':400}
    except Exception:
        result={'http_status':503}
    sys.stdout.write(json.dumps(result,separators=(',',':'))+'\n')
    sys.stdout.flush()


if __name__=='__main__':main()
