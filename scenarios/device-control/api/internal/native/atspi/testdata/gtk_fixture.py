"""Disposable GTK application: Go performs all accessibility mutation."""
import json
import os
from pathlib import Path
import sys
import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk, Gio, GLib

window = Gtk.Window(title='Go AT-SPI Unicode fixture')
entry = Gtk.Entry()
entry.set_text('é|tail')
entry.get_accessible().set_name('go-unicode-entry')
entry.connect('changed', lambda widget: Path(sys.argv[1]).write_text(widget.get_text(), encoding='utf-8'))
window.add(entry)
window.connect('destroy', Gtk.main_quit)
window.show_all()

def ready():
    bus = Gio.bus_get_sync(Gio.BusType.SESSION, None)
    response = bus.call_sync('org.a11y.Bus', '/org/a11y/bus', 'org.a11y.Bus', 'GetAddress', None,
                             GLib.VariantType.new('(s)'), Gio.DBusCallFlags.NONE, 2000, None)
    print(json.dumps({'address': response.unpack()[0], 'pid': os.getpid()}), flush=True)
    return False

GLib.timeout_add(200, ready)
GLib.timeout_add_seconds(15, lambda: (Gtk.main_quit(), False)[1])
Gtk.main()
