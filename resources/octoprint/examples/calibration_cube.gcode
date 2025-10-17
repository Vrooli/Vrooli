; Calibration Cube G-code
; 20mm x 20mm x 20mm test cube for printer calibration
; Generated for demonstration purposes

; Start G-code
G28 ; Home all axes
G1 Z15.0 F9000 ; Move Z up
G92 E0 ; Reset extruder
G1 F200 E3 ; Prime extruder
G92 E0 ; Reset extruder

; Layer 0 - First layer at 0.2mm
G0 F9000 X80 Y80 Z0.2
G1 F1500 E5 ; Prime

; Draw square base
G1 X100 Y80 E10 F1200
G1 X100 Y100 E15
G1 X80 Y100 E20
G1 X80 Y80 E25

; Layer 1 at 0.4mm
G0 Z0.4
G1 X100 Y80 E30
G1 X100 Y100 E35
G1 X80 Y100 E40
G1 X80 Y80 E45

; Continue layers (simplified for demo)
G0 Z0.6
G1 X100 Y80 E50
G1 X100 Y100 E55
G1 X80 Y100 E60
G1 X80 Y80 E65

; Final layer at 20mm
G0 Z20.0
G1 X100 Y80 E200
G1 X100 Y100 E205
G1 X80 Y100 E210
G1 X80 Y80 E215

; End G-code
G91 ; Relative positioning
G1 E-2 F2700 ; Retract
G1 E-2 Z0.2 F2400 ; Retract and raise Z
G1 X5 Y5 F3000 ; Wipe
G1 Z10 ; Raise Z more
G90 ; Absolute positioning
G1 X0 Y200 ; Present print
M106 S0 ; Turn off fan
M104 S0 ; Turn off hotend
M140 S0 ; Turn off bed
M84 ; Disable motors

; Print statistics
; Total print time: ~15 minutes
; Filament used: ~2.5m
; Layer height: 0.2mm
; Print speed: 50mm/s