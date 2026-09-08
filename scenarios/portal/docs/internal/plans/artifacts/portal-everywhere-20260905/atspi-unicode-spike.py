"""Isolated mechanism spike, not production helper or native acceptance.
Run with XDG_RUNTIME_DIR matching the current user, GSETTINGS_BACKEND=memory,
GIO_USE_VFS=local: dbus-run-session -- xvfb-run -a /usr/bin/python3 <this file>
"""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import gi

gi.require_version('Gtk', '3.0')
gi.require_version('Atspi', '2.0')
from gi.repository import Gtk, GLib, Atspi

TEXT = '日本語 العربية café e\u0301 🧪'
EXPECTED = 'prefix|' + TEXT
if len(sys.argv) > 1 and sys.argv[1] == '--fixture':
    result = Path(sys.argv[2])
    window = Gtk.Window(title='Vrooli isolated Unicode fixture')
    entry = Gtk.Entry()
    entry.set_text('prefix|')
    entry.get_accessible().set_name('unicode-fixture-entry')
    def changed(widget):
        result.write_text(widget.get_text(), encoding='utf-8')
    entry.connect('changed', changed)
    window.add(entry)
    window.connect('destroy', Gtk.main_quit)
    window.show_all()
    GLib.timeout_add_seconds(15, lambda: (Gtk.main_quit(), False)[1])
    Gtk.main()
    sys.exit(0)

with tempfile.TemporaryDirectory(prefix='atspi-unicode-fixture-') as directory:
    result = Path(directory) / 'persisted.txt'
    child = subprocess.Popen([sys.executable, __file__, '--fixture', str(result)])
    try:
        deadline = time.monotonic() + 10
        target = None
        while time.monotonic() < deadline and target is None:
            desktop = Atspi.get_desktop(0)
            if desktop:
                for i in range(desktop.get_child_count()):
                    app = desktop.get_child_at_index(i)
                    if app.get_process_id() != child.pid:
                        continue
                    pending = [app]
                    visited = 0
                    while pending and visited < 32:
                        node = pending.pop(); visited += 1
                        if node.get_name() == 'unicode-fixture-entry':
                            target = node; break
                        pending.extend(node.get_child_at_index(j) for j in range(node.get_child_count()))
            if target is None: time.sleep(0.05)
        assert target is not None, 'fixture not exposed on isolated accessibility bus'
        editable = target.get_editable_text_iface()
        assert editable is not None
        assert editable.insert_text(7, TEXT, len(TEXT.encode('utf-8')))
        observed = Atspi.Text.get_text(target, 0, -1)
        assert observed == EXPECTED, repr(observed)
        assert result.read_text(encoding='utf-8') == EXPECTED
        evidence = {'mechanism': 'AT-SPI EditableText.InsertText', 'isolated_xvfb_and_bus': True,
                    'exact_fixture_pid': True, 'insert_offset_characters': 7,
                    'insert_length_bytes': len(TEXT.encode('utf-8')), 'expected': EXPECTED,
                    'accessible_text_equal': True, 'gtk_persisted_text_equal': True,
                    'production_helper_integration': False}
        Path(__file__).with_suffix('.json').write_text(json.dumps(evidence, ensure_ascii=False, indent=2)+'\n')
        print(json.dumps(evidence, ensure_ascii=False))
    finally:
        child.terminate()
        try: child.wait(timeout=3)
        except subprocess.TimeoutExpired: child.kill(); child.wait()
