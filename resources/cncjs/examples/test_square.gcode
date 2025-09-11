; Test Square - 50mm x 50mm
; Simple G-code for testing CNC movement
; Safe for dry runs (no cutting)

; Initialize
G21         ; Set units to millimeters
G90         ; Absolute positioning
G17         ; XY plane selection
G94         ; Feed rate mode (units per minute)

; Home position
G28         ; Home all axes
G92 X0 Y0 Z0 ; Set current position as origin

; Move to start position
G0 Z5       ; Raise Z to safe height
G0 X0 Y0    ; Move to origin
F300        ; Set feed rate to 300mm/min

; Draw 50mm square
G1 X50 Y0   ; Move to first corner
G1 X50 Y50  ; Move to second corner
G1 X0 Y50   ; Move to third corner
G1 X0 Y0    ; Return to origin

; Return to home
G0 Z10      ; Raise Z to safe height
G28         ; Home all axes

; Program end
M2          ; End program