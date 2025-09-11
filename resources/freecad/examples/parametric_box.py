#!/usr/bin/env python3
"""
FreeCAD Example: Parametric Box with Holes
Demonstrates parametric modeling and boolean operations
"""

import FreeCAD
import Part

# Create a new document
doc = FreeCAD.newDocument("ParametricBox")

# Create the main box
main_box = doc.addObject("Part::Box", "MainBox")
main_box.Length = 100  # mm
main_box.Width = 60    # mm
main_box.Height = 40   # mm

# Create cylinders for holes
hole1 = doc.addObject("Part::Cylinder", "Hole1")
hole1.Radius = 5
hole1.Height = 50
hole1.Placement = FreeCAD.Placement(
    FreeCAD.Vector(25, 30, -5),  # Position
    FreeCAD.Rotation(0, 0, 0, 1)  # Rotation
)

hole2 = doc.addObject("Part::Cylinder", "Hole2")
hole2.Radius = 5
hole2.Height = 50
hole2.Placement = FreeCAD.Placement(
    FreeCAD.Vector(75, 30, -5),
    FreeCAD.Rotation(0, 0, 0, 1)
)

# Create a boolean cut to make the holes
cut1 = doc.addObject("Part::Cut", "BoxWithHole1")
cut1.Base = main_box
cut1.Tool = hole1

final_part = doc.addObject("Part::Cut", "FinalPart")
final_part.Base = cut1
final_part.Tool = hole2

# Add a chamfer
chamfer = doc.addObject("Part::Chamfer", "ChamferedPart")
chamfer.Base = final_part
# Select edges to chamfer (top edges)
chamfer.Edges = [(1, 2.0), (3, 2.0), (5, 2.0), (7, 2.0)]

# Recompute the document
doc.recompute()

# Export to STEP format
output_path = "/exports/parametric_box.step"
Part.export([chamfer], output_path)
print(f"Model exported to: {output_path}")

# Save the FreeCAD document
doc.saveAs("/projects/parametric_box.FCStd")
print("Document saved to: /projects/parametric_box.FCStd")

# Print summary
print("\nModel Summary:")
print(f"- Main box: {main_box.Length}x{main_box.Width}x{main_box.Height} mm")
print(f"- Holes: 2x Ø{hole1.Radius*2} mm")
print(f"- Chamfers: 2mm on top edges")
print(f"- Total objects in document: {len(doc.Objects)}")